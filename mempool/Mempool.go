package mempool

import (
	"math"
	"sync"
	"unsafe"

	beeUtils "github.com/astaxie/beego/utils"
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
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
	MapTx                       *beeUtils.BeeMap    //map[hash]TxmempoolEntry
	MapLinks                    *beeUtils.BeeMap    //map[*TxMempoolEntry]Txlinks
	MapNextTx                   *algorithm.CacheMap //map[txPreOut]tx
	MapDeltas                   map[utils.Hash]PrioriFeeDelta
	vTxHashes                   []vTxhash
	mtx                         sync.RWMutex
}

type vTxhash struct {
	hash  utils.Hash
	txRef *TxMempoolEntry
}

type PrioriFeeDelta struct {
	dPriorityDelta float64
	fee            btcutil.Amount
}

func (mempool *Mempool) RemoveRecursive(origTx *model.Tx, reason int) {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()

	txToRemove := set.New()
	origit := mempool.MapTx.Get(origTx.Hash)
	if origit != nil {
		txToRemove.Add(origit)
	} else {
		// When recursively removing but origTx isn't in the mempool be sure
		// to remove any children that are in the pool. This can happen
		// during chain re-orgs if origTx isn't re-accepted into the mempool
		// for any reason.
		for i := range origTx.Outs {
			outPoint := model.NewOutPoint(&origTx.Hash, uint32(i))
			hasTx := mempool.MapNextTx.Get(*outPoint)
			if hasTx == nil {
				continue
			}
			tx := hasTx.(model.Tx)
			tmpTxmemPoolEntry := mempool.MapTx.Get(tx.Hash)
			if tmpTxmemPoolEntry == nil {
				panic("the hasTxmemPoolEntry should not be equal nil")
			}
			txToRemove.Add(tmpTxmemPoolEntry)
		}
	}

	setAllRemoves := set.New()
	txToRemove.Each(func(item interface{}) bool {
		tmpTxmemPoolEntry := item.(TxMempoolEntry)
		mempool.CalculateDescendants(&tmpTxmemPoolEntry, setAllRemoves)
		return true
	})

	mempool.RemoveStaged(setAllRemoves, false, reason)

}

func (mempool *Mempool) CalculateDescendants(txEntry *TxMempoolEntry, setDescendants *set.Set) {

}

