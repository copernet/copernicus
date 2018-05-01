package mempool

import (
	"fmt"
	"math"
	"sync"

	"container/list"
	"encoding/binary"
	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
	"github.com/google/btree"
	"io"
	"unsafe"
)

type PoolRemovalReason int

// Reason why a transaction was removed from the memPool, this is passed to the
// * notification signal.
const (
	// UNKNOWN Manually removed or unknown reason
	UNKNOWN PoolRemovalReason = iota
	// EXPIRY Expired from memPool
	EXPIRY
	// SIZELIMIT Removed in size limiting
	SIZELIMIT
	// REORG Removed for reorganization
	REORG
	// BLOCK Removed for block
	BLOCK
	// CONFLICT Removed for conflict with in-block transaction
	CONFLICT
	// REPLACED Removed for replacement
	REPLACED
)

const MEMPOOL_HEIGHT = 0x7FFFFFFF

// TxMempool is safe for concurrent write And read access.
type TxMempool struct {
	sync.RWMutex
	// current mempool best feerate for one transaction.
	Fee utils.FeeRate
	// poolData store the tx in the mempool
	PoolData map[utils.Hash]*TxEntry
	//NextTx key is txPreout, value is tx.
	NextTx map[core.OutPoint]*TxEntry
	//RootTx contain all root transaction in mempool.
	rootTx                  map[utils.Hash]*TxEntry
	TxByAncestorFeeRateSort btree.BTree
	timeSortData            btree.BTree
	cacheInnerUsage         int64
	checkFrequency          float64
	// sum of all mempool tx's size.
	totalTxSize uint64
	//transactionsUpdated mempool update transaction total number when create mempool late.
	transactionsUpdated uint64
}

func (m *TxMempool) GetCacheUsage() int64 {
	m.RLock()
	defer m.RUnlock()
	return m.cacheInnerUsage
}

func (m *TxMempool) GetCheckFreQuency() float64 {
	m.RLock()
	defer m.RUnlock()
	return m.checkFrequency
}

func (m *TxMempool) GetCoin(outpoint *core.OutPoint) *utxo.Coin {
	m.RLock()
	defer m.RUnlock()

	txMempoolEntry, ok := m.PoolData[outpoint.Hash]
	if !ok {
		return nil
	}

	if int(outpoint.Index) < len(txMempoolEntry.Tx.Outs) {
		coin := utxo.NewCoin(txMempoolEntry.Tx.Outs[outpoint.Index], MEMPOOL_HEIGHT, false)
		return coin
	}

	return nil
}

