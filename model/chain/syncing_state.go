package chain

import (
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/util"
)

const defaultMaxTipAge = 24 * 60 * 60

type SyncingState struct {
	isAlmostSynced bool
}

func (ds *SyncingState) UpdateSyncingState() {
	if ds.isAlmostSynced || GetInstance().Tip() == nil {
		return
	}

	tip := GetInstance().Tip()
	ds.isAlmostSynced = isRecentTip(tip) && hasEnoughWork(tip)
}

func isRecentTip(tip *blockindex.BlockIndex) bool {
	return int64(tip.GetBlockTime()) > util.GetTimeSec()-defaultMaxTipAge
}

func hasEnoughWork(tip *blockindex.BlockIndex) bool {
	minWorkSum := pow.HashToBig(&model.ActiveNetParams.MinimumChainWork)
	return tip.ChainWork.Cmp(minWorkSum) > 0
}

func (ds *SyncingState) IsAlmostSynced() bool {
	return ds.isAlmostSynced
}
