package mempool

import (
	"fmt"

	beeUtils "github.com/astaxie/beego/utils"
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

/**
 * Fake height value used in Coins to signify they are only in the memory
 * pool(since 0.8)
 */
const (
	MEMPOOL_HEIGHT       = 0x7FFFFFFF
	ROLLING_FEE_HALFLIFE = 60 * 60 * 12
)

type Mempool struct {
	CheckFrequency              uint32
	TransactionsUpdated         int
	MinerPolicyEstimator        *BlockPolicyEstimator
	totalTxSize                 uint64
	CachedInnerUsage            uint64
	LastRollingFeeUpdate        int64
	BlockSinceLatRollingFeeBump bool
	RollingMinimumFeeRate       float64
	MapTx                       *beeUtils.BeeMap
	MapLinks                    *beeUtils.BeeMap //<TxMempoolEntry,Txlinks>
}

// UpdateForDescendants : Update the given tx for any in-mempool descendants.
// Assumes that setMemPoolChildren is correct for the given tx and all
// descendants.
func (mempool *Mempool) UpdateForDescendants(updateIt *TxMempoolEntry, cachedDescendants *beeUtils.BeeMap, setExclude algorithm.Vector) {

	var stageEntries algorithm.Vector
	var setAllDescendants algorithm.Vector

	for !stageEntries.Empty() {
		cit, _ := stageEntries.At(0)
		setAllDescendants.PushBack(cit)
		stageEntries.RemoveAt(0)
		txMempoolEntry := cit.(TxMempoolEntry)
		setChildren := mempool.GetMempoolChildren(&txMempoolEntry)

		for _, childEntry := range setChildren.Array {
			cacheIt := cachedDescendants.Get(childEntry)
			fmt.Println(cacheIt)

		}

	}
}

func (mempool *Mempool) GetMempoolChildren(entry *TxMempoolEntry) *algorithm.Vector {
	result := mempool.MapLinks.Get(entry)
	return result.(TxLinks).Children
}

func (mempool *Mempool) GetMemPoolParents(entry *TxMempoolEntry) *algorithm.Vector {
	result := mempool.MapLinks.Get(entry)
	return result.(TxLinks).Parents
}

func (mempool *Mempool) GetMinFee(sizeLimit uint) utils.FeeRate {
	return utils.FeeRate{SataoshisPerK: 0}
}

func AllowFee(priority float64) bool {
	// Large (in bytes) low-priority (new, small-coin) transactions need a fee.
	return priority > AllowFreeThreshold()
}

func GetTxFromMemPool(hash *utils.Hash) *model.Tx {
	return new(model.Tx)
}

func AllowFreeThreshold() float64 {
	return (float64(utils.COIN) * 144) / 250

}
