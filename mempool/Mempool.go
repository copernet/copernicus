package mempool

import (
	"math"
	"sync"
	"unsafe"

	"fmt"

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
	MapNextTx                   *algorithm.CacheMap //map[*OutPoint]tx
	MapDeltas                   map[utils.Hash]PrioriFeeDelta
	vTxHashes                   []vTxhash
	mtx                         sync.RWMutex
}

type vTxhash struct {
	hash  utils.Hash
	entry *TxMempoolEntry
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
			hasTx := mempool.MapNextTx.Get(outPoint)
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

	setAllRemoves := set.New() //the set element type : *TxMempoolEntry
	txToRemove.Each(func(item interface{}) bool {
		tmpTxmemPoolEntry := item.(TxMempoolEntry)
		mempool.CalculateDescendants(&tmpTxmemPoolEntry, setAllRemoves)
		return true
	})

	mempool.RemoveStaged(setAllRemoves, false, reason)

}

// CalculateDescendants calculates descendants of entry that are not already in setDescendants, and
// adds to setDescendants. Assumes entryit is already a tx in the mempool and
// setMemPoolChildren is correct for tx and all descendants. Also assumes that
// if an entry is in setDescendants already, then all in-mempool descendants of
// it are already in setDescendants as well, so that we can save time by not
// iterating over those entries.
func (mempool *Mempool) CalculateDescendants(txEntry *TxMempoolEntry, setDescendants *set.Set) {
	stage := set.New()
	if has := setDescendants.Has(txEntry); !has {
		stage.Add(txEntry)
	}

	// Traverse down the children of entry, only adding children that are not
	// accounted for in setDescendants already (because those children have
	// either already been walked, or will be walked in this iteration).
	stage.Each(func(item interface{}) bool {
		txEntryPtr := item.(*TxMempoolEntry)
		setDescendants.Add(txEntryPtr)
		setChildren := mempool.GetMempoolChildren(txEntryPtr)

		setChildren.Each(func(item interface{}) bool {
			if has := setDescendants.Has(item); !has {
				stage.Add(item)
			}
			return true
		})
		return true
	})

}

// RemoveStaged Remove a set of transactions from the mempool. If a transaction is in
// this set, then all in-mempool descendants must also be in the set, unless
// this transaction is being removed for being in a block. Set
// updateDescendants to true when removing a tx that was in a block, so that
// any in-mempool descendants have their ancestor state updated.
func (mempool *Mempool) RemoveStaged(stage *set.Set, updateDescendants bool, reason int) {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()

	mempool.UpdateForRemoveFromMempool(stage, updateDescendants)
	stage.Each(func(item interface{}) bool {
		tmpTxMempoolEntryPtr := item.(*TxMempoolEntry)
		mempool.removeUnchecked(tmpTxMempoolEntryPtr, reason)
		return true
	})
}

