// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package service

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/logic/mempool"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/model/block"
	mpool "github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/net/wire"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/peer"
	"github.com/btcboost/copernicus/util"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/connmgr"
)

// peerSyncState stores additional information that the SyncManager tracks
// about a peer.
type peerSyncState struct {
	syncCandidate   bool
	requestQueue    []*wire.InvVect
	requestedTxns   map[util.Hash]struct{}
	requestedBlocks map[util.Hash]struct{}
}

type MsgHandle struct {
	mtx sync.Mutex
	recvFromNet  	<- chan peer.PeerMessage
	txAndBlockPro	chan peer.PeerMessage
	chainparam    *chainparams.BitcoinParams
	//connect manager
	connManager connmgr.ConnManager

	// These fields should only be accessed from the blockHandler thread
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
}

// NewMsgHandle create a msgHandle for these message from peer And RPC.
// Then begins the core block handler which processes block and inv messages.
func NewMsgHandle(ctx context.Context, cmdCh <- chan peer.PeerMessage) *MsgHandle {
	msg := &MsgHandle{mtx:sync.Mutex{}, recvFromNet:cmdCh}
	ctxChild, _ := context.WithCancel(ctx)

	go msg.startProcess(ctxChild)
	return msg
}

func (mh *MsgHandle)startProcess(ctx context.Context)  {
	ctxChild, cancel := context.WithCancel(ctx)
	defer cancel()

	go mh.txAndBlockProcess(ctxChild)
out:
	for {
		select {
		case msg := <-mh.recvFromNet:
			peerFrom := msg.Peerp
			switch data := msg.Msg.(type) {
			case *wire.MsgGetBlocks:
			//	receive getblocks request, response this request.
			case *wire.MsgGetData:
			case *wire.MsgTx:
				mh.txAndBlockPro <- msg
			case *wire.MsgBlock:
				mh.txAndBlockPro <- msg
			case *wire.MsgVersion:
			case *wire.MsgVerAck:
			case *wire.MsgPing:
			case *wire.MsgPong:
			case *wire.MsgMemPool:
			case *wire.MsgAddr:
			case *wire.MsgAlert:
			case *wire.MsgFilterLoad:
			case *wire.MsgFilterAdd:
			case *wire.MsgFeeFilter:
			case *wire.MsgFilterClear:
			case *wire.MsgInv:
			case *wire.MsgGetAddr:
			case *wire.MsgHeaders:
			case *wire.MsgSendHeaders:
			case *wire.MsgSendCmpct:
			case *wire.MsgReject:
			case *wire.MsgNotFound:
			case *wire.MsgMerkleBlock:
			case *wire.MsgHeaderAndShortTxIDs:

			}
		case <-ctx.Done():
			log.Info("msgHandle service exit. function : startProcess", )
			break out
		}
	}

}

func (mh *MsgHandle) txAndBlockProcess(ctx context.Context) {
out:
	for {
		select {
		case <-ctx.Done():
			break out
		case msg := <-mh.txAndBlockPro:
			peers := msg.Peerp
			switch data := msg.Msg.(type) {
			case *wire.MsgTx:
				acceptTx, err := mempool.ProcessTransaction(data, peers.ID())
				if err != nil{
					_ = acceptTx
				}

			case *wire.MsgBlock:
			}
		}
	}
}