func (mempool *Mempool) RemoveStaged(stage *set.Set, updateDescendants bool, reason int) {

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

func (mempool *Mempool) CalculateDescendants() {

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

func (mempool *Mempool) TrackPackageRemoved(rate *utils.FeeRate) {
	defer mempool.mtx.Lock()
	if float64(rate.SataoshisPerK) > mempool.RollingMinimumFeeRate {
		mempool.RollingMinimumFeeRate = float64(rate.GetFeePerK())
		mempool.BlockSinceLatRollingFeeBump = false
	}
}

func (mempool *Mempool) TrimToSize(sizeLimit int64, pvNoSpendsRemaining *algorithm.Vector) {
	defer mempool.mtx.Lock()
	//txnRemoved := 0
	//maxFeeRateRemoved := utils.NewFeeRate(0)
	for mempool.MapTx.Count() > 0 && mempool.DynamicMemoryUsage() > sizeLimit {

	}

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

//CalculateMemPoolAncestors try to calculate all in-mempool ancestors of entry.
// (these are all calculated including the tx itself)
// fSearchForParents = whether to search a tx's vin for in-mempool parents,
// or look up parents from mapLinks. Must be true for entries not in the mempool
func (mempool *Mempool) CalculateMemPoolAncestors(entry *TxMempoolEntry, setAncestors *set.Set,
	limitAncestorCount, limitAncestorSize, limitDescendantCount,
	limitDescendantSize uint64, fSearchForParents bool) error {

	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()
	parentHashes := set.New()
	tx := *entry.TxRef

	if fSearchForParents {
		// Get parents of this transaction that are in the mempool  获取该交易在池中的父交易
		// GetMemPoolParents() is only valid for entries in the mempool, so we
		// iterate mapTx to find parents.
		for i := 0; i < len(tx.Ins); i++ {
			tx := mempool.MapTx.Get(tx.Ins[i].PreviousOutPoint.Hash)
			if tx != nil {
				parentHashes.Add(&tx)
				if uint64(parentHashes.Size()+1) > limitAncestorCount {
					return errors.Errorf("too many unconfirmed parents [limit: %d]", limitAncestorCount)
				}
			}
		}
	} else {
		// If we're not searching for parents, we require this to be an entry in
		// the mempool already.
		tx := mempool.MapTx.Get(entry.TxRef.Hash)
		if tx == nil {
			return errors.New("pass the entry is not in mempool")
		}
		parentHashes = mempool.GetMemPoolParents(tx.(*TxMempoolEntry))
	}
	totalSizeWithAncestors := entry.TxSize
	parentList := parentHashes.List()

	for len(parentList) > 0 {
		stageit := parentList[0].(*TxMempoolEntry)
		setAncestors.Add(stageit)
		parentList = parentList[1:]
		totalSizeWithAncestors = stageit.TxSize

		if stageit.SizeWithDescendants+uint64(entry.TxSize) > limitDescendantSize {
			return errors.Errorf(
				"exceeds descendant size limit for tx %s [limit: %d]",
				stageit.TxRef.Hash.ToString(), limitDescendantSize)
		} else if stageit.CountWithDescendants+1 > limitDescendantCount {
			return errors.Errorf(
				"too many descendants for tx %s [limit: %d]",
				stageit.TxRef.Hash.ToString(), limitDescendantCount)
		} else if uint64(totalSizeWithAncestors) > limitAncestorSize {
			return errors.Errorf(
				"exceeds ancestor size limit [limit: %d]",
				limitAncestorSize)
		}

		setMemPoolParents := mempool.GetMemPoolParents(stageit)
		setMemPoolParents.Each(func(item interface{}) bool {
			if !setAncestors.Has(item) {
				parentList = append(parentList, item)
			}
			if uint64(len(parentList)+setAncestors.Size()+1) > limitAncestorCount {
				return false
			}
			return true
		})

		if uint64(len(parentList)+setAncestors.Size()+1) > limitAncestorCount {
			return errors.Errorf(
				"too many unconfirmed ancestors [limit: %d]", limitAncestorCount)
		}

	}

	return nil
}

//AddUnchecked addUnchecked must updated state for all ancestors of a given transaction,
// to track size/count of descendant transactions. First version of
// addUnchecked can be used to have it call CalculateMemPoolAncestors(), and
// then invoke the second version.
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
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()

	//todo: add signal/slot In where for passes entry

	// Add to memory pool without checking anything.
	// Used by AcceptToMemoryPool(), which DOES do all the appropriate checks.
	mempool.MapTx.Set(*hash, *entry)
	mempool.MapLinks.Set(entry, TxLinks{})

	// Update transaction for any feeDelta created by PrioritiseTransaction mapTx
	if prioriFeeDelta, ok := mempool.MapDeltas[*hash]; ok {
		if prioriFeeDelta.fee > 0 {
			txEntry := mempool.MapTx.Get(*hash).(TxMempoolEntry)
			txEntry.UpdateFeeDelta(int64(prioriFeeDelta.fee))
			mempool.MapTx.Set(*hash, txEntry)
		}
	}

	// Update cachedInnerUsage to include contained transaction's usage.
	// (When we update the entry for in-mempool parents, memory usage will be
	// further updated.)
	mempool.CachedInnerUsage += uint64(entry.UsageSize)

	setParentTransactions := set.New()
	for i := 0; i < len(entry.TxRef.Ins); i++ {
		mempool.MapNextTx.Add(entry.TxRef.Ins[i].PreviousOutPoint, entry.TxRef)
		setParentTransactions.Add(*(entry.TxRef.Ins[i].PreviousOutPoint.Hash))
	}

	// Don't bother worrying about child transactions of this one. Normal case
	// of a new transaction arriving is that there can't be any children,
	// because such children would be orphans. An exception to that is if a
	// transaction enters that used to be in a block. In that case, our
	// disconnect block logic will call UpdateTransactionsFromBlock to clean up
	// the mess we're leaving here.

	// Update ancestors with information about this tx
	setParentTransactions.Each(func(item interface{}) bool {
		hash := item.(utils.Hash)
		tx := mempool.MapTx.Get(hash)
		if tx != nil {
			parentTx := tx.(TxMempoolEntry)
			mempool.UpdateParent(entry, &parentTx, true)
		}
		return true
	})
	mempool.UpdateAncestorsOf(true, entry, setAncestors)
	mempool.UpdateEntryForAncestors(entry, setAncestors)

	mempool.TransactionsUpdated++
	mempool.totalTxSize += uint64(entry.TxSize)
	mempool.MinerPolicyEstimator.ProcessTransaction(entry, validFeeEstimate)
	mempool.vTxHashes = append(mempool.vTxHashes, vTxhash{entry.TxRef.Hash, entry})
	entry.TxHashesIdx = len(mempool.vTxHashes) - 1

	return true
}

/*UpdateAncestorsOf Update ancestors of hash to add/remove it as a descendant transaction.*/
func (mempool *Mempool) UpdateAncestorsOf(add bool, entry *TxMempoolEntry, setAncestors *set.Set) {
	parentIters := mempool.GetMemPoolParents(entry)
	// add or remove this tx as a child of each parent
	parentIters.Each(func(item interface{}) bool {
		piter := item.(*TxMempoolEntry)
		mempool.UpdateChild(piter, entry, add)
		return true
	})

	updateCount := 1
	if !add {
		updateCount = -1
	}
	updateSize := updateCount * entry.TxSize
	updateFee := int64(updateCount) * entry.GetModifiedFee()

	setAncestors.Each(func(item interface{}) bool {
		ancestorIt := item.(*TxMempoolEntry)
		txEntry := mempool.MapTx.Get(ancestorIt.TxRef.Hash).(TxMempoolEntry)
		txEntry.SizeWithDescendants += uint64(updateSize)
		txEntry.UpdateDescendantState(int64(updateSize), btcutil.Amount(updateFee), int64(updateCount))
		mempool.MapTx.Set(ancestorIt.TxRef.Hash, txEntry)

		return true
	})

}

/*UpdateEntryForAncestors Set ancestor state for an entry */
func (mempool *Mempool) UpdateEntryForAncestors(entry *TxMempoolEntry, setAncestors *set.Set) {
	updateCount := setAncestors.Size()
	updateSize := int64(0)
	updateFee := btcutil.Amount(0)
	updateSigOpsCount := int64(0)

	setAncestors.Each(func(item interface{}) bool {
		ancestorIt := item.(*TxMempoolEntry)
		updateSize += int64(ancestorIt.TxSize)
		updateFee += btcutil.Amount(ancestorIt.GetModifiedFee())
		updateSigOpsCount += ancestorIt.SigOpCoungWithAncestors

		return true
	})

	entry.UpdateAncestorState(updateSize, int64(updateCount), updateSigOpsCount, updateFee)

}
