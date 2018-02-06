package mempool

import (
	"math"
	"sync"
	"unsafe"

	"fmt"

	"encoding/binary"
	"io"

	beeUtils "github.com/astaxie/beego/utils"
	"github.com/bradfitz/slice"
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/conf"
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

type Mempool struct {
	CheckFrequency              uint32
	TransactionsUpdated         int
	MinerPolicyEstimator        *BlockPolicyEstimator
	totalTxSize                 uint64
	CachedInnerUsage            uint64
	LastRollingFeeUpdate        int64
	BlockSinceLatRollingFeeBump bool
	RollingMinimumFeeRate       float64
	MapTx                       *MultiIndex
	MapLinks                    *beeUtils.BeeMap    //map[utils.Hash]*Txlinks
	MapNextTx                   *algorithm.CacheMap //map[refOutPoint]tx
	MapDeltas                   map[utils.Hash]PriorityFeeDelta
	vTxHashes                   []TxHash
	mtx                         sync.RWMutex
}

func (mempool *Mempool) RemoveRecursive(origTx *model.Tx, reason int) {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()
	txToRemove := set.New()
	origit := mempool.MapTx.GetEntryByHash(origTx.Hash)
	if origit != nil {
		txToRemove.Add(origit)
	} else {
		// When recursively removing but origTx isn't in the mempool be sure
		// to remove any children that are in the pool. This can happen
		// during chain re-orgs if origTx isn't re-accepted into the mempool
		// for any reason.
		for i := range origTx.Outs {
			hasTx := mempool.MapNextTx.Get(refOutPoint{origTx.Hash, uint32(i)})
			if hasTx == nil {
				continue
			}
			tx := hasTx.(model.Tx)
			tmpTxmemPoolEntry := mempool.MapTx.GetEntryByHash(tx.Hash)
			if tmpTxmemPoolEntry == nil {
				panic("the hasTxmemPoolEntry should not be equal nil")
			}
			txToRemove.Add(tmpTxmemPoolEntry)
		}
	}
	setAllRemoves := set.New() //the set element type : *TxMempoolEntry
	txToRemove.Each(func(item interface{}) bool {
		tmpTxmemPoolEntry := item.(*TxMempoolEntry)
		mempool.CalculateDescendants(tmpTxmemPoolEntry, setAllRemoves)
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
	stageList := stage.List()
	for len(stageList) > 0 {
		txEntryPtr := stageList[0].(*TxMempoolEntry)
		setDescendants.Add(txEntryPtr)
		stageList = stageList[1:]
		setChildren := mempool.GetMempoolChildren(txEntryPtr)
		setChildren.Each(func(item interface{}) bool {
			if has := setDescendants.Has(item); !has {
				stageList = append(stageList, item)
			}
			return true
		})
	}
}

// RemoveStaged Remove a set of transactions from the mempool. If a transaction is in
// this set, then all in-mempool descendants must also be in the set, unless
// this transaction is being removed for being in a block. Set
// updateDescendants to true when removing a tx that was in a block, so that
// any in-mempool descendants have their ancestor state updated.
func (mempool *Mempool) RemoveStaged(stage *set.Set, updateDescendants bool, reason int) {
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

func (mempool *Mempool) PrioritiseTransaction(hash utils.Hash, strHash string, priorityDelta float64, feeDelta int64) {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()
	deltas := mempool.MapDeltas[hash]
	deltas.priorityDelta += priorityDelta
	deltas.fee += btcutil.Amount(feeDelta)
	it := mempool.MapTx.GetEntryByHash(hash)
	if it != nil {
		it.UpdateFeeDelta(int64(deltas.fee))

		// Now update all ancestors' modified fees with descendants
		setAncestors := set.New()
		noLimit := uint64(math.MaxUint64)
		err := mempool.CalculateMemPoolAncestors(it, setAncestors, noLimit, noLimit, noLimit, noLimit, false)
		if err != nil {
			fmt.Printf("calculate mempool ancestors  err :%s \n", err.Error())
		}
		for _, txiter := range setAncestors.List() {
			txMempoolEntry := txiter.(*TxMempoolEntry)
			txMempoolEntry.UpdateDescendantState(0, btcutil.Amount(feeDelta), 0)
		}
		// Now update all descendants' modified fees with ancestors
		setDescendants := set.New()
		mempool.CalculateDescendants(it, setDescendants)
		setDescendants.Remove(it)
		for _, txiter := range setDescendants.List() {
			txMempoolEntry := txiter.(*TxMempoolEntry)
			txMempoolEntry.UpdateAncestorState(0, 0, 0, btcutil.Amount(feeDelta))
		}
	}
}

func (mempool *Mempool) ApplyDeltas(hash utils.Hash, priorityDelta float64, feeDelta int64) (float64, int64) {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()
	if deltas, ok := mempool.MapDeltas[hash]; ok {
		priorityDelta += deltas.priorityDelta
		feeDelta += int64(deltas.fee)
	}

	return priorityDelta, feeDelta
}

func (mempool *Mempool) ClearPrioritisation(hash *utils.Hash) {
	delete(mempool.MapDeltas, *hash)
}

/* removeUnchecked Before calling removeUnchecked for a given transaction,
 * UpdateForRemoveFromMempool must be called on the entire (dependent) set
 * of transactions being removed at the same time. We use each
 * CTxMemPoolEntry's setMemPoolParents in order to walk ancestors of a given
 * transaction that is removed, so we can't remove intermediate transactions
 * in a chain before we've updated all the state for the removal.
 */
func (mempool *Mempool) removeUnchecked(entry *TxMempoolEntry, reason int) {
	//todo: add signal/slot In where for passed entry
	for _, txin := range entry.TxRef.Ins {
		mempool.MapNextTx.Del(refOutPoint{txin.PreviousOutPoint.Hash, txin.PreviousOutPoint.Index})
	}

	if len(mempool.vTxHashes) > 1 {
		mempool.vTxHashes[entry.vTxHashesIdx] = mempool.vTxHashes[len(mempool.vTxHashes)-1]
		mempool.vTxHashes[entry.vTxHashesIdx].entry.vTxHashesIdx = entry.vTxHashesIdx
		mempool.vTxHashes = mempool.vTxHashes[:len(mempool.vTxHashes)-1]
	} else {
		mempool.vTxHashes = make([]TxHash, 0)
	}
	mempool.totalTxSize -= uint64(entry.TxSize)
	mempool.CachedInnerUsage -= uint64(entry.UsageSize)
	s := set.New()
	tmpTxLink := mempool.MapLinks.Get(entry.TxRef.Hash).(*TxLinks)
	mempool.CachedInnerUsage -= uint64(int64(tmpTxLink.Children.Size())*IncrementalDynamicUsageTxMempoolEntry(s) + int64(tmpTxLink.Parents.Size())*IncrementalDynamicUsageTxMempoolEntry(s))
	mempool.MapLinks.Delete(entry.TxRef.Hash)
	mempool.MapTx.DelEntryByHash(entry.TxRef.Hash)
	mempool.TransactionsUpdated++
	mempool.MinerPolicyEstimator.RemoveTx(entry.TxRef.Hash)
}

// UpdateForDescendants : Update the given tx for any in-mempool descendants.
// Assumes that setMemPoolChildren is correct for the given tx and all
// descendants.
func (mempool *Mempool) UpdateForDescendants(updateIt *TxMempoolEntry, cachedDescendants *algorithm.CacheMap, setExclude *set.Set) {

	stageEntries := mempool.GetMempoolChildren(updateIt)
	setAllDescendants := set.New()

	for !stageEntries.IsEmpty() {
		cit := stageEntries.List()[0]
		setAllDescendants.Add(cit)
		stageEntries.RemoveItem(cit)
		txMempoolEntry := cit.(*TxMempoolEntry)
		setChildren := mempool.GetMempoolChildren(txMempoolEntry)

		for _, childEntry := range setChildren.List() {
			childTx := childEntry.(*TxMempoolEntry)
			cacheIt := cachedDescendants.Get(childTx)
			if cacheIt != nil {
				cacheItVector := cacheIt.(algorithm.Vector)
				// We've already calculated this one, just add the entries for
				// this set but don't traverse again.
				for _, cacheEntry := range cacheItVector.Array {
					setAllDescendants.Add(cacheEntry)
				}
			} else if !setAllDescendants.Has(childEntry) {
				// Schedule for later processing
				stageEntries.AddItem(childEntry)
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
			cachedSet := cachedDescendants.Get(updateIt).(algorithm.Vector)
			cachedSet.PushBack(txCit)
			tmpTx := mempool.MapTx.GetEntryByHash(txCit.TxRef.Hash)
			tmpTx.UpdateAncestorState(int64(updateIt.TxSize), 1, updateIt.SigOpCount, updateIt.Fee+btcutil.Amount(updateIt.FeeDelta))
		}
	}
	tmpTx := mempool.MapTx.GetEntryByHash(updateIt.TxRef.Hash)
	tmpTx.UpdateDescendantState(int64(modifySize), btcutil.Amount(modifyFee), int64(modifyCount))
}

// UpdateTransactionsFromBlock : hashesToUpdate is the set of transaction hashes from a disconnected block
// which has been re-added to the mempool. For each entry, look for descendants
// that are outside hashesToUpdate, and add fee/size information for such
// descendants to the parent. For each such descendant, also update the ancestor
// state to include the parent.
func (mempool *Mempool) UpdateTransactionsFromBlock(hashesToUpdate algorithm.Vector) {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()
	// For each entry in hashesToUpdate, store the set of in-mempool, but not
	// in-hashesToUpdate transactions, so that we don't have to recalculate
	// descendants when we come across a previously seen entry.
	mapMemPoolDescendantsToUpdate := algorithm.NewCacheMap(utils.CompareByHash)
	setAlreadyIncluded := set.New()
	hashesToUpdate.Each(func(item interface{}) bool {
		setAlreadyIncluded.Add(item)
		return true
	})

	// Iterate in reverse, so that whenever we are looking at at a transaction
	// we are sure that all in-mempool descendants have already been processed.
	// This maximizes the benefit of the descendant cache and guarantees that
	// setMemPoolChildren will be updated, an assumption made in
	// UpdateForDescendants.
	reverseArray := hashesToUpdate.ReverseArray()
	for _, v := range reverseArray {
		setChildren := set.New()
		txHash := v.(utils.Hash)

		if txEntry := mempool.MapTx.GetEntryByHash(txHash); txEntry != nil {
			allKeys := mempool.MapNextTx.GetAllKeys()
			for _, key := range allKeys {
				if CompareByRefOutPoint(key, refOutPoint{txHash, 0}) {
					continue
				}
				outPointPtr := key.(*model.OutPoint)
				if !outPointPtr.Hash.IsEqual(&txHash) {
					continue
				}
				childTx := mempool.MapNextTx.Get(key).(model.Tx)
				childTxHash := childTx.Hash
				if childTxPtr := mempool.MapTx.GetEntryByHash(childTxHash); childTxPtr != nil {
					setChildren.Add(childTxPtr)
					if has := setAlreadyIncluded.Has(childTxHash); !has {
						mempool.UpdateChild(txEntry, childTxPtr, true)
						mempool.UpdateParent(childTxPtr, txEntry, true)
					}
				}
			}
			mempool.UpdateForDescendants(txEntry, mapMemPoolDescendantsToUpdate, setAlreadyIncluded)
		}
	}
}

func (mempool *Mempool) UpdateChild(entry *TxMempoolEntry, entryChild *TxMempoolEntry, add bool) {
	s := set.New()
	hasTxLinks := mempool.MapLinks.Get(entry.TxRef.Hash)
	if hasTxLinks != nil {
		txLinks := hasTxLinks.(*TxLinks)
		children := txLinks.Children
		if add {
			if ok := children.AddItem(entryChild); ok {
				mempool.CachedInnerUsage += uint64(IncrementalDynamicUsageTxMempoolEntry(s))
			}
		} else {
			if ok := children.RemoveItem(entryChild); ok {
				mempool.CachedInnerUsage -= uint64(IncrementalDynamicUsageTxMempoolEntry(s))
			}
		}
	}
}

func (mempool *Mempool) UpdateParent(entry, entryParent *TxMempoolEntry, add bool) {
	s := set.New()
	hasTxLinks := mempool.MapLinks.Get(entry.TxRef.Hash)

	if hasTxLinks != nil {
		txLinks := hasTxLinks.(*TxLinks)
		parents := txLinks.Parents
		if add {
			if ok := parents.AddItem(entryParent); ok {
				mempool.CachedInnerUsage += uint64(IncrementalDynamicUsageTxMempoolEntry(s))
			}
		} else {
			if ok := parents.RemoveItem(entryParent); ok {
				mempool.CachedInnerUsage -= uint64(IncrementalDynamicUsageTxMempoolEntry(s))
			}
		}
	}
}

func (mempool *Mempool) GetMempoolChildren(entry *TxMempoolEntry) *algorithm.Set {
	result := mempool.MapLinks.Get(entry.TxRef.Hash)
	if result == nil {
		panic("No have children In mempool for this TxmempoolEntry")
	}
	return result.(*TxLinks).Children
}

func (mempool *Mempool) GetMemPoolParents(entry *TxMempoolEntry) *algorithm.Set {
	result := mempool.MapLinks.Get(entry.TxRef.Hash)
	if result == nil {
		panic("No have parant In mempool for this TxmempoolEntry")
	}
	return result.(*TxLinks).Parents
}

func (mempool *Mempool) GetMinFee(sizeLimit int64) *utils.FeeRate {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()
	if !mempool.BlockSinceLatRollingFeeBump || mempool.RollingMinimumFeeRate == 0 {
		return utils.NewFeeRate(int64(mempool.RollingMinimumFeeRate))
	}

	increRelFee := conf.GlobalValueInstance.GetIncrementalRelayFee()
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
		//if mempool.RollingMinimumFeeRate < float64(IncrementalRelayFee.GetFeePerK())/2 {
		if mempool.RollingMinimumFeeRate < float64(increRelFee.GetFeePerK())/2 {
			mempool.RollingMinimumFeeRate = 0
			return utils.NewFeeRate(0)
		}

	}
	result := int64(math.Max(mempool.RollingMinimumFeeRate, float64(increRelFee.SataoshisPerK)))
	return utils.NewFeeRate(result)
}

func (mempool *Mempool) TrackPackageRemoved(rate *utils.FeeRate) {
	if float64(rate.GetFeePerK()) > mempool.RollingMinimumFeeRate {
		mempool.RollingMinimumFeeRate = float64(rate.GetFeePerK())
		mempool.BlockSinceLatRollingFeeBump = false
	}
}

/*TrimToSize Remove transactions from the mempool until its dynamic size is <=
* sizelimit. pvNoSpendsRemaining, if set, will be populated with the list
* of outpoints which are not in mempool which no longer have any spends in
* this mempool.
 */
func (mempool *Mempool) TrimToSize(sizeLimit int64, pvNoSpendsRemaining *algorithm.Vector) error {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()
	txNRemoved := 0
	maxFeeRateRemoved := utils.NewFeeRate(0)
	for mempool.MapTx.Size() > 0 && mempool.DynamicMemoryUsage() > sizeLimit {
		it := mempool.MapTx.GetByDescendantScoreSortBegin().(*TxMempoolEntry)
		// We set the new mempool min fee to the feerate of the removed set,
		// plus the "minimum reasonable fee rate" (ie some value under which we
		// consider txn to have 0 fee). This way, we don't allow txn to enter
		// mempool with feerate equal to txn which were removed with no block in
		// between.
		removed := utils.NewFeeRateWithSize(int64(it.ModFeesWithDescendants), int(it.SizeWithDescendants))
		removed.SataoshisPerK += conf.GlobalValueInstance.GetIncrementalRelayFee().SataoshisPerK
		mempool.TrackPackageRemoved(removed)
		if removed.SataoshisPerK > maxFeeRateRemoved.SataoshisPerK {
			maxFeeRateRemoved = removed
		}
		stage := set.New()
		mempool.CalculateDescendants(it, stage)
		txNRemoved += stage.Size()
		txn := algorithm.NewVector()
		if pvNoSpendsRemaining != nil {
			for _, iter := range stage.List() {
				txMempoolEntryIter := iter.(TxMempoolEntry)
				txn.PushBack(txMempoolEntryIter.TxRef)
			}
		}

		mempool.RemoveStaged(stage, false, SIZELIMIT)
		if pvNoSpendsRemaining != nil {
			for _, t := range txn.Array {
				tx := t.(*model.Tx)
				for _, txin := range tx.Ins {
					if mempool.ExistsHash(txin.PreviousOutPoint.Hash) {
						continue
					}
					iter := mempool.MapNextTx.GetLowerBoundKey(refOutPoint{txin.PreviousOutPoint.Hash, 0})
					if iter == nil {
						pvNoSpendsRemaining.PushBack(txin.PreviousOutPoint)
					} else if oriHash := (iter.(refOutPoint).Hash); !(&oriHash).IsEqual(&txin.PreviousOutPoint.Hash) {
						pvNoSpendsRemaining.PushBack(txin.PreviousOutPoint)
					}
				}

			}
		}

	}

	if maxFeeRateRemoved.SataoshisPerK > 0 {
		return errors.Errorf("mempool removed %v txn , rolling minimum fee bumped tp %s \n", txNRemoved, maxFeeRateRemoved.String())
	}
	return nil
}

func (mempool *Mempool) DynamicMemoryUsage() int64 {
	entry := TxMempoolEntry{}

	return MallocUsage(int64(unsafe.Sizeof(&entry)+unsafe.Sizeof(utils.HashOne)))*int64(mempool.MapTx.Size()) +
		int64(len(mempool.vTxHashes)*int(unsafe.Sizeof(TxHash{}))) +
		int64(mempool.MapNextTx.Size()*int(unsafe.Sizeof(refOutPoint{})+unsafe.Sizeof(&entry))) +
		int64(len(mempool.MapDeltas)*int(unsafe.Sizeof(utils.HashOne)+unsafe.Sizeof(PriorityFeeDelta{}))) +
		int64(mempool.MapLinks.Count()*int(unsafe.Sizeof(utils.HashOne)+unsafe.Sizeof(&entry))) +
		int64(mempool.CachedInnerUsage)
}

func (mempool *Mempool) ExistsHash(hash utils.Hash) bool {
	return mempool.MapTx.GetEntryByHash(hash) != nil
}

func (mempool *Mempool) ExistsOutPoint(outpoint *model.OutPoint) bool {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()

	txMempoolEntry := mempool.MapTx.GetEntryByHash(outpoint.Hash)
	if txMempoolEntry == nil {
		return false
	}
	return int(outpoint.Index) < len(txMempoolEntry.TxRef.Outs)
}

func AllowFee(priority float64) bool {
	// Large (in bytes) low-priority (new, small-coin) transactions need a fee.
	return priority > AllowFreeThreshold()
}

func GetTxFromMemPool(hash utils.Hash) *model.Tx {
	return new(model.Tx)
}

func AllowFreeThreshold() float64 {
	return (float64(utils.COIN) * 144) / 250
}

//CalculateMemPoolAncestors try to calculate all in-mempool ancestors of entry.
// (these are all calculated including the tx itself)
// fSearchForParents = whether to search a tx's vin for in-mempool parents,
// or look up parents from mapLinks. Must be true for entries not in the mempool
// setAncestors element Type is *TxMempoolEntry;
func (mempool *Mempool) CalculateMemPoolAncestors(entry *TxMempoolEntry, setAncestors *set.Set,
	limitAncestorCount, limitAncestorSize, limitDescendantCount,
	limitDescendantSize uint64, fSearchForParents bool) error {

	//parentHashes element type is *TxMempoolEntry;
	parentHashes := algorithm.NewSet()
	tx := entry.TxRef

	if fSearchForParents {
		// Get parents of this transaction that are in the mempool
		// GetMemPoolParents() is only valid for entries in the mempool, so we
		// iterate mapTx to find parents.
		for _, txIn := range tx.Ins {
			if tx := mempool.MapTx.GetEntryByHash(txIn.PreviousOutPoint.Hash); tx != nil {
				parentHashes.AddItem(tx)
				if uint64(parentHashes.Size()+1) > limitAncestorCount {
					return errors.Errorf("too many unconfirmed parents [limit: %d]", limitAncestorCount)
				}
			}
		}
	} else {
		// If we're not searching for parents, we require this to be an entry in
		// the mempool already.
		if hasTx := mempool.MapTx.GetEntryByHash(entry.TxRef.Hash); hasTx != nil {
			parentHashes = mempool.GetMemPoolParents(hasTx)
		} else {
			panic("passed the entry is not in mempool")
		}
	}

	totalSizeWithAncestors := entry.TxRef.SerializeSize()
	parentList := parentHashes.List()
	for len(parentList) > 0 {
		stageit := parentList[0].(*TxMempoolEntry)
		setAncestors.Add(stageit)
		parentList = parentList[1:]
		totalSizeWithAncestors += stageit.TxSize

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
	err := mempool.CalculateMemPoolAncestors(entry, setAncestors, nNoLimit, nNoLimit, nNoLimit, nNoLimit, true)
	if err != nil {
		return false
	}

	return mempool.AddUncheckedWithAncestors(hash, entry, setAncestors, validFeeEstimate)
}

func (mempool *Mempool) AddUncheckedWithAncestors(hash *utils.Hash, entry *TxMempoolEntry, setAncestors *set.Set, validFeeEstimate bool) bool {

	//todo: add signal/slot In where for passed entry

	// Add to memory pool without checking anything.
	// Used by AcceptToMemoryPool(), which DOES do all the appropriate checks.
	mempool.MapTx.AddElement(*hash, entry)
	mempool.MapLinks.Set(entry.TxRef.Hash, &TxLinks{algorithm.NewSet(), algorithm.NewSet()})

	// Update transaction for any feeDelta created by PrioritiseTransaction mapTx
	if prioriFeeDelta, ok := mempool.MapDeltas[*hash]; ok {
		if prioriFeeDelta.fee != 0 {
			txEntry := mempool.MapTx.GetEntryByHash(*hash)
			txEntry.UpdateFeeDelta(int64(prioriFeeDelta.fee))
		}
	}

	// Update cachedInnerUsage to include contained transaction's usage.
	// (When we update the entry for in-mempool parents, memory usage will be
	// further updated.)
	mempool.CachedInnerUsage += uint64(entry.UsageSize)
	setParentTransactions := set.New()
	for _, txIn := range entry.TxRef.Ins {
		mempool.MapNextTx.Add(refOutPoint{txIn.PreviousOutPoint.Hash, txIn.PreviousOutPoint.Index}, *entry.TxRef)
		setParentTransactions.Add(txIn.PreviousOutPoint.Hash)
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
		if parentTx := mempool.MapTx.GetEntryByHash(hash); parentTx != nil {
			mempool.UpdateParent(entry, parentTx, true)
		}

		return true
	})
	mempool.UpdateAncestorsOf(true, entry, setAncestors)
	mempool.UpdateEntryForAncestors(entry, setAncestors)
	mempool.TransactionsUpdated++
	mempool.totalTxSize += uint64(entry.TxSize)
	mempool.MinerPolicyEstimator.ProcessTransaction(entry, validFeeEstimate)
	mempool.vTxHashes = append(mempool.vTxHashes, TxHash{entry.TxRef.Hash, entry})
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
		if txEntry := mempool.MapTx.GetEntryByHash(ancestorIt.TxRef.Hash); txEntry != nil {
			txEntry.UpdateDescendantState(int64(updateSize), updateFee, int64(updateCount))
		}

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
		updateSigOpsCount += ancestorIt.SigOpCountWithAncestors

		return true
	})

	entry.UpdateAncestorState(updateSize, int64(updateCount), updateSigOpsCount, updateFee)
}

func (mempool *Mempool) Size() int {
	mempool.mtx.RLock()
	size := mempool.MapTx.Size()
	mempool.mtx.RUnlock()
	return size
}

func (mempool *Mempool) GetTotalTxSize() uint64 {
	mempool.mtx.RLock()
	size := mempool.totalTxSize
	mempool.mtx.RUnlock()
	return size
}

func (mempool *Mempool) Exists(hash utils.Hash) bool {
	mempool.mtx.RLock()
	has := mempool.MapTx.GetEntryByHash(hash)
	mempool.mtx.RUnlock()

	return has != nil
}

func (mempool *Mempool) HasNoInputsOf(tx *model.Tx) bool {
	for i := 0; i < len(tx.Ins); i++ {
		if has := mempool.Exists(tx.Ins[i].PreviousOutPoint.Hash); has {
			return false
		}
	}
	return true
}

func (mempool *Mempool) RemoveConflicts(tx *model.Tx) {
	// Remove transactions which depend on inputs of tx, recursively
	for _, txIn := range tx.Ins {
		hasTx := mempool.MapNextTx.Get(refOutPoint{txIn.PreviousOutPoint.Hash, txIn.PreviousOutPoint.Index})
		if hasTx != nil {
			txConflict := hasTx.(model.Tx)
			if !txConflict.Equal(tx) {
				mempool.ClearPrioritisation(&txConflict.Hash)
				mempool.RemoveRecursive(&txConflict, CONFLICT)
			}
		}
	}
}

func (mempool *Mempool) QueryHashes(vtxid *algorithm.Vector) {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()

	iters := mempool.GetSortedDepthAndScore()
	vtxid.Clear()
	for _, entry := range iters {
		vtxid.PushBack(entry.TxRef.Hash)
	}
}

func (mempool *Mempool) WriteFeeEstimates(writer io.Writer) error {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()
	// version required to read: 0.13.99 or later
	err := binary.Write(writer, binary.LittleEndian, int(139900))
	if err != nil {
		return err
	}
	//todo write version that wrote the file
	err = mempool.MinerPolicyEstimator.Serialize(writer)
	return err
}

func (mempool *Mempool) ReadFeeEstimates(reader io.Reader) error {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()
	var versionRequired, versionThatWrote int
	err := binary.Read(reader, binary.LittleEndian, versionRequired)
	if err != nil {
		return err
	}

	//todo read version that read the file
	err = mempool.MinerPolicyEstimator.Deserialize(reader, versionThatWrote)
	return err
}

/*RemoveForBlock Called when a block is connected. Removes from mempool and updates the miner
 *fee estimator.*/
func (mempool *Mempool) RemoveForBlock(vtx *algorithm.Vector, blockHeight uint) {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()

	//entries element Type : *TxMempoolEntry;
	entries := algorithm.NewVector()
	vtx.Each(func(item interface{}) bool {
		txid := item.(*model.Tx).Hash
		if entry := mempool.MapTx.GetEntryByHash(txid); entry != nil {
			entries.PushBack(entry)
		}

		return true
	})

	// Before the txs in the new block have been removed from the mempool,
	// update policy estimates
	mempool.MinerPolicyEstimator.ProcessBlock(blockHeight, entries)
	vtx.Each(func(item interface{}) bool {
		txPtr := item.(*model.Tx)
		if entryPtr := mempool.MapTx.GetEntryByHash(txPtr.Hash); entryPtr != nil {
			stage := set.New()
			stage.Add(entryPtr)
			mempool.RemoveStaged(stage, true, BLOCK)
		}
		mempool.RemoveConflicts(txPtr)
		mempool.ClearPrioritisation(&txPtr.Hash)
		return true
	})

	mempool.LastRollingFeeUpdate = utils.GetMockTime()
	mempool.BlockSinceLatRollingFeeBump = true
}

func clear(mempool *Mempool) {
	mempool.MapLinks = beeUtils.NewBeeMap()
	mempool.MapTx = NewMultiIndex()
	mempool.MapNextTx = algorithm.NewCacheMap(CompareByRefOutPoint)
	mempool.vTxHashes = make([]TxHash, 0)
	mempool.MapDeltas = make(map[utils.Hash]PriorityFeeDelta)
	mempool.totalTxSize = 0
	mempool.CachedInnerUsage = 0
	mempool.LastRollingFeeUpdate = utils.GetMockTime()
	mempool.BlockSinceLatRollingFeeBump = false
	mempool.RollingMinimumFeeRate = 0
	mempool.TransactionsUpdated++
}

func (mempool *Mempool) Clear() {
	mempool.mtx.Lock()
	defer mempool.mtx.Unlock()
	clear(mempool)

}

func NewMemPool(minReasonableRelayFee utils.FeeRate) *Mempool {
	mempool := Mempool{}
	mempool.TransactionsUpdated = 0
	clear(&mempool)
	// Sanity checks off by default for performance, because otherwise accepting
	// transactions becomes O(N^2) where N is the number of transactions in the
	// pool
	mempool.CheckFrequency = 0
	mempool.MinerPolicyEstimator = NewBlockPolicyEstmator(minReasonableRelayFee)
	return &mempool
}

func (mempool *Mempool) Expire(time int64) int {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()
	//toremove element type is *TxMempoolEntry
	toremove := set.New()
	for _, txMempoolEntry := range mempool.MapTx.poolNode {
		if txMempoolEntry.Time < time {
			toremove.Add(txMempoolEntry)
		}
	}
	stage := set.New()
	for _, txiter := range toremove.List() {
		txMempoolEntry := txiter.(*TxMempoolEntry)
		mempool.CalculateDescendants(txMempoolEntry, stage)
	}
	mempool.RemoveStaged(stage, false, EXPIRY)

	return stage.Size()
}

func (mempool *Mempool) TransactionWithinChainLimit(txid utils.Hash, chainLimit uint64) bool {
	mempool.mtx.RLock()
	txMempoolEntry := mempool.MapTx.GetEntryByHash(txid)
	ret := txMempoolEntry.CountWithDescendants < chainLimit && txMempoolEntry.CountWithAncestors < chainLimit
	mempool.mtx.RUnlock()

	return ret
}

func (mempool *Mempool) EstimateSmartPriority(blocks int, answerFoundAtBlocks int) float64 {
	mempool.mtx.RLock()
	ret := mempool.MinerPolicyEstimator.EstimateSmartPriority(blocks, &answerFoundAtBlocks, mempool)
	mempool.mtx.RUnlock()
	return ret
}

func (mempool *Mempool) EstimatePriority(blocks int) float64 {
	mempool.mtx.RLock()
	ret := mempool.MinerPolicyEstimator.EstimatePriority(blocks)
	mempool.mtx.RUnlock()
	return ret
}

func (mempool *Mempool) EstimateSmartFee(blocks int, answerFoundAtBlocks int) utils.FeeRate {
	mempool.mtx.RLock()
	ret := mempool.MinerPolicyEstimator.EstimateSmartFee(blocks, &answerFoundAtBlocks, mempool)
	mempool.mtx.RUnlock()
	return ret
}

func (mempool *Mempool) EstimateFee(blocks int) utils.FeeRate {
	mempool.mtx.RLock()
	ret := mempool.MinerPolicyEstimator.EstimateFee(blocks)
	mempool.mtx.RUnlock()
	return ret
}

func (mempool *Mempool) InfoAll() []*TxMempoolEntry {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()
	iters := mempool.GetSortedDepthAndScore()
	ret := make([]*TxMempoolEntry, 0)
	ret = append(ret, iters...)
	return ret
}

func (mempool *Mempool) QueryHashs() []*utils.Hash {
	mempool.mtx.RLock()
	defer mempool.mtx.RUnlock()
	iters := mempool.GetSortedDepthAndScore()
	ret := make([]*utils.Hash, 0)
	for _, it := range iters {
		ret = append(ret, &it.TxRef.Hash)
	}
	return ret

}

func (mempool *Mempool) GetSortedDepthAndScore() []*TxMempoolEntry {
	iters := make([]*TxMempoolEntry, 0)
	for _, it := range mempool.MapTx.poolNode {
		iters = append(iters, it)
	}
	slice.Sort(iters[:], func(i, j int) bool {
		return DepthAndScoreComparator(iters[i], iters[j])

	})

	return iters
}
