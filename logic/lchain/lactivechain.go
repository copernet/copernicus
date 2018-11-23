package lchain

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/util"
	"syscall"
	"time"
)

var shutdownScheduled bool

// ActivateBestChain Make the best chain active, in multiple steps. The result is either failure
// or an activated best chain. pblock is either nullptr or a pointer to a block
// that is already loaded (to avoid loading it again from disk).
// Find the best known block, and make it the tip of the block chain
func ActivateBestChain(pblock *block.Block) error {
	// Note that while we're often called here from ProcessNewBlock, this is
	// far from a guarantee. Things in the P2P/RPC will often end up calling
	// us in the middle of ProcessNewBlock - do not assume pblock is set
	// sanely for performance or correctness!
	var pindexMostWork *blockindex.BlockIndex

	// global.CsMain.Lock()
	// defer global.CsMain.Unlock()
	for {
		if shutdownScheduled {
			break
		}

		//	todo, Add channel for receive interruption from P2P/RPC
		connTrace := make(connectTrace)

		// TODO !!! And sync.lock, cs_main
		// TODO: Tempoarily ensure that mempool removals are notified
		// before connected transactions. This shouldn't matter, but the
		// abandoned state of transactions in our wallet is currently
		// cleared when we receive another notification and there is a
		// race condition where notification of a connected conflict
		// might cause an outside process to abandon a transaction and
		// then have it inadvertently cleared by the notification that
		// the conflicted transaction was evicted.
		//mrt := mempool.NewMempoolConflictRemoveTrack(GMemPool)
		//_ = mrt
		gChain := chain.GetInstance()
		pindexOldTip := gChain.Tip()
		if pindexMostWork == nil {
			pindexMostWork = gChain.FindMostWorkChain()
		}

		// Whether we have anything to do at all.
		if pindexMostWork == nil || pindexMostWork == pindexOldTip {
			return nil
		}

		if pindexOldTip != nil && pindexMostWork.ChainWork.Cmp(&pindexOldTip.ChainWork) <= 0 {
			return nil
		}

		fInvalidFound := false
		var nullBlockPtr *block.Block
		var tmpBlock *block.Block
		hashA := pindexMostWork.GetBlockHash()
		var newHash util.Hash
		if pblock != nil {
			newHash = pblock.GetHash()
		}
		if pblock != nil && bytes.Equal(newHash[:], hashA[:]) {
			tmpBlock = pblock
		} else {
			tmpBlock = nullBlockPtr
		}

		if err := ActivateBestChainStep(pindexMostWork, tmpBlock, &fInvalidFound, connTrace); err != nil {
			return err
		}

		if fInvalidFound {
			// Wipe cache, we may need another branch now.
			pindexMostWork = nil
		}

		// MemPoolConflictRemovalTracker destroyed and conflict evictions
		// are notified

		sendNotifications(pindexOldTip, pblock)

		if gChain.Tip() == pindexMostWork {
			break
		}
	}
	// Write changes periodically to disk, after relay.
	err := disk.FlushStateToDisk(disk.FlushStatePeriodic, 0)
	stopAtHeightIfNeed(chain.GetInstance().Tip().Height)
	return err
}

func stopAtHeightIfNeed(currentHeight int32) {
	if conf.Args.StopAtHeight != -1 && conf.Args.StopAtHeight == currentHeight {
		shutdownScheduled = true
		log.Warn("got --stopatheight height %d, exiting scheduled", currentHeight)
		go func() {
			time.Sleep(2 * time.Second)
			log.Warn("got --stopatheight height %d, exiting now", currentHeight)
			syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		}()
	}
}

// sendNotifications When we reach this point, we switched to a new tip.
// Notify external listeners about the new tip.
func sendNotifications(pindexOldTip *blockindex.BlockIndex, pblock *block.Block) {
	if pblock == nil {
		return
	}

	gChain := chain.GetInstance()

	gChain.SendNotification(chain.NTBlockConnected, pblock)

	forkIndex := gChain.FindFork(pindexOldTip)
	event := chain.TipUpdatedEvent{TipIndex: gChain.Tip(), ForkIndex: forkIndex, IsInitialDownload: IsInitialBlockDownload()}
	gChain.SendNotification(chain.NTChainTipUpdated, &event)
}

