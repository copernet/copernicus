package lchain

import (
	"bytes"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/persist/disk"
)

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
	return err
}

// sendNotifications When we reach this point, we switched to a new tip.
// Notify external listeners about the new tip.
func sendNotifications(pindexOldTip *blockindex.BlockIndex, pblock *block.Block) {
	gChain := chain.GetInstance()

	gChain.SendNotification(chain.NTBlockConnected, pblock)

	forkIndex := gChain.FindFork(pindexOldTip)
	event := chain.TipUpdatedEvent{gChain.Tip(), forkIndex, IsInitialBlockDownload()}
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
	lmempool.CheckMempool()
	return nil
}

func CheckBlockIndex() error {
	return nil
}
