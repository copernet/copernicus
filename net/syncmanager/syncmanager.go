// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package syncmanager

import (
	"container/list"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/peer"
	"github.com/copernet/copernicus/util"
)

const (
	// minInFlightBlocks is the minimum number of blocks that should be
	// in the request queue for headers-first mode before requesting
	// more.
	minInFlightBlocks = 10

	// maxRejectedTxns is the maximum number of rejected transactions
	// hashes to store in memory.
	maxRejectedTxns = 1000

	// maxRequestedBlocks is the maximum number of requested block
	// hashes to store in memory.
	maxRequestedBlocks = wire.MaxInvPerMsg

	// maxRequestedTxns is the maximum number of requested transactions
	// hashes to store in memory.
	maxRequestedTxns = wire.MaxInvPerMsg

	blockRequestTimeoutTime = 20 * time.Minute
)

// zeroHash is the zero value hash (all zeros).  It is defined as a convenience.
var zeroHash util.Hash

// newPeerMsg signifies a newly connected peer to the block handler.
type newPeerMsg struct {
	peer *peer.Peer
}

// blockMsg packages a bitcoin block message and the peer it came from together
// so the block handler has access to that information.
type blockMsg struct {
	block *block.Block
	buf   []byte
	peer  *peer.Peer
	reply chan<- struct{}
}

// poolMsg package a bitcoin mempool message and peer it come from together
type poolMsg struct {
	pool  *wire.MsgMemPool
	peer  *peer.Peer
	reply chan<- struct{}
}

// getdataMsg package a bitcoin getdata message And peer it come from together
type getdataMsg struct {
	getdata *wire.MsgGetData
	peer    *peer.Peer
	reply   chan<- struct{}
}

// getBlocksMsg package a bitcoin getblocks message And peer it come from together
type getBlocksMsg struct {
	getblocks *wire.MsgGetBlocks
	peer      *peer.Peer
	reply     chan<- struct{}
}

// invMsg packages a bitcoin inv message and the peer it came from together
// so the block handler has access to that information.
type invMsg struct {
	inv  *wire.MsgInv
	peer *peer.Peer
}

// headersMsg packages a bitcoin headers message and the peer it came from
// together so the block handler has access to that information.
type headersMsg struct {
	headers *wire.MsgHeaders
	peer    *peer.Peer
}

// donePeerMsg signifies a newly disconnected peer to the block handler.
type donePeerMsg struct {
	peer *peer.Peer
}

// txMsg packages a bitcoin tx message and the peer it came from together
// so the block handler has access to that information.
type txMsg struct {
	tx    *tx.Tx
	peer  *peer.Peer
	reply chan<- struct{}
}

// getSyncPeerMsg is a message type to be sent across the message channel for
// retrieving the current sync peer.
type getSyncPeerMsg struct {
	reply chan int32
}

// processBlockResponse is a response sent to the reply channel of a
// processBlockMsg.
type processBlockResponse struct {
	isOrphan bool
	err      error
}

// processBlockMsg is a message type to be sent across the message channel
// for requested a block is processed.  Note this call differs from blockMsg
// above in that blockMsg is intended for blocks that came from peers and have
// extra handling whereas this message essentially is just a concurrent safe
// way to call ProcessBlock on the internal block chain instance.
type processBlockMsg struct {
	block *block.Block
	flags chain.BehaviorFlags
	reply chan processBlockResponse
}

// isCurrentMsg is a message type to be sent across the message channel for
// requesting whether or not the sync manager believes it is synced with the
// currently connected peers.
type isCurrentMsg struct {
	reply chan bool
}

// pauseMsg is a message type to be sent across the message channel for
// pausing the sync manager.  This effectively provides the caller with
// exclusive access over the manager until a receive is performed on the
// unpause channel.
type pauseMsg struct {
	unpause <-chan struct{}
}

// headerNode is used as a node in a list of headers that are linked together
// between checkpoints.
type headerNode struct {
	height int32
	hash   *util.Hash
}

// peerSyncState stores additional information that the SyncManager tracks
// about a peer.
type peerSyncState struct {
	syncCandidate   bool
	requestQueue    []*wire.InvVect
	requestedTxns   map[util.Hash]struct{}
	requestedBlocks map[util.Hash]struct{}
}

// SyncManager is used to communicate block related messages with peers. The
// SyncManager is started as by executing Start() in a goroutine. Once started,
// it selects peers to sync from and starts the initial block download. Once the
// chain is in sync, the SyncManager handles incoming block and header
// notifications and relays announcements of new blocks to peers.
type SyncManager struct {
	peerNotifier        PeerNotifier
	started             int32
	shutdown            int32
	chainParams         *model.BitcoinParams
	progressLogger      *blockProgressLogger
	processBusinessChan chan interface{}
	wg                  sync.WaitGroup
	quit                chan struct{}

	// These fields should only be accessed from the messagesHandler
	rejectedTxns    map[util.Hash]struct{}
	requestedTxns   map[util.Hash]struct{}
	requestedBlocks map[util.Hash]struct{}
	syncPeer        *peer.Peer
	peerStates      map[*peer.Peer]*peerSyncState

	// The following fields are used for headers-first mode.
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
	nextCheckpoint   *model.Checkpoint

	// callback for transaction And block process
	ProcessTransactionCallBack func(*tx.Tx, int64) ([]*tx.Tx, []util.Hash, error)
	ProcessBlockCallBack       func(*block.Block) (bool, error)
	ProcessBlockHeadCallBack   func([]*block.BlockHeader, *blockindex.BlockIndex) error

	requestBlkInvCnt int

	// An optional fee estimator.
	feeEstimator *mempool.FeeEstimator
}

// resetHeaderState sets the headers-first mode state to values appropriate for
// syncing from a new peer.
func (sm *SyncManager) resetHeaderState(newestHash *util.Hash, newestHeight int32) {
	sm.headersFirstMode = false
	sm.headerList.Init()
	sm.startHeader = nil

	// When there is a next checkpoint, add an entry for the latest known
	// block into the header pool.  This allows the next downloaded header
	// to prove it links to the chain properly.
	if sm.nextCheckpoint != nil {
		node := headerNode{height: newestHeight, hash: newestHash}
		sm.headerList.PushBack(&node)
	}
}