// Check If sanity-checking is turned on, check makes sure the pool is consistent
// (does not contain two transactions that spend the same inputs, all inputs
// are in the mapNextTx array). If sanity-checking is turned off, check does
// nothing.
func (m *TxMempool) Check(coins *utxo.CoinsViewCache, bestHeight int) {
	if m.GetCheckFreQuency() == 0 {
		return
	}
	if float64(utils.GetRand(math.MaxUint32)) >= m.GetCheckFreQuency() {
		return
	}

	logs.SetLogger("mempool", fmt.Sprintf("checking mempool with %d transaction and %d inputs ...", len(m.PoolData), len(m.NextTx)))
	checkTotal := uint64(0)
	m.Lock()
	defer m.Unlock()

	waitingOnDependants := list.New()
	for _, entry := range m.PoolData {

		checkTotal += uint64(entry.TxSize)
		fDependsWait := false
		setParentCheck := make(map[utils.Hash]struct{})

		for _, txin := range entry.Tx.Ins {
			if entry, ok := m.PoolData[txin.PreviousOutPoint.Hash]; ok {
				tx2 := entry.Tx
				if !(len(tx2.Outs) > int(txin.PreviousOutPoint.Index) &&
					!tx2.Outs[txin.PreviousOutPoint.Index].IsNull()) {
					panic("the tx introduced input dose not exist, or the input amount is nil ")
				}
				fDependsWait = true
				setParentCheck[tx2.Hash] = struct{}{}
			} else {
				if !coins.HaveCoin(txin.PreviousOutPoint) {
					panic("the tx introduced input dose not exist mempool And UTXO set !!!")
				}
			}

			if _, ok := m.NextTx[*txin.PreviousOutPoint]; !ok {
				panic("the introduced tx is not in mempool")
			}
		}
		if len(setParentCheck) != len(entry.ParentTx) {
			panic("the two parent set should be equal")
		}

		// Verify ancestor state is correct.
		nNoLimit := uint64(math.MaxUint64)
		setAncestors, err := m.CalculateMemPoolAncestors(entry.Tx, nNoLimit, nNoLimit, nNoLimit, nNoLimit, true)
		if err != nil {
			return
		}
		nCountCheck := int64(len(setAncestors)) + 1
		nSizeCheck := int64(entry.TxSize)
		nSigOpCheck := int64(entry.SigOpCount)
		nFeesCheck := entry.TxFee
		for ancestorIt := range setAncestors {
			nSizeCheck += int64(ancestorIt.TxSize)
			nSigOpCheck += int64(ancestorIt.SigOpCount)
			nFeesCheck += ancestorIt.TxFee
		}
		if entry.SumTxCountWithAncestors != nCountCheck {
			panic("the txentry's ancestors number is incorrect .")
		}
		if entry.SumSizeWitAncestors != nSizeCheck {
			panic("the txentry's ancestors size is incorrect .")
		}
		if entry.SumSigOpCountWithAncestors != nSigOpCheck {
			panic("the txentry's ancestors sigopcount is incorrect .")
		}
		if entry.SumFeeWithAncestors != nFeesCheck {
			panic("the txentry's ancestors fee is incorrect .")
		}

		setChildrenCheck := make(map[*TxEntry]struct{})
		childSize := 0
		for i := range entry.Tx.Outs {
			o := core.OutPoint{Hash: entry.Tx.Hash, Index: uint32(i)}
			if e, ok := m.NextTx[o]; ok {
				if _, ok := m.PoolData[e.Tx.Hash]; !ok {
					panic("the transaction should be in mempool ...")
				}
				if _, ok := setChildrenCheck[e]; !ok {
					setChildrenCheck[e] = struct{}{}
					childSize += e.TxSize
				}
			}
		}

		if len(setChildrenCheck) != len(entry.ChildTx) {
			panic("the transaction children set is different ...")
		}
		if entry.SumSizeWithDescendants < int64(childSize+entry.TxSize) {
			panic("the transaction descendant's fee is less its children fee ...")
		}

		// Also check to make sure size is greater than sum with immediate
		// children. Just a sanity check, not definitive that this calc is
		// correct...
		if fDependsWait {
			waitingOnDependants.PushBack(entry)
		} else {
			var state core.ValidationState
			fCheckResult := entry.Tx.IsCoinBase() ||
				coins.CheckTxInputs(entry.Tx, &state, bestHeight)
			if !fCheckResult {
				panic("the txentry check failed with utxo set...")
			}
			coins.UpdateCoins(entry.Tx, 1000000)
		}
	}
	stepsSinceLastRemove := 0
	for waitingOnDependants.Len() > 0 {
		it := waitingOnDependants.Front()
		entry := it.Value.(*TxEntry)
		waitingOnDependants.Remove(it)
		if !coins.HaveInputs(entry.Tx) {
			waitingOnDependants.PushBack(entry)
			stepsSinceLastRemove++
			if !(stepsSinceLastRemove < waitingOnDependants.Len()) {
				panic("the waitingOnDependants list have incorrect number ...")
			}
		} else {
			fCheckResult := entry.Tx.IsCoinBase() ||
				coins.CheckTxInputs(entry.Tx, nil, bestHeight)
			if !fCheckResult {
				panic("")
			}
			coins.UpdateCoins(entry.Tx, 1000000)
			stepsSinceLastRemove = 0
		}
	}

	for _, entry := range m.NextTx {
		txid := entry.Tx.Hash
		if e, ok := m.PoolData[txid]; !ok {
			panic("the transaction not exsit mempool. . .")
		} else {
			if e.Tx != entry.Tx {
				panic("mempool store the transaction is different with it's two struct . . .")
			}
		}
	}

	if m.totalTxSize != checkTotal {
		panic("mempool have all transaction size state is incorrect ...")
	}
}