func (mh *MsgHandle)startSync()  {
	if mh.syncPeer != nil{
		return
	}

	best := mh.chain.BestSnapshot()
	var bestPeer *peer.Peer
	for peer, state := range mh.peerStates {
		if !state.syncCandidate {
			continue
		}

		// Remove sync candidate peers that are no longer candidates due
		// to passing their latest known block.  NOTE: The < is
		// intentional as opposed to <=.  While technically the peer
		// doesn't have a later block when it's equal, it will likely
		// have one soon so it is a reasonable choice.  It also allows
		// the case where both are at 0 such as during regression test.
		if peer.LastBlock() < best.Height {
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
		mh.requestedBlocks = make(map[util.Hash]struct{})

		//3. locator
		locator, err := mh.chain.LatestBlockLocator()
		if err != nil {
			log.Error("Failed to get block locator for the "+
				"latest block: %v", err)
			return
		}

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
		if mh.nextCheckpoint != nil &&
			best.Height < mh.nextCheckpoint.Height &&
			mh.chainparam != &chainparams.RegressionNetParams {
			//	3. push peer
			bestPeer.PushGetHeadersMsg(locator, mh.nextCheckpoint.Hash)
			mh.headersFirstMode = true
			log.Info("Downloading headers for blocks %d to "+
				"%d from peer %s", best.Height+1,
				mh.nextCheckpoint.Height, bestPeer.Addr())
		} else {
			bestPeer.PushGetBlocksMsg(locator, &util.HashZero)
		}
		mh.syncPeer = bestPeer			//赋值 同步节点。
	} else {
		log.Warn("No sync peer candidates available")
	}
}


// handleGetData is invoked when a peer receives a getdata bitcoin message and
// is used to deliver block and transaction information.
func (mh *MsgHandle) OnGetData(msg *wire.MsgGetData) {
	numAdded := 0
	notFound := wire.NewMsgNotFound()

	p := msg.Peer
	length := len(msg.InvList)
	// A decaying ban score increase is applied to prevent exhausting resources
	// with unusually large inventory queries.
	// Requesting more than the maximum inventory vector length within a short
	// period of time yields a score above the default ban threshold. Sustained
	// bursts of small requests are not penalized as that would potentially ban
	// peers performing IBD.
	// This incremental score decays each minute to half of its value.
	p.addBanScore(0, uint32(length)*99/wire.MaxInvPerMsg, "getdata")

	// We wait on this wait channel periodically to prevent queuing
	// far more data than we can send in a reasonable time, wasting memory.
	// The waiting occurs after the database fetch for the next one to
	// provide a little pipelining.
	var waitChan chan struct{}
	doneChan := make(chan struct{}, 1)

	for i, iv := range msg.MsgData.InvList {
		var c chan struct{}
		// If this will be the last message we send.
		if i == length-1 && len(notFound.InvList) == 0 {
			c = doneChan
		} else if (i+1)%3 == 0 {
			// Buffered so as to not make the send goroutine block.
			c = make(chan struct{}, 1)
		}
		var err error
		switch iv.Type {
		case wire.InvTypeWitnessTx:
			err = sp.server.pushTxMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeTx:
			err = sp.server.pushTxMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		case wire.InvTypeWitnessBlock:
			err = sp.server.pushBlockMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeBlock:
			err = sp.server.pushBlockMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		case wire.InvTypeFilteredWitnessBlock:
			err = sp.server.pushMerkleBlockMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeFilteredBlock:
			err = sp.server.pushMerkleBlockMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		default:
			peerLog.Warnf("Unknown type in inventory request %d",
				iv.Type)
			continue
		}
		if err != nil {
			notFound.AddInvVect(iv)

			// When there is a failure fetching the final entry
			// and the done channel was sent in due to there
			// being no outstanding not found inventory, consume
			// it here because there is now not found inventory
			// that will use the channel momentarily.
			if i == len(msg.InvList)-1 && c != nil {
				<-c
			}
		}
		numAdded++
		waitChan = c
	}
	if len(notFound.InvList) != 0 {
		sp.QueueMessage(notFound, doneChan)
	}

	// Wait for messages to be sent. We can send quite a lot of data at this
	// point and this will keep the peer busy for a decent amount of time.
	// We don't process anything else by them in this time so that we
	// have an idea of when we should hear back from them - else the idle
	// timeout could fire when we were only half done sending the blocks.
	if numAdded > 0 {
		<-doneChan
	}
}

// handleHeadersMsg handles block header messages from all peers.  Headers are
// requested when performing a headers-first sync.
func (mh *MsgHandle) handleHeadersMsg(hmsg *headersMsg) {
	peer := hmsg.peer
	_, exists := mh.peerStates[peer]
	if !exists {
		log.Warn("Received headers message from unknown peer %s", peer)
		return
	}

	// The remote peer is misbehaving if we didn't request headers.
	numHeaders := len(hmsg.headers)
	if !mh.headersFirstMode {
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
	for _, blockHeader := range hmsg.headers {
		blockHash := blockHeader.GetHash()
		finalHash = &blockHash

		// Ensure there is a previous header to compare against.
		prevNodeEl := mh.headerList.Back()
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
			e := mh.headerList.PushBack(&node)
			if mh.startHeader == nil {
				mh.startHeader = e
			}
		} else {
			log.Warn("Received block header that does not "+
				"properly connect to the chain from peer %s "+
				"-- disconnecting", peer.Addr())
			peer.Disconnect()
			return
		}

		// Verify the header at the next checkpoint height matches.
		if node.height == mh.nextCheckpoint.Height {
			if node.hash.IsEqual(mh.nextCheckpoint.Hash) {
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
					mh.nextCheckpoint.Hash)
				peer.Disconnect()
				return
			}
			break
		}
	}

	// When this header is a checkpoint, switch to fetching the blocks for
	// all of the headers since the last checkpoint.
	if receivedCheckpoint {
		// Since the first entry of the list is always the final block
		// that is already in the database and is only used to ensure
		// the next header links properly, it must be removed before
		// fetching the blocks.
		mh.headerList.Remove(mh.headerList.Front())
		log.Info("Received %v block headers: Fetching blocks",
			mh.headerList.Len())
		mh.progressLogger.SetLastLogTime(time.Now())
		mh.fetchHeaderBlocks()
		return
	}

	// This header is not a checkpoint, so request the next batch of
	// headers starting from the latest known header and ending with the
	// next checkpoint.
	locator := blockchain.BlockLocator([]*chainhash.Hash{finalHash})
	err := peer.PushGetHeadersMsg(locator, mh.nextCheckpoint.Hash)
	if err != nil {
		log.Warn("Failed to send getheaders message to "+
			"peer %s: %v", peer.Addr(), err)
		return
	}

}