// findNextHeaderCheckpoint returns the next checkpoint after the passed height.
// It returns nil when there is not one either because the height is already
// later than the final checkpoint or some other reason such as disabled
// checkpoints.
func (sm *SyncManager) findNextHeaderCheckpoint(height int32) *model.Checkpoint {
	//todo !!! need to be modified to be flexible for checkpoint with chainpram.
	checkpoints := model.ActiveNetParams.Checkpoints
	log.Trace("come into findNextHeaderCheckpoint, numbers : %d ...", len(checkpoints))
	if len(checkpoints) == 0 {
		return nil
	}

	// There is no next checkpoint if the height is already after the final
	// checkpoint.
	finalCheckpoint := checkpoints[len(checkpoints)-1]
	log.Trace("finalCheckpoint.Height : %d, current height : %d ", finalCheckpoint.Height, height)
	if height >= finalCheckpoint.Height {
		return nil
	}

	// Find the next checkpoint.
	nextCheckpoint := finalCheckpoint
	for i := len(checkpoints) - 2; i >= 0; i-- {
		if height >= checkpoints[i].Height {
			break
		}
		nextCheckpoint = checkpoints[i]
	}
	log.Trace("return checkpoint heigth : %d", nextCheckpoint.Height)
	return nextCheckpoint
}

// startSync will choose the best peer among the available candidate peers to
// download/sync the blockchain from.  When syncing is already running, it
// simply returns.  It also examines the candidates for any which are no longer
// candidates and removes them as needed.
func (sm *SyncManager) startSync() {
	// Return now if we're already syncing.
	if sm.syncPeer != nil {
		return
	}

	best := chain.GetInstance().Tip()
	var bestPeer *peer.Peer
	for peer, state := range sm.peerStates {
		if !state.syncCandidate {
			continue
		}

		// Remove sync candidate peers that are no longer candidates due
		// to passing their latest known block.  NOTE: The < is
		// intentional as opposed to <=.  While technically the peer
		// doesn't have a later block when it's equal, it will likely
		// have one soon so it is a reasonable choice.  It also allows
		// the case where both are at 0 such as during regression test.
		if peer.LastBlock() <= best.Height {
			state.syncCandidate = false
			continue
		}

		// TODO(davec): Use a better algorithm to choose the best peer.
		// For now, just pick the first available candidate.
		bestPeer = peer
	}

	// Start syncing from the best peer if one was selected.
	if bestPeer != nil {
		// Clear the requestedBlocks if the sync peer changes, otherwise
		// we may ignore blocks we need that the last sync peer failed
		// to send.
		sm.requestedBlocks = make(map[util.Hash]struct{})

		activeChain := chain.GetInstance()
		locator := activeChain.GetLocator(nil)
		log.Info("Syncing to block height %d from peer %v",
			bestPeer.LastBlock(), bestPeer.Addr())

		// When the current height is less than a known checkpoint we
		// can use block headers to learn about which blocks comprise
		// the chain up to the checkpoint and perform less validation
		// for them.  This is possible since each header contains the
		// hash of the previous header and a merkle root.  Therefore if
		// we validate all of the received headers link together
		// properly and the checkpoint hashes match, we can be sure the
		// hashes for the blocks in between are accurate.  Further, once
		// the full blocks are downloaded, the merkle root is computed
		// and compared against the value in the header which proves the
		// full block hasn't been tampered with.
		//
		// Once we have passed the final checkpoint, or checkpoints are
		// disabled, use standard inv messages learn about the blocks
		// and fully validate them.  Finally, regression test mode does
		// not support the headers-first approach so do normal block
		// downloads when in regression test mode.
		if sm.nextCheckpoint != nil &&
			best.Height < sm.nextCheckpoint.Height &&
			sm.chainParams != &model.RegressionNetParams {
			//	3. push peer
			bestPeer.PushGetHeadersMsg(*locator, sm.nextCheckpoint.Hash)
			sm.headersFirstMode = true
			log.Info("Downloading headers for blocks %d to "+
				"%d from peer %s", best.Height+1,
				sm.nextCheckpoint.Height, bestPeer.Addr())
		} else {
			log.Info("no checkpoint in syncmanager, so download block stophash is all zero...")
			bestPeer.PushGetBlocksMsg(*locator, &zeroHash)
		}
		sm.syncPeer = bestPeer
		sm.requestBlkInvCnt = 0
		if sm.current() {
			log.Debug("request mempool in startSync")
			bestPeer.RequestMemPool()
		}
	} else {
		log.Warn("No sync peer candidates available")
	}
}

// isSyncCandidate returns whether or not the peer is a candidate to consider
// syncing from.
func (sm *SyncManager) isSyncCandidate(peer *peer.Peer) bool {
	// Typically a peer is not a candidate for sync if it's not a full node,
	// however regression test is special in that the regression tool is
	// not a full node and still needs to be considered a sync candidate.
	if sm.chainParams == &model.RegressionNetParams {
		// The peer is not a candidate if it's not coming from localhost
		// or the hostname can't be determined for some reason.
		host, _, err := net.SplitHostPort(peer.Addr())
		if err != nil {
			return false
		}

		if host != "127.0.0.1" && host != "localhost" {
			return false
		}
	} else {
		// The peer is not a candidate for sync if it's not a full
		// node. Additionally, if the segwit soft-fork package has
		// activated, then the peer must also be upgraded.
		nodeServices := peer.Services()
		if nodeServices&wire.SFNodeNetwork != wire.SFNodeNetwork {
			return false
		}
	}

	// Candidate if all checks passed.
	return true
}

// handleNewPeerMsg deals with new peers that have signalled they may
// be considered as a sync peer (they have already successfully negotiated).  It
// also starts syncing if needed.  It is invoked from the syncHandler goroutine.
func (sm *SyncManager) handleNewPeerMsg(peer *peer.Peer) {
	// Ignore if in the process of shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	log.Info("New valid peer %s (%s), start height : %d", peer.Addr(), peer.UserAgent(), peer.StartingHeight())

	// Initialize the peer state
	isSyncCandidate := sm.isSyncCandidate(peer)
	sm.peerStates[peer] = &peerSyncState{
		syncCandidate:   isSyncCandidate,
		requestedTxns:   make(map[util.Hash]struct{}),
		requestedBlocks: make(map[util.Hash]struct{}),
	}

	// Start syncing by choosing the best candidate if needed.
	if isSyncCandidate && sm.syncPeer == nil {
		sm.startSync()
	}
}