// RemoveTxSelf will only remove these transaction self.
func (m *TxMempool) RemoveTxSelf(txs []*core.Tx) {
	m.Lock()
	defer m.Unlock()

	entries := make([]*TxEntry, 0, len(txs))
	for _, tx := range txs {
		if entry, ok := m.PoolData[tx.Hash]; ok {
			entries = append(entries, entry)
		}
	}

	// todo base on entries to set the new feerate for mempool.

	for _, tx := range txs {
		if entry, ok := m.PoolData[tx.Hash]; ok {
			stage := make(map[*TxEntry]struct{})
			stage[entry] = struct{}{}
			m.RemoveStaged(stage, true, BLOCK)
		}
		m.removeConflicts(tx)
	}
}

// AddTx operator is safe for concurrent write And read access.
// this function is used to add tx to the memPool, and now the tx should
// be passed all appropriate checks.
func (m *TxMempool) AddTx(txentry *TxEntry, limitAncestorCount uint64,
	limitAncestorSize uint64, limitDescendantCount uint64, limitDescendantSize uint64, searchForParent bool) error {
	// todo: send signal to all interesting the caller.
	m.Lock()
	defer m.Unlock()
	ancestors, err := m.CalculateMemPoolAncestors(txentry.Tx, limitAncestorCount, limitAncestorSize, limitDescendantCount, limitDescendantSize, searchForParent)
	if err != nil {
		return err
	}

	// insert new txEntry to the memPool; and update the memPool's memory consume.
	m.timeSortData.ReplaceOrInsert(txentry)
	m.PoolData[txentry.Tx.Hash] = txentry
	m.cacheInnerUsage += int64(txentry.usageSize) + int64(unsafe.Sizeof(txentry))

	// Update ancestors with information about this tx
	setParentTransactions := make(map[utils.Hash]struct{})
	tx := txentry.Tx
	for _, txin := range tx.Ins {
		m.NextTx[*txin.PreviousOutPoint] = txentry
		setParentTransactions[txin.PreviousOutPoint.Hash] = struct{}{}
	}

	for hash := range setParentTransactions {
		if parent, ok := m.PoolData[hash]; ok {
			txentry.UpdateParent(parent, &m.cacheInnerUsage, true)
		}
	}

	m.updateAncestorsOf(true, txentry, ancestors)
	m.UpdateEntryForAncestors(txentry, ancestors)
	m.totalTxSize += uint64(txentry.TxSize)
	m.transactionsUpdated++
	m.TxByAncestorFeeRateSort.ReplaceOrInsert(EntryAncestorFeeRateSort(*txentry))
	if txentry.SumTxCountWithAncestors == 1 {
		m.rootTx[txentry.Tx.Hash] = txentry
	}
	return nil
}

// LimitMempoolSize limit mempool size with time And limit size. when the noSpendsRemaining
// set, the function will return these have removed transaction's txin from mempool which use
// TrimToSize rule. Later, caller will remove these txin from uxto cache.
func (m *TxMempool) LimitMempoolSize() []*core.OutPoint {
	//todo, parse expire time from config
	m.Expire(0)
	//todo, parse limit mempoolsize from config
	c := m.TrimToSize(0)
	return c
}

