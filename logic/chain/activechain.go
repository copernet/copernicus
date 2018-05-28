package chain

import (
	
	lmp "github.com/btcboost/copernicus/logic/mempool"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	mchain "github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/tx"
)

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