// fetchHeaderBlocks creates and sends a request to the syncPeer for the next
// list of blocks to be downloaded based on the current list of headers.
func (mh *MsgHandle) fetchHeaderBlocks() {
	// Nothing to do if there is no start header.
	if mh.startHeader == nil {
		log.Warn("fetchHeaderBlocks called with no start header")
		return
	}

	// Build up a getdata request for the list of blocks the headers
	// describe.  The size hint will be limited to wire.MaxInvPerMsg by
	// the function, so no need to double check it here.
	gdmsg := wire.NewMsgGetDataSizeHint(uint(mh.headerList.Len()))
	numRequested := 0
	for e := mh.startHeader; e != nil; e = e.Next() {
		node, ok := e.Value.(*headerNode)
		if !ok {
			log.Warn("Header list node type is not a headerNode")
			continue
		}

		iv := wire.NewInvVect(wire.InvTypeBlock, node.hash)
		haveInv, err := mh.haveInventory(iv)
		if err != nil {
			log.Warn("Unexpected failure when checking for "+
				"existing inventory during header block "+
				"fetch: %v", err)
		}
		if !haveInv {
			syncPeerState := mh.peerStates[mh.syncPeer]
			mh.requestedBlocks[*node.hash] = struct{}{}
			syncPeerState.requestedBlocks[*node.hash] = struct{}{}

			gdmsg.AddInvVect(iv)
			numRequested++
		}

		mh.startHeader = e.Next()
		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}
	if len(gdmsg.InvList) > 0 {
		mh.syncPeer.QueueMessage(gdmsg, nil)
	}
}

// haveInventory returns whether or not the inventory represented by the passed
// inventory vector is known.  This includes checking all of the various places
// inventory can be when it is in different states such as blocks that are part
// of the main chain, on a side chain, in the orphan pool, and transactions that
// are in the memory pool (either the main pool or orphan pool).
func (mh *MsgHandle) haveInventory(invVect *wire.InvVect) (bool, error) {
	switch invVect.Type {
	case wire.InvTypeWitnessBlock:
		fallthrough
	case wire.InvTypeBlock:
		// Ask chain if the block is known to it in any form (main
		// chain, side chain, or orphan).
		return mh.chain.HaveBlock(&invVect.Hash)

	case wire.InvTypeWitnessTx:
		fallthrough
	case wire.InvTypeTx:
		// Ask the transaction memory pool if the transaction is known
		// to it in any form (main pool or orphan).
		if mpool.Gpool.FindTx(invVect.Hash) != nil {
			return true, nil
		}

		// Check if the transaction exists from the point of view of the
		// end of the main chain.
		utxoTip := utxo.GetUtxoCacheInstance()

		coin, err := utxoTip.GetCoin(&outpoint.OutPoint{invVect.Hash, 0})
		if err != nil {
			return false, err
		}
		return entry != nil && !entry.IsFullySpent(), nil
	}

	// The requested inventory is is an unsupported type, so just claim
	// it is known to avoid requesting it.
	return true, nil
}

// Rpc process things
func (mh *MsgHandle) ProcessForRpc(message interface{}) (rsp interface{}, err error) {
	switch m := message.(type) {

	case NodeOperateMsg:
		err = mh.NodeOpera(m)

	case *tx.Tx:
		mh.recvChannel <- m
		ret := <-mh.resultChannel
		switch r := ret.(type) {
		case error:
			return nil, r
		case []*tx.Tx:
			return r, nil
		}

	case *block.Block:
		mh.recvChannel <- m
		ret := <-mh.resultChannel
		switch r := ret.(type) {
		case error:
			return nil, r
		case BlockState:
			return r, nil
		}

	case GetConnectCount:
		return mh.connManager.ConnectedCount(), nil

	case *wire.MsgPing:
		return mh.connManager.BroadCast(), nil

	case GetAddedNodeInfoMsg:
		return mh.connManager.PersistentPeers(), nil
	}

	return nil, fmt.Errorf("Unknown command")
}