// TrimToSize Remove transactions from the mempool until its dynamic size is <=
// sizelimit. noSpendsRemaining, if set, will be populated with the list
// of outpoints which are not in mempool which no longer have any spends in
// this mempool.
func (m *TxMempool) TrimToSize(sizeLimit int64) []*core.OutPoint {
	m.Lock()
	defer m.Unlock()

	nTxnRemoved := 0
	ret := make([]*core.OutPoint, 0)
	maxFeeRateRemove := int64(0)

	for len(m.PoolData) > 0 && m.cacheInnerUsage > sizeLimit {
		removeIt := TxEntry(m.TxByAncestorFeeRateSort.Min().(EntryAncestorFeeRateSort))
		rem := m.TxByAncestorFeeRateSort.Delete(EntryAncestorFeeRateSort(removeIt)).(EntryAncestorFeeRateSort)
		if rem.Tx.Hash != removeIt.Tx.Hash {
			panic("the two element should have the same Txhash")
		}
		maxFeeRateRemove = utils.NewFeeRateWithSize(removeIt.SumFeeWithDescendants, removeIt.SumSizeWithDescendants).SataoshisPerK
		stage := make(map[*TxEntry]struct{})
		m.CalculateDescendants(&removeIt, stage)
		nTxnRemoved += len(stage)
		txn := make([]*core.Tx, 0, len(stage))
		for iter := range stage {
			txn = append(txn, iter.Tx)
		}

		// here, don't update Descendant transaction state's reason :
		// all Descendant transaction of the removed tx also will be removed.
		m.RemoveStaged(stage, false, SIZELIMIT)
		for e := range stage {
			fmt.Printf("remove tx hash : %s, mempool size : %d\n", e.Tx.Hash.ToString(), m.cacheInnerUsage)
		}
		for _, tx := range txn {
			for _, txin := range tx.Ins {
				if _, ok := m.PoolData[txin.PreviousOutPoint.Hash]; ok {
					continue
				}
				if _, ok := m.NextTx[*txin.PreviousOutPoint]; !ok {
					ret = append(ret, txin.PreviousOutPoint)
				}
			}
		}

	}

	logs.SetLogger("mempool", fmt.Sprintf("removed %d txn, rolling minimum fee bumped : %d", nTxnRemoved, maxFeeRateRemove))
	return ret
}

// Expire all transaction (and their dependencies) in the memPool older
// than time. Return the number of removed transactions.
func (m *TxMempool) Expire(time int64) int {
	m.Lock()
	defer m.Unlock()
	toremove := make(map[*TxEntry]struct{}, 100)
	m.timeSortData.Ascend(func(i btree.Item) bool {
		entry := i.(*TxEntry)
		if entry.time < time {
			toremove[entry] = struct{}{}
			return true
		}
		return false
	})

	stage := make(map[*TxEntry]struct{}, len(toremove)*3)
	for removeIt := range toremove {
		m.CalculateDescendants(removeIt, stage)
	}
	m.RemoveStaged(stage, false, EXPIRY)
	return len(stage)
}

func (m *TxMempool) FindTx(hash utils.Hash) *core.Tx {
	m.RLock()
	m.RUnlock()
	if find, ok := m.PoolData[hash]; ok {
		return find.Tx
	}
	return nil
}

func (m *TxMempool) Exists(hash utils.Hash) bool {
	has := m.FindTx(hash)
	return has != nil
}

// HasNoInputsOf Check that none of this transactions inputs are in the memPool,
// and thus the tx is not dependent on other memPool transactions to be included
// in a block.
func (m *TxMempool) HasNoInputsOf(tx *core.Tx) bool {
	m.RLock()
	defer m.RUnlock()

	for _, txin := range tx.Ins {
		if m.FindTx(txin.PreviousOutPoint.Hash) != nil {
			return false
		}
	}
	return true
}

func (m *TxMempool) GetRootTx() map[utils.Hash]TxEntry {
	m.RLock()
	defer m.RUnlock()

	n := make(map[utils.Hash]TxEntry)
	for k, v := range m.rootTx {
		n[k] = *v
	}
	return n
}