// handleDonePeerMsg deals with peers that have signalled they are done.  It
// removes the peer as a candidate for syncing and in the case where it was
// the current sync peer, attempts to select a new best peer to sync from.  It
// is invoked from the syncHandler goroutine.
func (sm *SyncManager) handleDonePeerMsg(peer *peer.Peer) {
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warn("Received done peer message for unknown peer %s", peer)
		return
	}

	// Remove the peer from the list of candidate peers.
	delete(sm.peerStates, peer)

	log.Info("Lost peer %s", peer.Addr())

	// Remove requested transactions from the global map so that they will
	// be fetched from elsewhere next time we get an inv.
	for txHash := range state.requestedTxns {
		delete(sm.requestedTxns, txHash)
	}

	// Remove requested blocks from the global map so that they will be
	// fetched from elsewhere next time we get an inv.
	// TODO: we could possibly here check which peers have these blocks
	// and request them now to speed things up a little.
	for blockHash := range state.requestedBlocks {
		delete(sm.requestedBlocks, blockHash)
	}

	// Attempt to find a new peer to sync from if the quitting peer is the
	// sync peer.  Also, reset the headers-first state if in headers-first
	// mode so
	if sm.syncPeer == peer {
		sm.syncPeer = nil
		if sm.headersFirstMode {
			best := chain.GetInstance().Tip()
			sm.resetHeaderState(best.GetBlockHash(), best.Height)
		}
		sm.startSync()
	}
}

func (sm *SyncManager) alreadyHave(txHash *util.Hash) bool {
	// Ignore transactions that we have already rejected.  Do not
	// send a reject message here because if the transaction was already
	// rejected, the transaction was unsolicited.
	if _, exists := sm.rejectedTxns[*txHash]; exists {
		log.Debug("Ignoring unsolicited previously rejected transaction %v", txHash)
		return true
	}

	have, err := sm.haveInventory(wire.NewInvVect(wire.InvTypeTx, txHash))
	return err == nil && have
}

// handleTxMsg handles transaction messages from all peers.
func (sm *SyncManager) handleTxMsg(tmsg *txMsg) {
	peer := tmsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warn("Received tx message from unknown peer %s", peer.Addr())
		return
	}

	txHash := tmsg.tx.GetHash()

	if sm.alreadyHave(&txHash) {
		//TODO: relay tx for whitelistrelay node
		log.Trace("ignore already processed tx from %s", peer.Addr())
		return
	}

	// Process the transaction to include validation, insertion in the
	// memory pool, orphan handling, etc.
	acceptedTxs, missTx, err := sm.ProcessTransactionCallBack(tmsg.tx, int64(peer.ID()))
	// Remove transaction from request maps. Either the mempool/chain
	// already knows about it and as such we shouldn't have any more
	// instances of trying to fetch it, or we failed to insert and thus
	// we'll retry next time we get an inv.
	delete(state.requestedTxns, txHash)
	delete(sm.requestedTxns, txHash)
	invMsg := wire.NewMsgInvSizeHint(uint(len(missTx)))
	for _, hash := range missTx {
		iv := wire.NewInvVect(wire.InvTypeTx, &hash)
		invMsg.AddInvVect(iv)
	}
	if len(missTx) > 0 {
		peer.QueueMessage(invMsg, nil)
	}

	if err != nil {
		// Do not request this transaction again until a new block
		// has been processed.
		sm.rejectedTxns[txHash] = struct{}{}
		sm.limitMap(sm.rejectedTxns, maxRejectedTxns)

		// When the error is a rule error, it means the transaction was
		// simply rejected as opposed to something actually going wrong,
		// so log it as such.  Otherwise, something really did go wrong,
		// so log it as an actual error.
		if !errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut) {
			log.Debug("Rejected transaction %v from %s: %v",
				txHash, peer.Addr(), err)
		} else {
			log.Warn("Failed to process transaction %v: %v",
				txHash.String(), err)
		}

		// Sending BIP61 reject code; never send internal reject codes over P2P.
		if rejectCode, reason, ok := errcode.IsRejectCode(err); ok {
			peer.PushRejectMsg(wire.CmdTx, rejectCode, reason, &txHash, false)
		}
		return
	}

	txentrys := make([]*mempool.TxEntry, 0, len(acceptedTxs))
	for _, tx := range acceptedTxs {
		if entry := lmempool.FindTxInMempool(tx.GetHash()); entry != nil {
			txentrys = append(txentrys, entry)
		} else {
			panic("the transaction must be in mempool")
		}
	}

	sm.peerNotifier.AnnounceNewTransactions(txentrys)
}

// current returns true if we believe we are synced with our peers, false if we
// still have blocks to check
func (sm *SyncManager) current() bool {
	if !chain.GetInstance().IsCurrent() {
		return false
	}

	// if blockChain thinks we are current and we have no syncPeer it
	// is probably right.
	if sm.syncPeer == nil {
		return true
	}

	// No matter what chain thinks, if we are below the block we are syncing
	// to we are not current.
	if chain.GetInstance().Tip().Height < sm.syncPeer.LastBlock() {
		return false
	}
	return true
}