// UpdateForRemoveFromMempool For each transaction being removed, update ancestors
// and any direct children. If updateDescendants is true, then also update in-mempool
// descendants' ancestor state.
func (mempool *Mempool) UpdateForRemoveFromMempool(entriesToRemove *set.Set, updateDescendants bool) {
	// For each entry, walk back all ancestors and decrement size associated
	// with this transaction.
	nNoLimit := uint64(math.MaxUint64)
	if updateDescendants {
		// updateDescendants should be true whenever we're not recursively
		// removing a tx and all its descendants, eg when a transaction is
		// confirmed in a block. Here we only update statistics and not data in
		// mapLinks (which we need to preserve until we're finished with all
		// operations that need to traverse the mempool).
		entriesToRemove.Each(func(item interface{}) bool {
			setDescendants := set.New()
			removeIt := item.(*TxMempoolEntry)
			mempool.CalculateDescendants(removeIt, setDescendants)
			setDescendants.Remove(removeIt)

			modifySize := -int64(removeIt.TxSize)
			modifyFee := -removeIt.GetModifiedFee()
			modifySigOps := -removeIt.SigOpCount
			setDescendants.Each(func(item interface{}) bool {
				dit := item.(*TxMempoolEntry)
				dit.UpdateAncestorState(modifySize, -1, modifySigOps, modifyFee)
				mempool.MapTx.Set(dit.TxRef.Hash, *dit)
				return true
			})
			return true
		})
	}

	entriesToRemove.Each(func(item interface{}) bool {
		entry := item.(*TxMempoolEntry)
		setAncestors := set.New()
		// Since this is a tx that is already in the mempool, we can call CMPA
		// with fSearchForParents = false.  If the mempool is in a consistent
		// state, then using true or false should both be correct, though false
		// should be a bit faster.
		// However, if we happen to be in the middle of processing a reorg, then
		// the mempool can be in an inconsistent state. In this case, the set of
		// ancestors reachable via mapLinks will be the same as the set of
		// ancestors whose packages include this transaction, because when we
		// add a new transaction to the mempool in addUnchecked(), we assume it
		// has no children, and in the case of a reorg where that assumption is
		// false, the in-mempool children aren't linked to the in-block tx's
		// until UpdateTransactionsFromBlock() is called. So if we're being
		// called during a reorg, ie before UpdateTransactionsFromBlock() has
		// been called, then mapLinks[] will differ from the set of mempool
		// parents we'd calculate by searching, and it's important that we use
		// the mapLinks[] notion of ancestor transactions as the set of things
		// to update for removal.
		mempool.CalculateMemPoolAncestors(entry, setAncestors, nNoLimit,
			nNoLimit, nNoLimit, nNoLimit, false)

		// Note that UpdateAncestorsOf severs the child links that point to
		// removeIt in the entries for the parents of removeIt.
		mempool.UpdateAncestorsOf(false, entry, setAncestors)

		return true
	})

	// After updating all the ancestor sizes, we can now sever the link between
	// each transaction being removed and any mempool children (ie, update
	// setMemPoolParents for each direct child of a transaction being removed).
	entriesToRemove.Each(func(item interface{}) bool {
		removeIt := item.(*TxMempoolEntry)
		mempool.UpdateChildrenForRemoval(removeIt)
		return true
	})
}

// UpdateChildrenForRemoval Sever link between specified transaction and direct children.
func (mempool *Mempool) UpdateChildrenForRemoval(entry *TxMempoolEntry) {
	setMemPoolChildren := mempool.GetMempoolChildren(entry)
	setMemPoolChildren.Each(func(item interface{}) bool {
		updateIt := item.(*TxMempoolEntry)
		mempool.UpdateParent(updateIt, entry, false)

		return true
	})
}

