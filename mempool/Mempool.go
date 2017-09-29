package mempool

import (
	"sync"

	"unsafe"

	"math"

	beeUtils "github.com/astaxie/beego/utils"
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"gopkg.in/fatih/set.v0"
)

/**
 * Fake height value used in Coins to signify they are only in the memory
 * pool(since 0.8)
 */
const (
	MEMPOOL_HEIGHT       = 0x7FFFFFFF
	ROLLING_FEE_HALFLIFE = 60 * 60 * 12
)

var IncrementalRelayFee utils.FeeRate

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
	MapNextTx                   *algorithm.CacheMap

	mtx sync.RWMutex
}

func (mempool *Mempool) RemoveRecursive(origTx model.Tx, reason int) {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()

}

// UpdateForDescendants : Update the given tx for any in-mempool descendants.
// Assumes that setMemPoolChildren is correct for the given tx and all
// descendants.
func (mempool *Mempool) UpdateForDescendants(updateIt *TxMempoolEntry, cachedDescendants *algorithm.CacheMap, setExclude set.Set) {

	stageEntries := set.New()
	setAllDescendants := set.New()

	for !stageEntries.IsEmpty() {
		cit := stageEntries.List()[0]
		setAllDescendants.Add(cit)
		stageEntries.Remove(cit)
		txMempoolEntry := cit.(TxMempoolEntry)
		setChildren := mempool.GetMempoolChildren(&txMempoolEntry)

		for _, childEntry := range setChildren.List() {
			childTx := childEntry.(TxMempoolEntry)
			cacheIt := cachedDescendants.Get(childTx)
			cacheItVector := cacheIt.(algorithm.Vector)
			if cacheIt != cachedDescendants.Last() {
				// We've already calculated this one, just add the entries for
				// this set but don't traverse again.
				for _, cacheEntry := range cacheItVector.Array {
					setAllDescendants.Add(cacheEntry)

				}
			} else if !setAllDescendants.Has(childEntry) {
				// Schedule for later processing
				stageEntries.Add(childEntry)
			}

		}

	}
	// setAllDescendants now contains all in-mempool descendants of updateIt.
	// Update and add to cached descendant map
	modifySize := 0
	modifyFee := 0
	modifyCount := 0

	for _, cit := range setAllDescendants.List() {
		txCit := cit.(TxMempoolEntry)
		if !setExclude.Has(txCit.TxRef.Hash) {
			modifySize = modifySize + txCit.TxSize
			modifyFee = modifyFee + txCit.ModSize
			modifyCount++
			cachedSet := cachedDescendants.Get(updateIt).(set.Set)
			cachedSet.Add(txCit)
			// todo Update ancestor state for each descendant
		}
	}
	//todo Update descendant
}

// UpdateTransactionsFromBlock : vHashesToUpdate is the set of transaction hashes from a disconnected block
// which has been re-added to the mempool. For each entry, look for descendants
// that are outside hashesToUpdate, and add fee/size information for such
// descendants to the parent. For each such descendant, also update the ancestor
// state to include the parent.
func (mempool *Mempool) UpdateTransactionsFromBlock(hashesToUpdate algorithm.Vector) {
	mempool.mtx.Lock()
	//var mapMemPoolDescendantsToUpdate algorithm.CacheMap
	//setAlreadyIncluded := set.New(hashesToUpdate.Array...)
	//
	//// Iterate in reverse, so that whenever we are looking at at a transaction
	//// we are sure that all in-mempool descendants have already been processed.
	//// This maximizes the benefit of the descendant cache and guarantees that
	//// setMemPoolChildren will be updated, an assumption made in
	//// UpdateForDescendants.
	//hashesToUpdateReverse := hashesToUpdate.ReverseArray()
	//for _, hash := range hashesToUpdateReverse {
	//	setChildren := set.New()
	//	txiter := mempool.MapTx.Get(hash)
	//
	//}
	mempool.mtx.Unlock()
}

func (mempool *Mempool) UpdateChild(entry *TxMempoolEntry, entryChild *TxMempoolEntry, add bool) {
	s := set.New()
	txLinks := mempool.MapLinks.Get(entry).(TxLinks)
	children := txLinks.Children
	if add {
		children.Add(entryChild)
		mempool.CachedInnerUsage += uint64(IncrementalDynamicUsageTxMempoolEntry(s))
	} else {
		children.Remove(entryChild)
		mempool.CachedInnerUsage -= uint64(IncrementalDynamicUsageTxMempoolEntry(s))
	}
}

