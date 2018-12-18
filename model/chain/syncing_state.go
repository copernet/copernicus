package chain

import (
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/util"
	"sync/atomic"
)

type SyncingState struct {
	isCurrent int32
}

func (ds *SyncingState) UpdateSyncingState() {
	isCurrent := atomic.LoadInt32(&ds.isCurrent) == 1
	if isCurrent || GetInstance().Tip() == nil {
		return
	}

	tip := GetInstance().Tip()

	if tipInOneDay(tip) && hasEnoughWork(tip) {
		atomic.StoreInt32(&ds.isCurrent, 1)
	}
}

func tipInOneDay(tip *blockindex.BlockIndex) bool {
	return int64(tip.GetBlockTime()) > util.GetTimeSec()-24*60*60
}

func hasEnoughWork(tip *blockindex.BlockIndex) bool {
	minWorkSum := pow.MiniChainWork()
	return tip.ChainWork.Cmp(&minWorkSum) > 0
}

func (ds *SyncingState) IsAlmostSynced() bool {
	return atomic.LoadInt32(&ds.isCurrent) == 1
}