// handleBlockMsg handles block messages from all peers.
func (sm *SyncManager) handleBlockMsg(bmsg *blockMsg) {
	peer := bmsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warn("Received block message from unknown peer %s", peer.Addr())
		return
	}

	// If we didn't ask for this block then the peer is misbehaving.
	blockHash := bmsg.block.GetHash()
	if _, exists = state.requestedBlocks[blockHash]; !exists {
		// The regression test intentionally sends some blocks twice
		// to test duplicate block insertion fails.  Don't disconnect
		// the peer or ignore the block when we're in regression test
		// mode in this case so the chain code is actually fed the
		// duplicate blocks.
		if sm.chainParams != &model.RegressionNetParams {
			log.Warn("Got unrequested block %v from %s -- "+
				"disconnecting", blockHash, peer.Addr())
			peer.Disconnect()
			return
		}
	}

	// When in headers-first mode, if the block matches the hash of the
	// first header in the list of headers that are being fetched, it's
	// eligible for less validation since the headers have already been
	// verified to link together and are valid up to the next checkpoint.
	// Also, remove the list entry for all blocks except the checkpoint
	// since it is needed to verify the next round of headers links
	// properly.
	isCheckpointBlock := false
	if sm.headersFirstMode {
		firstNodeEl := sm.headerList.Front()
		if firstNodeEl != nil {
			firstNode := firstNodeEl.Value.(*headerNode)
			if blockHash.IsEqual(firstNode.hash) {
				if firstNode.hash.IsEqual(sm.nextCheckpoint.Hash) {
					isCheckpointBlock = true
				} else {
					sm.headerList.Remove(firstNodeEl)
				}
			}
		}
	}

	// Remove block from request maps. Either chain will know about it and
	// so we shouldn't have any more instances of trying to fetch it, or we
	// will fail the insert and thus we'll retry next time we get an inv.
	delete(state.requestedBlocks, blockHash)
	delete(sm.requestedBlocks, blockHash)

	// Process the block to include validation, best chain selection, orphan
	// handling, etc.
	_, err := sm.ProcessBlockCallBack(bmsg.block)
	if err != nil {
		// When the error is a rule error, it means the block was simply
		// rejected as opposed to something actually going wrong, so log
		// it as such.  Otherwise, something really did go wrong, so log
		// it as an actual error.
		// todo !!! and error code process. yyx
		//if _, ok := err.(blockchain.RuleError); ok {
		//	log.Info("Rejected block %v from %s: %v", blockHash,
		//		peer, err)
		//} else {
		//	log.Error("Failed to process block %v: %v",
		//		blockHash, err)
		//}
		//if dbErr, ok := err.(database.Error); ok && dbErr.ErrorCode ==
		//	database.ErrCorruption {
		//	panic(dbErr)
		//}

		// Convert the error into an appropriate reject message and
		// send it.
		// todo !!! need process. yyx
		//code, reason := mpool.ErrToRejectErr(err)
		//peer.PushRejectMsg(wire.CmdBlock, code, reason, blockHash, false)
		if !sm.headersFirstMode {
			log.Debug("len of Requested block:%d, sm.requestBlockInvCnt: %d", len(sm.requestedBlocks), sm.requestBlkInvCnt)
			if peer == sm.syncPeer && sm.requestBlkInvCnt > 0 {
				if len(state.requestedBlocks) == 0 {
					sm.requestBlkInvCnt--
					activeChain := chain.GetInstance()
					locator := activeChain.GetLocator(nil)
					peer.PushGetBlocksMsg(*locator, &zeroHash)
				}
			}
		} else {
			if !isCheckpointBlock {
				if sm.startHeader != nil &&
					len(state.requestedBlocks) < minInFlightBlocks {
					sm.fetchHeaderBlocks()
				}
			}
		}

		log.Debug("ProcessBlockCallBack err:%v", err)
		return
	}

	// Meta-data about the new block this peer is reporting. We use this
	// below to update this peer's lastest block height and the heights of
	// other peers based on their last announced block hash. This allows us
	// to dynamically update the block heights of peers, avoiding stale
	// heights when looking for a new sync peer. Upon acceptance of a block
	// or recognition of an orphan, we also use this information to update
	// the block heights over other peers who's invs may have been ignored
	// if we are actively syncing while the chain is not yet current or
	// who may have lost the lock announcment race.
	var heightUpdate int32
	var blkHashUpdate *util.Hash

	// When the block is not an orphan, log information about it and
	// update the chain state.
	sm.progressLogger.LogBlockHeight(bmsg.block)

	// Update this peer's latest block height, for future
	// potential sync node candidacy.
	best := chain.GetInstance().Tip()
	heightUpdate = best.Height
	blkHashUpdate = best.GetBlockHash()

	// Clear the rejected transactions.
	sm.rejectedTxns = make(map[util.Hash]struct{})

	// Update the block height for this peer. But only send a message to
	// the server for updating peer heights if this is an orphan or our
	// chain is "current". This avoids sending a spammy amount of messages
	// if we're syncing the chain from scratch.
	if blkHashUpdate != nil && heightUpdate != 0 {
		peer.UpdateLastBlockHeight(heightUpdate)
		if sm.current() && peer == sm.syncPeer {
			go sm.peerNotifier.UpdatePeerHeights(blkHashUpdate, heightUpdate,
				peer)
			log.Debug("request mempool in handleBlockMsg")
			peer.RequestMemPool()
		}
	}

	// Nothing more to do if we aren't in headers-first mode.
	if !sm.headersFirstMode {
		log.Debug("len of Requested block:%d, requestBlkInvCnt: %d", len(sm.requestedBlocks), sm.requestBlkInvCnt)
		if peer == sm.syncPeer && sm.requestBlkInvCnt > 0 {
			if len(state.requestedBlocks) == 0 {
				sm.requestBlkInvCnt--
				activeChain := chain.GetInstance()
				locator := activeChain.GetLocator(nil)
				peer.PushGetBlocksMsg(*locator, &zeroHash)
			}
		}
		return
	}

	// This is headers-first mode, so if the block is not a checkpoint
	// request more blocks using the header list when the request queue is
	// getting short.
	if !isCheckpointBlock {
		if sm.startHeader != nil &&
			len(state.requestedBlocks) < minInFlightBlocks {
			sm.fetchHeaderBlocks()
		}
		return
	}

	// This is headers-first mode and the block is a checkpoint.  When
	// there is a next checkpoint, get the next round of headers by asking
	// for headers starting from the block after this one up to the next
	// checkpoint.
	prevHeight := sm.nextCheckpoint.Height
	prevHash := sm.nextCheckpoint.Hash
	sm.nextCheckpoint = sm.findNextHeaderCheckpoint(prevHeight)
	if sm.nextCheckpoint != nil {
		locator := chain.NewBlockLocator([]util.Hash{*prevHash})
		err := peer.PushGetHeadersMsg(*locator, sm.nextCheckpoint.Hash)
		if err != nil {
			log.Warn("Failed to send getheaders message to "+
				"peer %s: %v", peer.Addr(), err)
			return
		}
		log.Info("Downloading headers for blocks %d to %d from "+
			"peer %s", prevHeight+1, sm.nextCheckpoint.Height,
			sm.syncPeer.Addr())
		return
	}

	// This is headers-first mode, the block is a checkpoint, and there are
	// no more checkpoints, so switch to normal mode by requesting blocks
	// from the block after this one up to the end of the chain (zero hash).
	sm.headersFirstMode = false
	sm.headerList.Init()
	log.Info("Reached the final checkpoint -- switching to normal mode")
	locator := chain.NewBlockLocator([]util.Hash{blockHash})
	err = peer.PushGetBlocksMsg(*locator, &zeroHash)
	if err != nil {
		log.Warn("Failed to send getblocks message to peer %s: %v",
			peer.Addr(), err)
		return
	}
}