func (mempool *Mempool) UpdateParent(entry, entryParent *TxMempoolEntry, add bool) {
	s := set.New()
	txLinks := mempool.MapLinks.Get(entry).(TxLinks)
	parents := txLinks.Parents
	if add {
		parents.Add(entryParent)
		mempool.CachedInnerUsage += uint64(IncrementalDynamicUsageTxMempoolEntry(s))
	} else {
		parents.Remove(entryParent)
		mempool.CachedInnerUsage -= uint64(IncrementalDynamicUsageTxMempoolEntry(s))
	}
}

func (mempool *Mempool) GetMempoolChildren(entry *TxMempoolEntry) *set.Set {
	result := mempool.MapLinks.Get(entry)
	if result == nil {
		panic("No have children In mempool for this TxmempoolEntry")
	}
	return result.(TxLinks).Children
}

func (mempool *Mempool) GetMemPoolParents(entry *TxMempoolEntry) *set.Set {
	result := mempool.MapLinks.Get(entry)
	if result == nil {
		panic("No have parant In mempool for this TxmempoolEntry")
	}
	return result.(TxLinks).Parents
}

func (mempool *Mempool) GetMinFee(sizeLimit int64) *utils.FeeRate {
	defer mempool.mtx.Lock()
	if mempool.BlockSinceLatRollingFeeBump || mempool.RollingMinimumFeeRate == 0 {
		return utils.NewFeeRate(int64(mempool.RollingMinimumFeeRate))
	}
	time := utils.GetMockTime()
	if time > mempool.LastRollingFeeUpdate+10 {
		halfLife := ROLLING_FEE_HALFLIFE
		mempoolUsage := mempool.DynamicMemoryUsage()
		if mempoolUsage < sizeLimit/4 {
			halfLife = halfLife / 4
		} else if mempoolUsage < sizeLimit/2 {
			halfLife = halfLife / 2
		}

		mempool.RollingMinimumFeeRate = mempool.RollingMinimumFeeRate / math.Pow(2.0, float64(time-mempool.LastRollingFeeUpdate)/float64(halfLife))
		mempool.LastRollingFeeUpdate = time
		if mempool.RollingMinimumFeeRate < float64(IncrementalRelayFee.SataoshisPerK)/2 {
			mempool.RollingMinimumFeeRate = 0
			return utils.NewFeeRate(0)
		}

	}
	result := int64(math.Max(mempool.RollingMinimumFeeRate, float64(IncrementalRelayFee.SataoshisPerK)))
	return utils.NewFeeRate(result)
}

func (mempool *Mempool) DynamicMemoryUsage() int64 {
	defer mempool.mtx.Lock()
	return int64(unsafe.Sizeof(mempool.MapTx)) + int64(unsafe.Sizeof(mempool.MapNextTx)) + int64(mempool.CachedInnerUsage)

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

func (mempool *Mempool) CalculateMemPoolAncestors(entry *TxMempoolEntry, setEntries *set.Set,
	limitAncestorCount, limitAncestorSize, limitDescendantCount,
	limitDescendantSize uint64, fSearchForParents bool) error {

	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()
	//parentHashes := set.New()
	tx := *entry.TxRef

	if fSearchForParents {
		// Get parents of this transaction that are in the mempool  获取该交易在池中的父交易
		// GetMemPoolParents() is only valid for entries in the mempool, so we
		// iterate mapTx to find parents.
		for i := 0; i < len(tx.Ins); i++ {

		}
	}

	return nil
}

func (mempool *Mempool) AddUnchecked(hash *utils.Hash, entry *TxMempoolEntry, validFeeEstimate bool) bool {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()
	setAncestors := set.New()
	nNoLimit := uint64(math.MaxUint64)

	err := mempool.CalculateMemPoolAncestors(entry, setAncestors, nNoLimit, nNoLimit, nNoLimit, nNoLimit, false)
	if err != nil {
		return false
	}
	return mempool.AddUncheckedWithAncestors(hash, entry, setAncestors, validFeeEstimate)
}

func (mempool *Mempool) AddUncheckedWithAncestors(hash *utils.Hash, entry *TxMempoolEntry, setAncestors *set.Set, validFeeEstimate bool) bool {
	return false
}