func (m *TxMempool) updateForRemoveFromMempool(entriesToRemove map[*TxEntry]struct{}, updateDescendants bool) {
	nNoLimit := uint64(math.MaxUint64)

	if updateDescendants {
		for removeIt := range entriesToRemove {
			setDescendants := make(map[*TxEntry]struct{})
			m.CalculateDescendants(removeIt, setDescendants)
			delete(setDescendants, removeIt)
			modifySize := -removeIt.TxSize
			modifyFee := -removeIt.TxFee
			modifySigOps := -removeIt.SigOpCount

			for dit := range setDescendants {
				dit.UpdateAncestorState(-1, modifySize, modifySigOps, modifyFee)
			}
		}
	}

	for removeIt := range entriesToRemove {
		for updateIt := range removeIt.ChildTx {
			updateIt.UpdateParent(removeIt, &m.cacheInnerUsage, false)
		}
	}

	for removeIt := range entriesToRemove {
		ancestors, err := m.CalculateMemPoolAncestors(removeIt.Tx, nNoLimit, nNoLimit, nNoLimit, nNoLimit, false)
		if err != nil {
			return
		}
		m.updateAncestorsOf(false, removeIt, ancestors)
	}
}

func (m *TxMempool) RemoveStaged(entriesToRemove map[*TxEntry]struct{}, updateDescendants bool, reason PoolRemovalReason) {
	m.updateForRemoveFromMempool(entriesToRemove, updateDescendants)
	for rem := range entriesToRemove {
		if _, ok := m.rootTx[rem.Tx.Hash]; ok {
			delete(m.rootTx, rem.Tx.Hash)
		}
		m.delTxentry(rem, reason)
		fmt.Println("remove one transaction late, the mempool size : ", m.cacheInnerUsage)
	}
}

func (m *TxMempool) removeConflicts(tx *core.Tx) {
	// Remove transactions which depend on inputs of tx, recursively
	for _, txin := range tx.Ins {
		if flictEntry, ok := m.NextTx[*txin.PreviousOutPoint]; ok {
			if flictEntry.Tx.Hash != tx.Hash {
				m.RemoveTxRecursive(flictEntry.Tx, CONFLICT)
			}
		}
	}
}

// RemoveTxRecursive remove this transaction And its all descent transaction from mempool.
func (m *TxMempool) RemoveTxRecursive(origTx *core.Tx, reason PoolRemovalReason) {
	// Remove transaction from memory pool
	txToRemove := make(map[*TxEntry]struct{})

	if entry, ok := m.PoolData[origTx.Hash]; ok {
		txToRemove[entry] = struct{}{}
	} else {
		// When recursively removing but origTx isn't in the mempool be sure
		// to remove any children that are in the pool. This can happen
		// during chain re-orgs if origTx isn't re-accepted into the mempool
		// for any reason.
		for i := range origTx.Outs {
			outPoint := core.OutPoint{Hash: origTx.Hash, Index: uint32(i)}
			if en, ok := m.NextTx[outPoint]; !ok {
				continue
			} else {
				if find, ok := m.PoolData[en.Tx.Hash]; ok {
					txToRemove[find] = struct{}{}
				} else {
					panic("the transaction must in mempool, because NextTx struct of mempool have its data")
				}
			}
		}
	}
	allRemoves := make(map[*TxEntry]struct{})
	for it := range txToRemove {
		m.CalculateDescendants(it, allRemoves)
	}
	m.RemoveStaged(allRemoves, false, reason)
}

// CalculateDescendants Calculates descendants of entry that are not already in setDescendants, and
// adds to setDescendants. Assumes entry it is already a tx in the mempool and
// setMemPoolChildren is correct for tx and all descendants. Also assumes that
// if an entry is in setDescendants already, then all in-mempool descendants of
// it are already in setDescendants as well, so that we can save time by not
// iterating over those entries.
func (m *TxMempool) CalculateDescendants(entry *TxEntry, descendants map[*TxEntry]struct{}) {
	//stage := make(map[*TxEntry]struct{})
	stage := make([]*TxEntry, 0)
	if _, ok := descendants[entry]; !ok {
		stage = append(stage, entry)
	}

	// Traverse down the children of entry, only adding children that are not
	// accounted for in setDescendants already (because those children have
	// either already been walked, or will be walked in this iteration).
	for len(stage) > 0 {
		desEntry := stage[0]
		descendants[desEntry] = struct{}{}
		stage = stage[1:]

		for child := range desEntry.ChildTx {
			if _, ok := descendants[child]; !ok {
				stage = append(stage, child)
			}
		}
	}
}