// fetchHeaderBlocks creates and sends a request to the syncPeer for the next
// list of blocks to be downloaded based on the current list of headers.
func (sm *SyncManager) fetchHeaderBlocks() {
	// Nothing to do if there is no start header.
	if sm.startHeader == nil {
		log.Warn("fetchHeaderBlocks called with no start header")
		return
	}

	// Build up a getdata request for the list of blocks the headers
	// describe.  The size hint will be limited to wire.MaxInvPerMsg by
	// the function, so no need to double check it here.
	gdmsg := wire.NewMsgGetDataSizeHint(uint(sm.headerList.Len()))
	numRequested := 0

	for e := sm.startHeader; e != nil; e = e.Next() {
		node, ok := e.Value.(*headerNode)
		if !ok {
			log.Warn("Header list node type is not a headerNode")
			continue
		}

		iv := wire.NewInvVect(wire.InvTypeBlock, node.hash)
		haveInv, err := sm.haveInventory(iv)
		if err != nil {
			log.Warn("Unexpected failure when checking for "+
				"existing inventory during header block "+
				"fetch: %v", err)
		}
		if !haveInv {
			syncPeerState := sm.peerStates[sm.syncPeer]
			sm.requestedBlocks[*node.hash] = struct{}{}
			syncPeerState.requestedBlocks[*node.hash] = struct{}{}
			gdmsg.AddInvVect(iv)
			numRequested++
		}
		sm.startHeader = e.Next()
		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}

	log.Trace("ready to send getdata request, inv Number : %d", len(gdmsg.InvList))
	if len(gdmsg.InvList) > 0 {
		sm.syncPeer.QueueMessage(gdmsg, nil)
	}
	log.Trace("let getdata request to queue, ready to send to peer.")
}

// handleHeadersMsg handles block header messages from all peers.  Headers are
// requested when performing a headers-first sync.
func (sm *SyncManager) handleHeadersMsg(hmsg *headersMsg) {
	log.Trace("headers message come into syncmanager ...")
	peer := hmsg.peer
	_, exists := sm.peerStates[peer]
	if !exists {
		log.Warn("Received headers message from unknown peer %s", peer.Addr())
		return
	}

	// The remote peer is misbehaving if we didn't request headers.
	msg := hmsg.headers
	numHeaders := len(msg.Headers)
	if !sm.headersFirstMode {
		log.Warn("Got %d unrequested headers from %s -- "+
			"disconnecting", numHeaders, peer.Addr())
		peer.Disconnect()
		return
	}

	// Nothing to do for an empty headers message.
	if numHeaders == 0 {
		return
	}

	// Process all of the received headers ensuring each one connects to the
	// previous and that checkpoints match.
	receivedCheckpoint := false
	var finalHash *util.Hash
	for _, blockHeader := range msg.Headers {
		blockHash := blockHeader.GetHash()
		finalHash = &blockHash

		// Ensure there is a previous header to compare against.
		prevNodeEl := sm.headerList.Back()
		if prevNodeEl == nil {
			log.Warn("Header list does not contain a previous" +
				"element as expected -- disconnecting peer")
			peer.Disconnect()
			return
		}

		// Ensure the header properly connects to the previous one and
		// add it to the list of headers.
		node := headerNode{hash: &blockHash}
		prevNode := prevNodeEl.Value.(*headerNode)
		if prevNode.hash.IsEqual(&blockHeader.HashPrevBlock) {
			node.height = prevNode.height + 1
			e := sm.headerList.PushBack(&node)
			if sm.startHeader == nil {
				sm.startHeader = e
			}
		} else {
			log.Warn("Received block header that does not "+
				"properly connect to the chain from peer %s "+
				"-- disconnecting, expect hash : %s, actual hash : %s",
				peer.Addr(), prevNode.hash.String(), blockHash.String())
			peer.Disconnect()
			return
		}

		// Verify the header at the next checkpoint height matches.
		if node.height == sm.nextCheckpoint.Height {
			if node.hash.IsEqual(sm.nextCheckpoint.Hash) {
				receivedCheckpoint = true
				log.Info("Verified downloaded block "+
					"header against checkpoint at height "+
					"%d/hash %s", node.height, node.hash)
			} else {
				log.Warn("Block header at height %d/hash "+
					"%s from peer %s does NOT match "+
					"expected checkpoint hash of %s -- "+
					"disconnecting", node.height,
					node.hash, peer.Addr(),
					sm.nextCheckpoint.Hash)
				peer.Disconnect()
				return
			}
			break
		}
	}
	log.Trace("begin to processblockheader ...")
	var lastBlkIndex blockindex.BlockIndex
	if err := sm.ProcessBlockHeadCallBack(msg.Headers, &lastBlkIndex); err != nil {
		beginHash := msg.Headers[0].GetHash()
		endHash := msg.Headers[len(msg.Headers)-1].GetHash()
		log.Warn("processblockheader error, beginHeader hash : %s, endHeader hash : %s,"+
			"error news : %s.", beginHash.String(), endHash.String(), err.Error())
	}

	// When this header is a checkpoint, switch to fetching the blocks for
	// all of the headers since the last checkpoint.
	if receivedCheckpoint {
		// Since the first entry of the list is always the final block
		// that is already in the database and is only used to ensure
		// the next header links properly, it must be removed before
		// fetching the blocks.
		sm.headerList.Remove(sm.headerList.Front())
		log.Info("Received %v block headers: Fetching blocks",
			sm.headerList.Len())
		sm.progressLogger.SetLastLogTime(time.Now())
		sm.fetchHeaderBlocks()
		return
	}
	activeChain := chain.GetInstance()
	// This header is not a checkpoint, so request the next batch of
	// headers starting from the latest known header and ending with the
	// next checkpoint.
	blkIndex := activeChain.FindBlockIndex(*finalHash)
	locator := activeChain.GetLocator(blkIndex)
	err := peer.PushGetHeadersMsg(*locator, sm.nextCheckpoint.Hash)
	if err != nil {
		log.Warn("Failed to send getheaders message to "+
			"peer %s: %v", peer.Addr(), err)
		return
	}
}

