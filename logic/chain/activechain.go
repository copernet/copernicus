package chain

import (
	"bytes"
	
	
	lmp "github.com/btcboost/copernicus/logic/mempool"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	mchain "github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/persist/disk"
)

// ActivateBestChain Make the best chain active, in multiple steps. The result is either failure
// or an activated best chain. pblock is either nullptr or a pointer to a block
// that is already loaded (to avoid loading it again from disk).
// Find the best known block, and make it the tip of the block chain
func ActivateBestChain(param *chainparams.BitcoinParams, state *block.ValidationState, pblock *block.Block) bool {
	// Note that while we're often called here from ProcessNewBlock, this is
	// far from a guarantee. Things in the P2P/RPC will often end up calling
	// us in the middle of ProcessNewBlock - do not assume pblock is set
	// sanely for performance or correctness!
	var (
		pindexMostWork *blockindex.BlockIndex
		pindexNewTip   *blockindex.BlockIndex
	)
	// global.CsMain.Lock()
	// defer global.CsMain.Unlock()
	for {
		//	todo, Add channel for receive interruption from P2P/RPC
		connTrace := make(connectTrace)
		{
			// TODO !!! And sync.lock, cs_main
			// TODO: Tempoarily ensure that mempool removals are notified
			// before connected transactions. This shouldn't matter, but the
			// abandoned state of transactions in our wallet is currently
			// cleared when we receive another notification and there is a
			// race condition where notification of a connected conflict
			// might cause an outside process to abandon a transaction and
			// then have it inadvertantly cleared by the notification that
			// the conflicted transaction was evicted.
			//mrt := mempool.NewMempoolConflictRemoveTrack(GMemPool)
			//_ = mrt
			gChain := mchain.GetInstance()
			pindexOldTip := gChain.Tip()
			if pindexMostWork == nil {
				pindexMostWork = gChain.FindMostWorkChain()
			}

			// Whether we have anything to do at all.
			if pindexMostWork == nil || pindexMostWork == pindexOldTip {
				return true
			}

			fInvalidFound := false
			var nullBlockPtr *block.Block
			var tmpBlock *block.Block
			hashA := pindexMostWork.GetBlockHash()
			newHash := pblock.GetHash()
			if pblock != nil && bytes.Equal(newHash[:], hashA[:]) {
				tmpBlock = pblock
			} else {
				tmpBlock = nullBlockPtr
			}

			if !ActivateBestChainStep(param, state, pindexMostWork, tmpBlock, &fInvalidFound, connTrace) {
				return false
			}

			if fInvalidFound {
				// Wipe cache, we may need another branch now.
				pindexMostWork = nil
			}
			pindexNewTip = gChain.Tip()
			// throw all transactions though the signal-interface

		} // MemPoolConflictRemovalTracker destroyed and conflict evictions
		// are notified

		// todo  Transactions in the connnected block are notified
		

		// When we reach this point, we switched to a new tip (stored in
		// pindexNewTip).
		// Notifications/callbacks that can run without cs_main
		// Notify external listeners about the new tip.
		// TODO!!! send Asynchronous signal to external listeners.

		// Always notify the UI if a new block tip was connected
		
		if pindexNewTip == pindexMostWork {
			break
		}
	}
	// Write changes periodically to disk, after relay.
	ok := disk.FlushStateToDisk(state, disk.FlushStatePeriodic, 0)
	return ok
}

// ActivateBestChainStep Try to make some progress towards making pindexMostWork
// the active block. pblock is either nullptr or a pointer to a CBlock corresponding to
// pindexMostWork.
func ActivateBestChainStep(param *chainparams.BitcoinParams, state *block.ValidationState, pindexMostWork *blockindex.BlockIndex,
	pblock *block.Block, fInvalidFound *bool, connTrace connectTrace) bool {

	// has held cs_main lock
	gChain := mchain.GetInstance()
	pindexOldTip := gChain.Tip()
	pindexFork := gChain.FindFork(pindexMostWork)

	// Disconnect active blocks which are no longer in the best chain.
	fBlocksDisconnected := false
	for pindexOldTip != nil && pindexOldTip != pindexFork {
		if !DisconnectTip(param, state, false) {
			return false
		}
		fBlocksDisconnected = true
	}

	// Build list of new blocks to connect.
	vpindexToConnect := make([]*blockindex.BlockIndex, 0)
	fContinue := true
	nHeight := int32(-1)
	if pindexFork != nil {
		nHeight = pindexFork.Height
	}
	for fContinue && nHeight != pindexFork.Height {
		// Don't iterate the entire list of potential improvements toward the
		// best tip, as we likely only need a few blocks along the way.
		nTargetHeight := pindexMostWork.Height
		if nHeight+32 < pindexMostWork.Height {
			nTargetHeight = nHeight + 32
		}
		vpindexToConnect = make([]*blockindex.BlockIndex, 0, nTargetHeight-nHeight)
		pindexIter := pindexMostWork.GetAncestor(nTargetHeight)
		for pindexIter != nil && pindexIter.Height != nHeight {
			vpindexToConnect = append(vpindexToConnect, pindexIter)
			pindexIter = pindexIter.Prev
		}
		nHeight = nTargetHeight

		// Connect new blocks.
		var pindexConnect *blockindex.BlockIndex
		
		for idx := len(vpindexToConnect)-1;idx >= 0;idx-- {
			pindexConnect = vpindexToConnect[idx]
			tmpBlock := pblock
			if pindexConnect != pindexMostWork {
				tmpBlock = nil
			}
			if !ConnectTip(param, state, pindexConnect, tmpBlock, connTrace) {
				if state.IsInvalid() {
					// The block violates a core rule.  todo
					if !state.CorruptionPossible() {
						// InvalidChainFound(vpindexToConnect[len(vpindexToConnect)-1])
					}
					state = block.NewValidationState()
					*fInvalidFound = true
					fContinue = false
					// If we didn't actually connect the block, don't notify
					// listeners about it
					delete(connTrace, pindexConnect)
					break
				} else {
					// A system error occurred (disk space, database error, ...)
					return false
				}
			} else {
				currentTip := gChain.Tip()
				if pindexOldTip == nil || currentTip.ChainWork.Cmp(&pindexOldTip.ChainWork) > 0 {
					// We're in a better position than we were. Return temporarily to release the lock.
					fContinue = false
					break
				}
			}
		}
	}

	if fBlocksDisconnected {
		currentTip := gChain.Tip()
		lmp.RemoveForReorg( currentTip.Height+1, int(tx.StandardLockTimeVerifyFlags))
	}
	lmp.CheckMempool()
	return true
}



func CheckBlockIndex(params *chainparams.BitcoinParams) bool{
	return true
}