// updateAncestorsOf update each of ancestors transaction state; add or remove this
// txentry txfee, txsize, txcount.
func (m *TxMempool) updateAncestorsOf(add bool, txentry *TxEntry, ancestors map[*TxEntry]struct{}) {

	// update the parent's child transaction set;
	for piter := range txentry.ParentTx {
		if add {
			piter.UpdateChild(txentry, &m.cacheInnerUsage, true)
		} else {
			fmt.Println("tx will romove tx3's from its'parent, tx3 : ", txentry.Tx.Hash.ToString(), ", tx1 : ", piter.Tx.Hash.ToString())
			piter.UpdateChild(txentry, &m.cacheInnerUsage, false)
		}
	}

	updateCount := -1
	if add {
		updateCount = 1
	}
	updateSize := updateCount * txentry.TxSize
	updateFee := int64(updateCount) * txentry.TxFee
	// update each of ancestors transaction state;
	for ancestorit := range ancestors {
		//fmt.Println("ancestor hash : ", ancestorit.Tx.Hash.ToString())
		ancestorit.UpdateDescendantState(updateCount, updateSize, updateFee)
	}
}

func (m *TxMempool) UpdateEntryForAncestors(entry *TxEntry, setAncestors map[*TxEntry]struct{}) {
	updateCount := len(setAncestors)
	updateSize := 0
	updateFee := int64(0)
	updateSigOpsCount := 0

	for ancestorIt := range setAncestors {
		updateFee += ancestorIt.TxFee
		updateSigOpsCount += ancestorIt.SigOpCount
		updateSize += ancestorIt.TxSize
	}
	entry.UpdateAncestorState(updateCount, updateSize, updateSigOpsCount, updateFee)
}

// CalculateMemPoolAncestors get tx all ancestors transaction in mempool.
// when the find is false: the tx must in mempool, so directly get his parent.
func (m *TxMempool) CalculateMemPoolAncestors(tx *core.Tx, limitAncestorCount uint64,
	limitAncestorSize uint64, limitDescendantCount uint64, limitDescendantSize uint64,
	searchForParent bool) (ancestors map[*TxEntry]struct{}, err error) {

	ancestors = make(map[*TxEntry]struct{})
	parent := make(map[*TxEntry]struct{})
	if searchForParent {
		for _, txin := range tx.Ins {
			if entry, ok := m.PoolData[txin.PreviousOutPoint.Hash]; ok {
				parent[entry] = struct{}{}
				if uint64(len(parent))+1 > limitAncestorCount {
					return nil,
						fmt.Errorf("too many unconfirmed parents [limit: %d]", limitAncestorCount)
				}
			}
		}
	} else {
		// If we're not searching for parents, we require this to be an entry in
		// the mempool already.
		if entry, ok := m.PoolData[tx.Hash]; ok {
			parent = entry.ParentTx
		} else {
			panic("the tx must be in mempool")
		}
	}

	totalSizeWithAncestors := int64(tx.SerializeSize())
	paSLice := make([]*TxEntry, len(parent))
	j := 0
	for entry := range parent {
		paSLice[j] = entry
		j++
	}

	for len(paSLice) > 0 {
		entry := paSLice[0]
		paSLice = paSLice[1:]
		//delete(parent, entry)
		ancestors[entry] = struct{}{}
		totalSizeWithAncestors += int64(entry.TxSize)
		hash := entry.Tx.Hash
		if uint64(entry.SumSizeWithDescendants+int64(entry.TxSize)) > limitDescendantSize {
			return nil,
				fmt.Errorf("exceeds descendant size limit for tx %s [limit: %d]", hash.ToString(), limitDescendantSize)
		} else if uint64(entry.SumTxCountWithDescendants+1) > limitDescendantCount {
			return nil,
				fmt.Errorf("too many descendants for tx %s [limit: %d]", hash.ToString(), limitDescendantCount)
		} else if uint64(totalSizeWithAncestors) > limitAncestorSize {
			return nil,
				fmt.Errorf("exceeds ancestor size limit [limit: %d]", limitAncestorSize)
		}

		graTxentrys := entry.ParentTx
		for gentry := range graTxentrys {
			if _, ok := ancestors[gentry]; !ok {
				paSLice = append(paSLice, gentry)
			}
			if uint64(len(parent)+len(ancestors)+1) > limitAncestorCount {
				return nil,
					fmt.Errorf("too many unconfirmed ancestors [limit: %d]", limitAncestorCount)
			}
		}
	}

	return ancestors, nil
}