// haveInventory returns whether or not the inventory represented by the passed
// inventory vector is known.  This includes checking all of the various places
// inventory can be when it is in different states such as blocks that are part
// of the main chain, on a side chain, in the orphan pool, and transactions that
// are in the memory pool (either the main pool or orphan pool).
func (sm *SyncManager) haveInventory(invVect *wire.InvVect) (bool, error) {
	activeChain := chain.GetInstance()
	switch invVect.Type {
	case wire.InvTypeBlock:
		// Ask chain if the block is known to it in any form (main
		// chain, side chain, or orphan).
		blkIndex := activeChain.FindBlockIndex(invVect.Hash)
		if blkIndex == nil {
			return false, nil
		}
		if blkIndex.HasData() {
			return true, nil
		}
		return true, nil

	case wire.InvTypeTx:
		// Ask the transaction memory pool if the transaction is known
		// to it in any form (main pool or orphan).
		if lmempool.FindTxInMempool(invVect.Hash) != nil {
			return true, nil
		}
		// Check if the transaction exists from the point of view of the
		// end of the main chain.
		pcoins := utxo.GetUtxoCacheInstance()
		out := outpoint.OutPoint{Hash: invVect.Hash, Index: 0}
		if pcoins.GetCoin(&out) != nil {
			return true, nil
		}
		out.Index = 1
		if pcoins.GetCoin(&out) != nil {
			return true, nil
		}
		if lmempool.FindRejectTxInMempool(invVect.Hash) ||
			lmempool.FindOrphanTxInMemPool(invVect.Hash) != nil {
			return true, nil
		}
		return false, nil
	}

	// The requested inventory is is an unsupported type, so just claim
	// it is known to avoid requesting it.
	return true, nil
}

// handleInvMsg handles inv messages from all peers.
// We examine the inventory advertised by the remote peer and act accordingly.
func (sm *SyncManager) handleInvMsg(imsg *invMsg) {
	peer := imsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warn("Received inv message from unknown peer %s", peer.Addr())
		return
	}

	log.Trace("Received INV msg, And current headerfirstMode is %v", sm.headersFirstMode)

	// Attempt to find the final block in the inventory list.  There may
	// not be one.
	lastBlock := -1
	invVects := imsg.inv.InvList
	for i := len(invVects) - 1; i >= 0; i-- {
		if invVects[i].Type == wire.InvTypeBlock {
			lastBlock = i
			break
		}
	}

	// If this inv contains a block announcement, and this isn't coming from
	// our current sync peer or we're current, then update the last
	// announced block for this peer. We'll use this information later to
	// update the heights of peers based on blocks we've accepted that they
	// previously announced.
	if lastBlock != -1 && (peer != sm.syncPeer || sm.current()) {
		peer.UpdateLastAnnouncedBlock(&invVects[lastBlock].Hash)
	}

	// Ignore invs from peers that aren't the sync if we are not current.
	// Helps prevent fetching a mass of orphans.
	// if peer != sm.syncPeer && !sm.current() {
	// 	return
	// }

	activeChain := chain.GetInstance()
	// If our chain is current and a peer announces a block we already
	// know of, then update their current block height.
	if lastBlock != -1 && sm.current() {
		blkIndex := activeChain.FindHashInActive(invVects[lastBlock].Hash)
		if blkIndex != nil {
			peer.UpdateLastBlockHeight(blkIndex.Height)
		}
	}

	var invBlkCnt int
	var isPushGetBlockMsg bool
	// Request the advertised inventory if we don't already have it.  Also,
	// request parent blocks of orphans if we receive one we already have.
	// Finally, attempt to detect potential stalls due to long side chains
	// we already have and request more blocks to prevent them.
	for i, iv := range invVects {
		// Ignore unsupported inventory types.
		switch iv.Type {
		case wire.InvTypeBlock:
			invBlkCnt++
		case wire.InvTypeTx:
		default:
			continue
		}

		// Add the inventory to the cache of known inventory
		// for the peer.
		peer.AddKnownInventory(iv)

		// Ignore inventory when we're in headers-first mode.
		if sm.headersFirstMode {
			continue
		}

		// Request the inventory if we don't already have it.
		haveInv, err := sm.haveInventory(iv)
		if err != nil {
			log.Warn("Unexpected failure when checking for "+
				"existing inventory during inv message "+
				"processing: %v", err)
			continue
		}
		if !haveInv {
			if iv.Type == wire.InvTypeTx {
				// Skip the transaction if it has already been
				// rejected.
				if _, exists := sm.rejectedTxns[iv.Hash]; exists {
					continue
				}
			}

			// Add it to the request queue.
			state.requestQueue = append(state.requestQueue, iv)
			continue
		}

		if iv.Type == wire.InvTypeBlock {
			// We already have the final block advertised by this
			// inventory message, so force a request for more.  This
			// should only happen if we're on a really long side
			// chain.
			if i == lastBlock {
				// Request blocks after this one up to the
				// final one the remote peer knows about (zero
				// stop hash).
				log.Debug("len of Requested block:%d", len(sm.requestedBlocks))
				activeChain := chain.GetInstance()
				blkIndex := activeChain.FindHashInActive(iv.Hash)
				locator := activeChain.GetLocator(blkIndex)
				peer.PushGetBlocksMsg(*locator, &zeroHash)
				isPushGetBlockMsg = true
			}
		}
	}

	if !isPushGetBlockMsg && invBlkCnt == len(invVects) &&
		invBlkCnt >= lchain.MaxBlocksResults && peer == sm.syncPeer {

		sm.requestBlkInvCnt = 1
	}

	log.Debug(
		"invBlkCnt=%d len(invVects)=%d sm.requestBlkInv=%v  peer=%p(%s) sm.syncPeer=%p",
		invBlkCnt, len(invVects), sm.requestBlkInvCnt,
		peer, peer.Addr(), sm.syncPeer)

	// Request as much as possible at once.  Anything that won't fit into
	// the request will be requested on the next inv message.
	numRequested := 0
	gdmsg := wire.NewMsgGetData()
	requestQueue := state.requestQueue
	for len(requestQueue) != 0 {
		iv := requestQueue[0]
		requestQueue[0] = nil
		requestQueue = requestQueue[1:]

		switch iv.Type {
		case wire.InvTypeBlock:
			// Request the block if there is not already a pending
			// request.
			if _, exists := sm.requestedBlocks[iv.Hash]; !exists {
				sm.requestedBlocks[iv.Hash] = struct{}{}
				sm.limitMap(sm.requestedBlocks, maxRequestedBlocks)
				state.requestedBlocks[iv.Hash] = struct{}{}
				gdmsg.AddInvVect(iv)
				numRequested++
			}

		case wire.InvTypeTx:
			// Request the transaction if there is not already a
			// pending request.
			if _, exists := sm.requestedTxns[iv.Hash]; !exists {
				sm.requestedTxns[iv.Hash] = struct{}{}
				sm.limitMap(sm.requestedTxns, maxRequestedTxns)
				state.requestedTxns[iv.Hash] = struct{}{}
				gdmsg.AddInvVect(iv)
				numRequested++
			}
		}

		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}

	state.requestQueue = requestQueue
	if len(gdmsg.InvList) > 0 {
		peer.QueueMessage(gdmsg, nil)
	}
}

