package chain

import (
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/util"
)

const defaultMaxTipAge = 24 * 60 * 60

type SyncingState struct {
	isAlmostSynced bool
}

func (ds *SyncingState) UpdateSyncingState() {
	if !ds.isAlmostSynced {

		gChain := GetInstance()

		if gChain.Tip() != nil {
			minWorkSum := pow.HashToBig(&model.ActiveNetParams.MinimumChainWork)
			hasEnoughWork := gChain.Tip().ChainWork.Cmp(minWorkSum) > 0

			hasRecentBlocks := int64(gChain.Tip().GetBlockTime()) > util.GetTime()-defaultMaxTipAge

			ds.isAlmostSynced = hasEnoughWork && hasRecentBlocks
		}
	}
}

func (ds *SyncingState) IsAlmostSynced() bool {
	return ds.isAlmostSynced
}