// ActivateBestChainStep Try to make some progress towards making pindexMostWork
// the active block. pblock is either nullptr or a pointer to a CBlock corresponding to
// pindexMostWork.
func ActivateBestChainStep(pindexMostWork *blockindex.BlockIndex,
	pblock *block.Block, fInvalidFound *bool, connTrace connectTrace) error {

	// has held cs_main lock
	gChain := chain.GetInstance()
	pindexOldTip := gChain.Tip()
	pindexFork := gChain.FindFork(pindexMostWork)

	// Disconnect active blocks which are no longer in the best chain.
	fBlocksDisconnected := false
	for gChain.Tip() != nil && gChain.Tip() != pindexFork {
		if err := DisconnectTip(false); err != nil {
			return err
		}
		fBlocksDisconnected = true
	}

	fContinue := true
	nHeight := int32(-1)
	if pindexFork != nil {
		nHeight = pindexFork.Height
	}
	for fContinue && nHeight != pindexMostWork.Height {
		// Don't iterate the entire list of potential improvements toward the
		// best tip, as we likely only need a few blocks along the way.
		nTargetHeight := pindexMostWork.Height
		if nHeight+32 < pindexMostWork.Height {
			nTargetHeight = nHeight + 32
		}
		vpindexToConnect := make([]*blockindex.BlockIndex, 0, nTargetHeight-nHeight)
		pindexIter := pindexMostWork.GetAncestor(nTargetHeight)
		for pindexIter != nil && pindexIter.Height != nHeight {
			vpindexToConnect = append(vpindexToConnect, pindexIter)
			pindexIter = pindexIter.Prev
		}
		nHeight = nTargetHeight

		// Connect new blocks.
		var pindexConnect *blockindex.BlockIndex

		for idx := len(vpindexToConnect) - 1; idx >= 0; idx-- {
			pindexConnect = vpindexToConnect[idx]
			tmpBlock := pblock
			if pindexConnect != pindexMostWork {
				tmpBlock = nil
			}
			err := ConnectTip(pindexConnect, tmpBlock, connTrace)
			if err != nil {
				*fInvalidFound = true
				fContinue = false
				// If we didn't actually connect the block, don't notify
				// listeners about it
				delete(connTrace, pindexConnect)
				return err
			}
			currentTip := gChain.Tip()
			if pindexOldTip == nil || currentTip.ChainWork.Cmp(&pindexOldTip.ChainWork) > 0 {
				// We're in a better position than we were. Return temporarily to release the lock.
				fContinue = false
				break
			}
		}
	}

	if fBlocksDisconnected {
		currentTip := gChain.Tip()
		lmempool.RemoveForReorg(currentTip.Height+1, int(tx.StandardLockTimeVerifyFlags))
	}
	lmempool.CheckMempool(chain.GetInstance().Height())
	return nil
}