// limitMap is a helper function for maps that require a maximum limit by
// evicting a random transaction if adding a new value would cause it to
// overflow the maximum allowed.
func (sm *SyncManager) limitMap(m map[util.Hash]struct{}, limit int) {
	if len(m)+1 > limit {
		// Remove a random entry from the map.  For most compilers, Go's
		// range statement iterates starting at a random item although
		// that is not 100% guaranteed by the spec.  The iteration order
		// is not important here because an adversary would have to be
		// able to pull off preimage attacks on the hashing function in
		// order to target eviction of specific entries anyways.
		for txHash := range m {
			delete(m, txHash)
			return
		}
	}
}

// messagesHandler is the main handler for the sync manager.  It must be run as a
// goroutine.  It processes block and inv messages in a separate goroutine
// from the peer handlers so the block (MsgBlock) messages are handled by a
// single thread without needing to lock memory data structures.  This is
// important because the sync manager controls which blocks are needed and how
// the fetching should proceed.
func (sm *SyncManager) messagesHandler() {
out:
	for {
		select {
		case m := <-sm.processBusinessChan:
			switch msg := m.(type) {
			case *newPeerMsg:
				sm.handleNewPeerMsg(msg.peer)

			case *txMsg:
				sm.handleTxMsg(msg)
				msg.reply <- struct{}{}

			case *blockMsg:
				sm.handleBlockMsg(msg)
				msg.reply <- struct{}{}

			case *invMsg:
				sm.handleInvMsg(msg)

			case *headersMsg:
				sm.handleHeadersMsg(msg)

			case *poolMsg:
				if msg.peer.Cfg.Listeners.OnMemPool != nil {
					msg.peer.Cfg.Listeners.OnMemPool(msg.peer, msg.pool)
				}
				msg.reply <- struct{}{}
			case getdataMsg:
				if msg.peer.Cfg.Listeners.OnGetData != nil {
					msg.peer.Cfg.Listeners.OnGetData(msg.peer, msg.getdata)
				}

				msg.reply <- struct{}{}
			case getBlocksMsg:
				if msg.peer.Cfg.Listeners.OnGetBlocks != nil {
					msg.peer.Cfg.Listeners.OnGetBlocks(msg.peer, msg.getblocks)
				}
				msg.reply <- struct{}{}
			case *donePeerMsg:
				sm.handleDonePeerMsg(msg.peer)

			case getSyncPeerMsg:
				var peerID int32
				if sm.syncPeer != nil {
					peerID = sm.syncPeer.ID()
				}
				msg.reply <- peerID

			case isCurrentMsg:
				msg.reply <- sm.current()

			case pauseMsg:
				// Wait until the sender unpauses the manager.
				<-msg.unpause

			default:
				log.Warn("Invalid message type in block "+
					"handler: %T, %#v", msg, msg)
			}

		case <-sm.quit:
			break out
		}
	}

	sm.wg.Done()
	log.Trace("Block handler done")
}

// handleBlockchainNotification handles notifications from blockchain.  It does
// things such as request orphan block parents and relay accepted blocks to
// connected peers.
func (sm *SyncManager) handleBlockchainNotification(notification *chain.Notification) {
	switch notification.Type {

	case chain.NTChainTipUpdated:
		event, ok := notification.Data.(*chain.TipUpdatedEvent)
		if !ok {
			panic("TipUpdatedEvent: malformed event payload")
		}

		sm.peerNotifier.RelayUpdatedTipBlocks(event)

	// A block has been accepted into the block chain.  Relay it to other peers.
	case chain.NTBlockAccepted:
		// Don't relay if we are not current. Other peers that are
		// current should already know about it.
		if !sm.current() {
			return
		}

		block, ok := notification.Data.(*block.Block)
		if !ok {
			log.Warn("Chain accepted notification is not a block.")
			break
		}

		// Generate the inventory vector and relay it.
		iv := wire.NewInvVect(wire.InvTypeBlock, &block.Header.Hash)
		sm.peerNotifier.RelayInventory(iv, &block.Header)

	// A block has been connected to the main block chain.
	case chain.NTBlockConnected:
		block, ok := notification.Data.(*block.Block)
		if !ok {
			log.Warn("Chain connected notification is not a block.")
			break
		}

		// Remove all of the transactions (except the coinbase) in the
		// connected block from the transaction pool.  Secondly, remove any
		// transactions which are now double spends as a result of these
		// new transactions.  Finally, remove any transaction that is
		// no longer an orphan. Transactions which depend on a confirmed
		// transaction are NOT removed recursively because they are still
		// valid.
		lmempool.RemoveTxSelf(block.Txs[1:])
		for _, tx := range block.Txs[1:] {
			// TODO: add it back when rcp command @SendRawTransaction is ready for broadcasting tx
			// sm.peerNotifier.TransactionConfirmed(tx)
			lmempool.ProcessOrphan(tx)
		}

		// Register block with the fee estimator, if it exists.
		if sm.feeEstimator != nil {
			err := sm.feeEstimator.RegisterBlock(block)

			// If an error is somehow generated then the fee estimator
			// has entered an invalid state. Since it doesn't know how
			// to recover, create a new one.
			if err != nil {
				sm.feeEstimator = mempool.NewFeeEstimator(
					mempool.DefaultEstimateFeeMaxRollback,
					mempool.DefaultEstimateFeeMinRegisteredBlocks)
			}
		}

		// A block has been disconnected from the main block chain.
	case chain.NTBlockDisconnected:
		block, ok := notification.Data.(*block.Block)
		if !ok {
			log.Warn("Chain disconnected notification is not a block.")
			break
		}

		// Rollback previous block recorded by the fee estimator.
		if sm.feeEstimator != nil {
			sm.feeEstimator.Rollback(&block.Header.Hash)
		}
	}
}

// NewPeer informs the sync manager of a newly active peer.
//
func (sm *SyncManager) NewPeer(peer *peer.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}
	sm.processBusinessChan <- &newPeerMsg{peer: peer}
}