func (mempool *Mempool) removeUnchecked(entry *TxMempoolEntry, reason int) {
	//todo: add signal/slot In where for passed entry

	txid := entry.TxRef.Hash
	for _, txin := range entry.TxRef.Ins {
		mempool.MapNextTx.Del(txin.PreviousOutPoint)
	}
	if len(mempool.vTxHashes) > 1 {
		mempool.vTxHashes[entry.vTxHashesIdx] = mempool.vTxHashes[len(mempool.vTxHashes)-1]
		mempool.vTxHashes[entry.vTxHashesIdx].entry.vTxHashesIdx = entry.vTxHashesIdx
		mempool.vTxHashes = mempool.vTxHashes[:len(mempool.vTxHashes)-1]
	} else {
		mempool.vTxHashes = make([]vTxhash, 0)
	}

	mempool.totalTxSize -= uint64(entry.TxSize)
	mempool.CachedInnerUsage -= uint64(mempool.DynamicMemoryUsage())
	//todo add all memusage function
	//mempool.CachedInnerUsage -= DynamicUsage
	mempool.MapLinks.Delete(entry)
	mempool.MapTx.Delete(entry.TxRef.Hash)
	mempool.TransactionsUpdated++
	mempool.MinerPolicyEstimator.RemoveTx(txid)

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

func (mempool *Mempool) TrackPackageRemoved(rate *utils.FeeRate) {
	defer mempool.mtx.Lock()
	if float64(rate.SataoshisPerK) > mempool.RollingMinimumFeeRate {
		mempool.RollingMinimumFeeRate = float64(rate.GetFeePerK())
		mempool.BlockSinceLatRollingFeeBump = false
	}
}

func (mempool *Mempool) TrimToSize(sizeLimit int64, pvNoSpendsRemaining *algorithm.Vector) {
	defer mempool.mtx.Lock()
	txNRemoved := 0
	maxFeeRateRemoved := utils.NewFeeRate(0)
	for _, it := range mempool.MapTx.Items() {
		if mempool.DynamicMemoryUsage() > sizeLimit {
			return
		}
		// We set the new mempool min fee to the feerate of the removed set,
		// plus the "minimum reasonable fee rate" (ie some value under which we
		// consider txn to have 0 fee). This way, we don't allow txn to enter
		// mempool with feerate equal to txn which were removed with no block in
		// between.
		txMempoolEntry := it.(TxMempoolEntry)
		removed := utils.NewFeeRateWithSize(int64(txMempoolEntry.ModFeesWithDescendants), int(txMempoolEntry.SizeWithDescendants))
		removed = utils.NewFeeRate(removed.SataoshisPerK + IncrementalRelayFee.SataoshisPerK)
		mempool.TrackPackageRemoved(removed)
		if removed.SataoshisPerK > maxFeeRateRemoved.SataoshisPerK {
			maxFeeRateRemoved = removed
		}
		stage := set.New()
		mempool.CalculateDescendants(&txMempoolEntry, stage)
		txNRemoved += stage.Size()
		txn := algorithm.NewVector()
		if pvNoSpendsRemaining != nil {
			txn.ReverseArray()
			for _, iter := range stage.List() {
				txMempoolEntryIter := iter.(TxMempoolEntry)
				txn.PushBack(txMempoolEntryIter.TxRef)
			}
		}
		mempool.RemoveStaged(stage, false, SIZELIMIT)
		if pvNoSpendsRemaining != nil {
			for _, t := range txn.Array {
				tx := t.(model.Tx)
				for _, txin := range tx.Ins {
					if mempool.ExistsHash(txin.PreviousOutPoint.Hash) {
						continue
					}
					if mempool.MapNextTx.Get(txin.PreviousOutPoint) != nil {
						pvNoSpendsRemaining.PushBack(txin.PreviousOutPoint)
					}

				}

			}
		}

	}
	if maxFeeRateRemoved.SataoshisPerK > 0 {
		fmt.Println("mempool", "removed %u txn , rolling minimum fee bumped tp %s ", txNRemoved, maxFeeRateRemoved.String())
	}

}

func (mempool *Mempool) DynamicMemoryUsage() int64 {
	defer mempool.mtx.Lock()
	return int64(unsafe.Sizeof(mempool.MapTx)) + int64(unsafe.Sizeof(mempool.MapNextTx)) + int64(mempool.CachedInnerUsage)

}

func (mempool *Mempool) ExistsHash(hash *utils.Hash) bool {
	defer mempool.mtx.RLock()
	return mempool.MapTx.Get(hash) != nil
}

func (mempool *Mempool) ExistsOutPoint(outpoint *model.OutPoint) bool {
	defer mempool.mtx.RLock()
	v := mempool.MapTx.Get(outpoint.Hash)
	if v == nil {
		return false
	}
	txMempoolEntry := v.(TxMempoolEntry)
	return int(outpoint.Index) < len(txMempoolEntry.TxRef.Outs)
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

// AddUnchecked addUnchecked must updated state for all ancestors of a given transaction,
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

	//todo: add signal/slot In where for passed entry

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
	entry.vTxHashesIdx = len(mempool.vTxHashes) - 1

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
	updateFee := btcutil.Amount(updateCount) * entry.GetModifiedFee()

	setAncestors.Each(func(item interface{}) bool {
		ancestorIt := item.(*TxMempoolEntry)
		txEntry := mempool.MapTx.Get(ancestorIt.TxRef.Hash).(TxMempoolEntry)
		txEntry.SizeWithDescendants += uint64(updateSize)
		txEntry.UpdateDescendantState(int64(updateSize), updateFee, int64(updateCount))
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
		updateFee += ancestorIt.GetModifiedFee()
		updateSigOpsCount += ancestorIt.SigOpCoungWithAncestors

		return true
	})

	entry.UpdateAncestorState(updateSize, int64(updateCount), updateSigOpsCount, updateFee)

}

func (mempool *Mempool) TransactionWithinChainLimit(txid *utils.Hash, chainLimit uint64) bool {
	defer mempool.mtx.RLock()
	it := mempool.MapTx.Get(txid)
	txMempoolEntry := it.(TxMempoolEntry)
	return txMempoolEntry.CountWithDescendants < chainLimit && txMempoolEntry.CountWithAncestors < chainLimit
}