func CheckBlockIndex() error {
	if !conf.Cfg.BlockIndex.CheckBlockIndex {
		return nil
	}

	gChain := chain.GetInstance()

	// During a reindex, we read the genesis block and call CheckBlockIndex
	// before ActivateBestChain, so we have the genesis block in mapBlockIndex
	// but no active chain. (A few of the tests when iterating the block tree
	// require that chainActive has been initialized.)
	if gChain.Height() < 0 {
		if gChain.IndexMapSize() > 1 {
			return errors.New("we have no active chain, but have more than 1 blockindex")
		}
	}

	forward := gChain.BuildForwardTree()
	forwardCount := 0
	for _, v := range forward {
		forwardCount += len(v)
	}
	indexCount := gChain.IndexMapSize()
	if forwardCount != indexCount {
		err := fmt.Errorf("forward tree node count wrong, expect:%d, actual:%d", indexCount, forwardCount)
		return err
	}

	genesisSlice, ok := forward[nil]
	if ok {
		if len(genesisSlice) != 1 {
			err := fmt.Errorf("genesis block number wrong, expect only 1, actual:%d, info:%v", len(genesisSlice), genesisSlice)
			return err
		}
	} else {
		return errors.New("no any genesis block, expect 1")
	}

	nNodes := 0
	nHeight := int32(0)
	var pindexFirstInvalid *blockindex.BlockIndex
	var pindexFirstMissing *blockindex.BlockIndex
	var pindexFirstNeverProcessed *blockindex.BlockIndex
	var pindexFirstNotTreeValid *blockindex.BlockIndex
	var pindexFirstNotTransactionsValid *blockindex.BlockIndex
	var pindexFirstNotChainValid *blockindex.BlockIndex
	var pindexFirstNotScriptsValid *blockindex.BlockIndex

	pindex := genesisSlice[0]

	pruneState := disk.GetPruneState()

	for pindex != nil {
		nNodes++
		if pindexFirstInvalid == nil && pindex.Failed() {
			pindexFirstInvalid = pindex
		}
		if pindexFirstMissing == nil && !pindex.HasData() {
			pindexFirstMissing = pindex
		}
		if pindexFirstNeverProcessed == nil && pindex.TxCount == 0 {
			pindexFirstNeverProcessed = pindex
		}
		if pindex.Prev != nil && pindexFirstNotTreeValid == nil && (pindex.Status&blockindex.BlockValidMask) < blockindex.BlockValidTree {
			pindexFirstNotTreeValid = pindex
		}
		if pindex.Prev != nil && pindexFirstNotTransactionsValid == nil && (pindex.Status&blockindex.BlockValidMask) < blockindex.BlockValidTransactions {
			pindexFirstNotTransactionsValid = pindex
		}
		if pindex.Prev != nil && pindexFirstNotChainValid == nil && (pindex.Status&blockindex.BlockValidMask) < blockindex.BlockValidChain {
			pindexFirstNotChainValid = pindex
		}
		if pindex.Prev != nil && pindexFirstNotScriptsValid == nil && (pindex.Status&blockindex.BlockValidMask) < blockindex.BlockValidScripts {
			pindexFirstNotScriptsValid = pindex
		}

		// Begin: actual consistency checks.
		if pindex.Prev == nil {
			// Genesis block checks.
			// Genesis block's hash must match.
			if !pindex.GetBlockHash().IsEqual(gChain.GetParams().GenesisHash) {
				return errors.New("genesis hash not same with ActiveNet")
			}
			// The current active chain's genesis block must be this block.
			if pindex != gChain.Genesis() {
				return errors.New("genesis block not same with ActiveNet")
			}
		}
		if pindex.ChainTxCount == 0 {
			if pindex.SequenceID > 0 {
				// nSequenceId can't be set positive for blocks that aren't linked
				// (negative is used for preciousblock)
				err := fmt.Errorf("ChainTxCount=0, but SequenceID:%d, pindex:%v", pindex.SequenceID, pindex)
				return err
			}
		}
		// VALID_TRANSACTIONS is equivalent to nTx > 0 for all nodes (whether or
		// not pruning has occurred). HAVE_DATA is only equivalent to nTx > 0
		// (or VALID_TRANSACTIONS) if no pruning has occurred.
		if !pruneState.HavePruned {
			// If we've never pruned, then HAVE_DATA should be equivalent to nTx
			// > 0
			if pindex.HasData() != (pindex.TxCount > 0) {
				err := fmt.Errorf("TxCount=%d, conflict with HasData()", pindex.TxCount)
				return err
			}
			if pindexFirstMissing != pindexFirstNeverProcessed {
				err := fmt.Errorf("this two not equal, pindexFirstMissing:%v, pindexFirstNeverProcessed:%v", pindexFirstMissing, pindexFirstNeverProcessed)
				return err
			}
		} else if pindex.HasData() {
			// If we have pruned, then we can only say that HAVE_DATA implies
			// nTx > 0
			if pindex.TxCount <= 0 {
				err := fmt.Errorf("TxCount=%d, should big than 0, %v", pindex.TxCount, pindex)
				return err
			}
		}
		if pindex.HasUndo() {
			if !pindex.HasData() {
				return errors.New("if pindex HasUndo, it must HasData")
			}
		}
		if ((pindex.Status & blockindex.BlockValidMask) >= blockindex.BlockValidTransactions) != (pindex.TxCount > 0) {
			return errors.New("Valid upon Transactions equal TxCount>0, vice versa")
		}
		// All parents having had data (at some point) is equivalent to all
		// parents being VALID_TRANSACTIONS, which is equivalent to nChainTx
		// being set.
		// nChainTx != 0 is used to signal that all parent blocks have been
		// processed (but may have been pruned).
		if (pindexFirstNeverProcessed != nil) != (pindex.ChainTxCount == 0) {
			return errors.New("Parent TxCount==0 equal ChainTxCount==0, vice versa")
		}
		if (pindexFirstNotTransactionsValid != nil) != (pindex.ChainTxCount == 0) {
			return errors.New("Parent Transaction invalid equal ChainTxCount==0, vice versa")
		}

		if pindex.Height != nHeight {
			err := fmt.Errorf("height=%d, should be %d, %v", pindex.Height, nHeight, pindex)
			return err
		}
		if pindex.Prev != nil {
			if pindex.ChainWork.Cmp(&pindex.Prev.ChainWork) < 0 {
				err := fmt.Errorf("ChainWork less than its prev, %v", pindex)
				return err
			}
		}
		//if nHeight >= 2 && pindex.Skip != nil {
		//	if pindex.Skip.Height >= pindex.Height {
		//		err := fmt.Errorf("Skip.Height not less than me, %v", pindex)
		//	}
		//}
		if pindexFirstNotTreeValid != nil {
			return errors.New("All mapBlockIndex entries must at least be TREE valid")
		}
		if (pindex.Status & blockindex.BlockValidMask) >= blockindex.BlockValidChain {
			if pindexFirstNotChainValid != nil {
				return errors.New("CHAIN valid implies all parents are CHAIN valid")
			}
		}
		if (pindex.Status & blockindex.BlockValidMask) >= blockindex.BlockValidScripts {
			if pindexFirstNotScriptsValid != nil {
				return errors.New("SCRIPTS valid implies all parents are SCRIPTS valid")
			}
		}
		if pindexFirstInvalid == nil {
			// Checks for not-invalid blocks.
			if pindex.IsInvalid() {
				return errors.New("the failed mask cannot be set for blocks without invalid parents")
			}
		}

		gPersist := persist.GetInstance()
		sliceUnlinked := gPersist.GlobalMapBlocksUnlinked[pindex.Prev]
		foundInUnlinked := false
		for _, value := range sliceUnlinked {
			if value.Prev != pindex.Prev {
				return errors.New("this two index do not have same Prev")
			}
			if value == pindex {
				foundInUnlinked = true
				break
			}
		}
		if pindex.Prev != nil && pindex.HasData() && pindexFirstNeverProcessed != nil && pindexFirstInvalid == nil {
			if !foundInUnlinked {
				// If this block has block data available, some parent was never
				// received, and has no invalid parents, it must be in
				// mapBlocksUnlinked.
				return errors.New("should be in mapBlocksUnlinked")
			}
		}
		if !pindex.HasData() {
			if foundInUnlinked {
				return errors.New("Can't be in mapBlocksUnlinked if we don't HAVE_DATA")
			}
		}
		if pindexFirstMissing == nil {
			if foundInUnlinked {
				return errors.New("We aren't missing data for any parent -- cannot be in mapBlocksUnlinked")
			}
		}
		if pindex.Prev != nil && pindex.HasData() && pindexFirstNeverProcessed == nil && pindexFirstMissing != nil {
			if !pruneState.HavePruned {
				// We HAVE_DATA for this block, have received data for all parents
				// at some point, but we're currently missing data for some parent.
				// We must have pruned.
				return errors.New("must have pruned")
			}
		}

		// Try descending into the first subnode.
		subnode, ok := forward[pindex]
		if ok {
			if len(subnode) > 0 {
				// A subnode was found.
				pindex = subnode[0]
				nHeight++
				continue
			}
		}
		// This is a leaf node. Move upwards until we reach a node of which we
		// have not yet visited the last child.
		for pindex != nil {
			// We are going to either move to a parent or a sibling of pindex.
			// If pindex was the first with a certain property, unset the
			// corresponding variable.
			if pindex == pindexFirstInvalid {
				pindexFirstInvalid = nil
			}
			if pindex == pindexFirstMissing {
				pindexFirstMissing = nil
			}
			if pindex == pindexFirstNeverProcessed {
				pindexFirstNeverProcessed = nil
			}
			if pindex == pindexFirstNotTreeValid {
				pindexFirstNotTreeValid = nil
			}
			if pindex == pindexFirstNotTransactionsValid {
				pindexFirstNotTransactionsValid = nil
			}
			if pindex == pindexFirstNotChainValid {
				pindexFirstNotChainValid = nil
			}
			if pindex == pindexFirstNotScriptsValid {
				pindexFirstNotScriptsValid = nil
			}
			// Find our parent.
			pindexPar := pindex.Prev
			// Find which child we just visited.
			siblingNodes, ok := forward[pindexPar]
			if ok {
				siblingNumber := len(siblingNodes)
				i := 0
				var value *blockindex.BlockIndex
				for i, value = range siblingNodes {
					if value == pindex {
						if i+1 < siblingNumber {
							// Move to the sibling.
							pindex = siblingNodes[i+1]
							break
						}
					}
				}
				if i+1 >= siblingNumber {
					// Move up further
					pindex = pindexPar
					nHeight--
				} else {
					break
				}
			} else {
				return errors.New("will never go here")
			}
		}
	}
	// Check that we actually traversed the entire map.
	if nNodes != forwardCount {
		return errors.New("this two number should be equal")
	}
	return nil
}