// QueueTx adds the passed transaction message and peer to the block handling
// queue. Responds to the done channel argument after the tx message is
// processed.
func (sm *SyncManager) QueueTx(tx *tx.Tx, peer *peer.Peer, done chan<- struct{}) {
	// Don't accept more transactions if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.processBusinessChan <- &txMsg{tx: tx, peer: peer, reply: done}
}

// QueueBlock adds the passed block message and peer to the block handling
// queue. Responds to the done channel argument after the block message is
// processed.
func (sm *SyncManager) QueueBlock(block *block.Block, buf []byte, peer *peer.Peer, done chan<- struct{}) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.processBusinessChan <- &blockMsg{block: block, buf: buf, peer: peer, reply: done}
}

func (sm *SyncManager) QueueMessgePool(pool *wire.MsgMemPool, peer *peer.Peer, done chan<- struct{}) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.processBusinessChan <- &poolMsg{pool, peer, done}
}

func (sm *SyncManager) QueueGetData(getdata *wire.MsgGetData, peer *peer.Peer, done chan<- struct{}) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.processBusinessChan <- getdataMsg{getdata, peer, done}
}

func (sm *SyncManager) QueueGetBlocks(getblocks *wire.MsgGetBlocks, peer *peer.Peer, done chan<- struct{}) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.processBusinessChan <- getBlocksMsg{getblocks, peer, done}
}

// QueueInv adds the passed inv message and peer to the block handling queue.
func (sm *SyncManager) QueueInv(inv *wire.MsgInv, peer *peer.Peer) {
	// No channel handling here because peers do not need to block on inv
	// messages.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.processBusinessChan <- &invMsg{inv: inv, peer: peer}
}

// QueueHeaders adds the passed headers message and peer to the block handling
// queue.
func (sm *SyncManager) QueueHeaders(headers *wire.MsgHeaders, peer *peer.Peer) {
	// No channel handling here because peers do not need to block on
	// headers messages.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.processBusinessChan <- &headersMsg{headers: headers, peer: peer}
}

// DonePeer informs the blockmanager that a peer has disconnected.
func (sm *SyncManager) DonePeer(peer *peer.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.processBusinessChan <- &donePeerMsg{peer: peer}
}

// Start begins the core block handler which processes block and inv messages.
func (sm *SyncManager) Start() {
	// Already started?
	if atomic.AddInt32(&sm.started, 1) != 1 {
		return
	}

	log.Trace("Starting sync manager")
	sm.wg.Add(1)
	go sm.messagesHandler()
}

// Stop gracefully shuts down the sync manager by stopping all asynchronous
// handlers and waiting for them to finish.
func (sm *SyncManager) Stop() error {
	if atomic.AddInt32(&sm.shutdown, 1) != 1 {
		log.Warn("Sync manager is already in the process of " +
			"shutting down")
		return nil
	}

	log.Info("Sync manager shutting down")
	close(sm.quit)
	sm.wg.Wait()
	return nil
}

// SyncPeerID returns the ID of the current sync peer, or 0 if there is none.
func (sm *SyncManager) SyncPeerID() int32 {
	reply := make(chan int32)
	sm.processBusinessChan <- getSyncPeerMsg{reply: reply}
	return <-reply
}

// ProcessBlock makes use of ProcessBlock on an internal instance of a block chain.
func (sm *SyncManager) ProcessBlock(block *block.Block, flags chain.BehaviorFlags) (bool, error) {
	reply := make(chan processBlockResponse, 1)
	sm.processBusinessChan <- processBlockMsg{block: block, flags: flags, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

// IsCurrent returns whether or not the sync manager believes it is synced with
// the connected peers.
func (sm *SyncManager) IsCurrent() bool {
	reply := make(chan bool)
	sm.processBusinessChan <- isCurrentMsg{reply: reply}
	return <-reply
}

// Pause pauses the sync manager until the returned channel is closed.
//
// Note that while paused, all peer and block processing is halted.  The
// message sender should avoid pausing the sync manager for long durations.
func (sm *SyncManager) Pause() chan<- struct{} {
	c := make(chan struct{})
	sm.processBusinessChan <- pauseMsg{c}
	return c
}

// New constructs a new SyncManager. Use Start to begin processing asynchronous
// block, tx, and inv updates.
func New(config *Config) (*SyncManager, error) {
	sm := SyncManager{
		peerNotifier:        config.PeerNotifier,
		chainParams:         config.ChainParams,
		rejectedTxns:        make(map[util.Hash]struct{}),
		requestedTxns:       make(map[util.Hash]struct{}),
		requestedBlocks:     make(map[util.Hash]struct{}),
		peerStates:          make(map[*peer.Peer]*peerSyncState),
		progressLogger:      newBlockProgressLogger("Processed", log.GetLogger()),
		processBusinessChan: make(chan interface{}, config.MaxPeers*3),
		headerList:          list.New(),
		quit:                make(chan struct{}),
	}
	//chain.InitGlobalChain(nil)
	best := chain.GetInstance().Tip()
	if best == nil {
		panic("best is nil")
	}
	if !config.DisableCheckpoints {
		// Initialize the next checkpoint based on the current height.
		sm.nextCheckpoint = sm.findNextHeaderCheckpoint(best.Height)
		log.Trace("sm.nextCheckpoint : %p, best height : %d", sm.nextCheckpoint, best.Height)
		if sm.nextCheckpoint != nil {
			sm.resetHeaderState(best.GetBlockHash(), best.Height)
		}
	} else {
		log.Info("Checkpoints are disabled")
	}

	chain.GetInstance().Subscribe(sm.handleBlockchainNotification)

	return &sm, nil
}

// PeerNotifier exposes methods to notify peers of status changes to
// transactions, blocks, etc. Currently server (in the main package) implements
// this interface.
type PeerNotifier interface {
	AnnounceNewTransactions(newTxs []*mempool.TxEntry)

	UpdatePeerHeights(latestBlkHash *util.Hash, latestHeight int32, updateSource *peer.Peer)

	RelayInventory(invVect *wire.InvVect, data interface{})

	RelayUpdatedTipBlocks(event *chain.TipUpdatedEvent)

	TransactionConfirmed(tx *tx.Tx)
}

// Config is a configuration struct used to initialize a new SyncManager.
type Config struct {
	PeerNotifier PeerNotifier
	ChainParams  *model.BitcoinParams

	DisableCheckpoints bool
	MaxPeers           int
}