func (m *TxMempool) delTxentry(removeEntry *TxEntry, reason PoolRemovalReason) {
	// todo add signal for any subscriber

	for _, txin := range removeEntry.Tx.Ins {
		delete(m.NextTx, *txin.PreviousOutPoint)
	}

	if _, ok := m.rootTx[removeEntry.Tx.Hash]; ok {

	}
	m.cacheInnerUsage -= int64(removeEntry.usageSize) + int64(unsafe.Sizeof(removeEntry))
	m.transactionsUpdated++
	m.totalTxSize -= uint64(removeEntry.TxSize)
	delete(m.PoolData, removeEntry.Tx.Hash)
	m.timeSortData.Delete(removeEntry)
	m.TxByAncestorFeeRateSort.Delete(EntryAncestorFeeRateSort(*removeEntry))
}

func (m *TxMempool) UpdateTransactionsFromBlock(ele interface{}) {
	//todo implement, this function is called in DisconnectTip()
}

func (m *TxMempool) InfoAll() []*TxMempoolInfo {
	m.RLock()
	defer m.RUnlock()

	ret := make([]*TxMempoolInfo, len(m.PoolData))
	index := 0
	m.TxByAncestorFeeRateSort.Ascend(func(i btree.Item) bool {
		entry := TxEntry(i.(EntryAncestorFeeRateSort))
		ret[index] = entry.GetInfo()
		index++
		return true
	})

	return ret
}

func (m *TxMempool) Size() int {
	m.RLock()
	defer m.RUnlock()

	return len(m.PoolData)
}

func NewTxMempool() *TxMempool {
	t := &TxMempool{}
	t.NextTx = make(map[core.OutPoint]*TxEntry)
	t.PoolData = make(map[utils.Hash]*TxEntry)
	t.timeSortData = *btree.New(32)
	t.rootTx = make(map[utils.Hash]*TxEntry)
	t.TxByAncestorFeeRateSort = *btree.New(32)
	return t
}

type TxMempoolInfo struct {
	Tx       *core.Tx      // The transaction itself
	Time     int64         // Time the transaction entered the memPool
	FeeRate  utils.FeeRate // FeeRate of the transaction
	FeeDelta int64         // The fee delta
}

func (info *TxMempoolInfo) Serialize(w io.Writer) error {
	err := info.Tx.Serialize(w)
	if err != nil {
		return err
	}

	err = utils.BinarySerializer.PutUint64(w, binary.LittleEndian, uint64(info.Time))
	if err != nil {
		return err
	}

	err = utils.BinarySerializer.PutUint64(w, binary.LittleEndian, uint64(info.FeeDelta))
	return err
}

func DeserializeInfo(r io.Reader) (*TxMempoolInfo, error) {
	tx, err := core.DeserializeTx(r)
	if err != nil {
		return nil, err
	}

	Time, err := utils.BinarySerializer.Uint64(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}

	FeeDelta, err := utils.BinarySerializer.Uint64(r, binary.LittleEndian)
	if err != nil {
		return nil, err
	}

	return &TxMempoolInfo{
		Tx:       tx,
		Time:     int64(Time),
		FeeDelta: int64(FeeDelta),
	}, nil
}
