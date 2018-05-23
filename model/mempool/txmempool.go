package mempool

import (
	"container/list"
	"fmt"
	"math"
	"sync"
	"time"
	"unsafe"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/util"
	"github.com/google/btree"
)

// error status for the transaction that entering txmempool.
const (
	RejectAlreadyKnown = 300
	RejectNonStandard  = 301

	ManyUnconfirmedAncestor       = 302
	ExceedUnconfirmedAncestorSize = 303
	ManyUnconfirmedDescendt       = 304
	ExceedUnconfirmedDescendtSize = 305
	LowTransactionFee             = 306
)

var Gpool *TxMempool

const (
	AncestorSize   uint64 = 101
	AncestorNum           = 25
	DescendantNum         = 25
	DescendantSize        = 101
)

type PoolRemovalReason int

// Reason why a transaction was removed from the memPool, this is passed to the
// notification signal.
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

// TxMempool is safe for concurrent write And read access.
type TxMempool struct {
	sync.RWMutex
	// current mempool best feerate for one transaction.
	feeRate util.FeeRate
	// poolData store the tx in the mempool
	poolData map[util.Hash]*TxEntry
	//NextTx key is txPrevout, value is tx.
	nextTx map[outpoint.OutPoint]*TxEntry
	//RootTx contain all root transaction in mempool.
	rootTx                  map[util.Hash]*TxEntry
	txByAncestorFeeRateSort btree.BTree
	timeSortData            btree.BTree
	cacheInnerUsage         int64
	checkFrequency          float64
	// sum of all mempool tx's size.
	totalTxSize uint64
	//transactionsUpdated mempool update transaction total number when create mempool late.
	TransactionsUpdated      uint64
	OrphanTransactionsByPrev map[outpoint.OutPoint]map[util.Hash]OrphanTx
	OrphanTransactions       map[util.Hash]OrphanTx
	RecentRejects            map[util.Hash]struct{}

	nextSweep      int
	MaxMemPoolSize int64
}

func (m *TxMempool) GetMinFeeRate() util.FeeRate {
	return m.feeRate
}

// AddTx operator is safe for concurrent write And read access.
// this function is used to add tx to the memPool, and now the tx should
// be passed all appropriate checks.
func (m *TxMempool) AddTx(txentry *TxEntry, ancestors map[*TxEntry]struct{}) error {
	// todo: send signal to all interesting the caller.
	m.Lock()
	defer m.Unlock()

	// insert new txEntry to the memPool; and update the memPool's memory consume.
	m.timeSortData.ReplaceOrInsert(txentry)
	m.poolData[txentry.Tx.GetHash()] = txentry
	m.cacheInnerUsage += int64(txentry.usageSize) + int64(unsafe.Sizeof(txentry))

	// Update ancestors with information about this tx
	setParentTransactions := make(map[util.Hash]struct{})
	tx := txentry.Tx
	for _, preout := range tx.GetAllPreviousOut() {
		m.nextTx[preout] = txentry
		setParentTransactions[preout.Hash] = struct{}{}
	}

	for hash := range setParentTransactions {
		if parent, ok := m.poolData[hash]; ok {
			txentry.UpdateParent(parent, &m.cacheInnerUsage, true)
		}
	}

	m.updateAncestorsOf(true, txentry, ancestors)
	m.updateEntryForAncestors(txentry, ancestors)
	m.totalTxSize += uint64(txentry.TxSize)
	m.TransactionsUpdated++
	m.txByAncestorFeeRateSort.ReplaceOrInsert(EntryAncestorFeeRateSort(*txentry))
	if txentry.SumTxCountWithAncestors == 1 {
		m.rootTx[txentry.Tx.GetHash()] = txentry
	}
	return nil
}

func (m *TxMempool) HasSpentOut(out *outpoint.OutPoint) bool {
	m.RLock()
	defer m.RUnlock()

	if _, ok := m.nextTx[*out]; ok {
		return true
	}
	return false
}

func (m *TxMempool) GetPoolAllTxSize() uint64 {
	m.RLock()
	size := m.totalTxSize
	m.RUnlock()
	return size
}

func (m *TxMempool) GetPoolUsage() int64 {
	m.RLock()
	size := m.cacheInnerUsage
	m.RUnlock()
	return size
}

func (m *TxMempool) CalculateDescendants(txHash *util.Hash) map[*TxEntry]struct{} {
	descendants := make(map[*TxEntry]struct{})
	m.RLock()
	defer m.RUnlock()
	if entry, ok := m.poolData[*txHash]; !ok {
		return nil
	} else {
		m.calculateDescendants(entry, descendants)
	}

	return descendants
}

func (m *TxMempool) CalculateMemPoolAncestors(txhash *util.Hash) map[*TxEntry]struct{} {
	m.RLock()
	defer m.RUnlock()
	ancestors := make(map[*TxEntry]struct{})
	if entry, ok := m.poolData[*txhash]; !ok {
		return nil
	} else {
		noLimit := uint64(math.MaxUint64)
		ancestors, _ = m.calculateMemPoolAncestors(entry.Tx, noLimit, noLimit, noLimit, noLimit, false)
	}
	return ancestors
}

func (m *TxMempool) RemoveTxRecursive(origTx *tx.Tx, reason PoolRemovalReason) {
	m.Lock()
	defer m.Unlock()
	m.removeTxRecursive(origTx, reason)
}

func (m *TxMempool) GetRootTx() map[util.Hash]TxEntry {
	m.RLock()
	defer m.RUnlock()

	n := make(map[util.Hash]TxEntry)
	for k, v := range m.rootTx {
		n[k] = *v
	}
	return n
}

func (m *TxMempool) Size() int {
	m.RLock()
	defer m.RUnlock()

	return len(m.poolData)
}

func (m *TxMempool) GetAllTxEntry() map[util.Hash]*TxEntry {
	m.RLock()
	ret := make(map[util.Hash]*TxEntry, len(m.poolData))
	for k, v := range m.poolData{
		ret[k] = v
	}
	m.RUnlock()
	return ret
}

func (m *TxMempool) RemoveUnFinalTx(chain *chain.Chain, view *utxo.CoinsCache, nMemPoolHeight int, flag int) {
	m.Lock()
	defer m.Unlock()

	// Remove transactions spending a coinbase which are now immature and
	// no-longer-final transactions
	txToRemove := make(map[*TxEntry]struct{})
	tip := chain.Tip()
	for _, entry := range m.poolData {
		lp := entry.GetLockPointFromTxEntry()
		validLP := entry.CheckLockPointValidity(chain)
		//state := NewValidationState()

		tx := entry.Tx
		allPreout := tx.GetAllPreviousOut()
		coins := make([]*utxo.Coin, len(allPreout))
		for i, preout := range allPreout {
			if coin := view.GetCoin(&preout); coin != nil {
				coins[i] = coin
			} else {
				if coin := m.GetCoin(&preout); coin != nil {
					coins[i] = coin
				} else {
					panic("the transaction in mempool, not found its parent " +
						"transaction in local node and utxo")
				}
			}
		}
		if !tx.ContextualCheckTransaction(flag) ||
			!checkSequenceLocks(tx, tip, flag, &lp, validLP, coins) {
			txToRemove[entry] = struct{}{}
		} else if entry.GetSpendsCoinbase() {
			for _, preout := range tx.GetAllPreviousOut() {
				if _, ok := m.poolData[preout.Hash]; ok {
					continue
				}

				coin := view.GetCoin(&preout)
				if m.checkFrequency != 0 {
					if coin.IsSpent() {
						panic("the coin must be unspent")
					}
				}

				if coin.IsSpent() || (coin.IsCoinBase() &&
					uint32(nMemPoolHeight)-coin.GetHeight() < consensus.CoinbaseMaturity) {
					txToRemove[entry] = struct{}{}
					break
				}
			}
		}

		if !validLP {
			entry.SetLockPointFromTxEntry(lp)
		}
	}

	allRemoves := make(map[*TxEntry]struct{})
	for it := range txToRemove {
		m.calculateDescendants(it, allRemoves)
	}
	m.removeStaged(allRemoves, false, REORG)
}

// RemoveTxSelf will only remove these transaction self.
func (m *TxMempool) RemoveTxSelf(txs []*tx.Tx) {
	m.Lock()
	defer m.Unlock()

	entries := make([]*TxEntry, 0, len(txs))
	for _, tx := range txs {
		if entry, ok := m.poolData[tx.GetHash()]; ok {
			entries = append(entries, entry)
		}
	}

	// todo base on entries to set the new feerate for mempool.
	for _, tx := range txs {
		if entry, ok := m.poolData[tx.GetHash()]; ok {
			stage := make(map[*TxEntry]struct{})
			stage[entry] = struct{}{}
			m.removeStaged(stage, true, BLOCK)
		}
		m.removeConflicts(tx)
	}
}

func (m *TxMempool) FindTx(hash util.Hash) *TxEntry {
	m.RLock()
	m.RUnlock()
	if find, ok := m.poolData[hash]; ok {
		return find
	}
	return nil
}

func (m *TxMempool) GetCoin(outpoint *outpoint.OutPoint) *utxo.Coin {
	m.RLock()
	defer m.RUnlock()

	txMempoolEntry, ok := m.poolData[outpoint.Hash]
	if !ok {
		return nil
	}
	out := txMempoolEntry.Tx.GetTxOut(int(outpoint.Index))
	if out != nil {
		coin := utxo.NewMempoolCoin(out)
		return coin
	}

	return nil
}

func (m *TxMempool) IsAcceptTx(tx *tx.Tx, txfee int64, mpHeight int, coins []*utxo.Coin,
	tip *blockindex.BlockIndex) (map[*TxEntry]struct{}, LockPoints, error) {

	lp := LockPoints{}
	if _, ok := m.poolData[tx.GetHash()]; ok {
		return nil, lp, errcode.New(errcode.AlreadHaveTx)
	}

	if !checkSequenceLocks(tx, tip, consensus.LocktimeVerifySequence|consensus.LocktimeMedianTimePast, &lp, false, coins) {
		return nil, lp, errcode.New(errcode.Nomature)
	}

	ancestors, err := m.calculateMemPoolAncestors(tx, AncestorNum, AncestorSize*1000,
		DescendantNum, DescendantSize*1000, true)
	if err != nil {
		return nil, lp, err
	}

	txsize := int64(tx.EncodeSize())
	txfeeRate := util.NewFeeRateWithSize(txfee, txsize)
	// compare the transaction feeRate with enter mempool min txfeeRate
	if txfeeRate.SataoshisPerK < m.feeRate.SataoshisPerK {
		return nil, lp, errcode.New(errcode.TooMinFeeRate)
	}

	return ancestors, lp, nil
}

// Check If sanity-checking is turned on, check makes sure the pool is consistent
// (does not contain two transactions that spend the same inputs, all inputs
// are in the mapNextTx array). If sanity-checking is turned off, check does
// nothing.
func (m *TxMempool) Check(view utxo.CacheView, bestHeight int) {
	if m.checkFrequency == 0 {
		return
	}
	if float64(util.GetRand(math.MaxUint32)) >= m.checkFrequency {
		return
	}

	mempoolDuplicate := utxo.NewEmptyCoinsMap()
	logs.SetLogger("mempool", fmt.Sprintf("checking mempool with %d transaction and %d inputs ...", len(m.poolData), len(m.nextTx)))
	checkTotal := uint64(0)

	waitingOnDependants := list.New()
	// foreach every txentry in mempool, and check these txentry correctness.
	for _, entry := range m.poolData {

		checkTotal += uint64(entry.TxSize)
		fDependsWait := false
		setParentCheck := make(map[util.Hash]struct{})

		for _, preout := range entry.Tx.GetAllPreviousOut() {
			if entry, ok := m.poolData[preout.Hash]; ok {
				tx2 := entry.Tx
				if !(tx2.GetOutsCount() > int(preout.Index)) {
					if !tx2.GetTxOut(int(preout.Index)).IsNull() {
						panic("the tx introduced input dose not exist, or the input amount is nil ")
					}
				}

				fDependsWait = true
				setParentCheck[tx2.Hash] = struct{}{}
			} else {
				if !view.HaveCoin(&preout) {
					panic("the tx introduced input dose not exist mempool And UTXO set !!!")
				}
			}

			if _, ok := m.nextTx[preout]; !ok {
				panic("the introduced tx is not in mempool")
			}
		}
		if len(setParentCheck) != len(entry.ParentTx) {
			panic("the two parent set should be equal")
		}

		// Verify ancestor state is correct.
		nNoLimit := uint64(math.MaxUint64)
		setAncestors, err := m.calculateMemPoolAncestors(entry.Tx, nNoLimit, nNoLimit, nNoLimit, nNoLimit, true)
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
		for i := 0; i < entry.Tx.GetOutsCount(); i++ {
			o := outpoint.OutPoint{Hash: entry.Tx.GetHash(), Index: uint32(i)}
			if e, ok := m.nextTx[o]; ok {
				if _, ok := m.poolData[e.Tx.GetHash()]; !ok {
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
			fCheckResult := entry.Tx.IsCoinBase() || entry.Tx.CheckInputsMoney()
			if !fCheckResult {
				panic("the txentry check failed with utxo set...")
			}

			for _, preout := range entry.Tx.GetAllPreviousOut() {
				mempoolDuplicate.SpendCoin(&preout)
			}
			isCoinBase := entry.Tx.IsCoinBase()
			for i := 0; i < entry.Tx.GetOutsCount(); i++ {
				a := outpoint.OutPoint{entry.Tx.GetHash(), uint32(i)}
				mempoolDuplicate.AddCoin(&a, utxo.NewCoin(entry.Tx.GetTxOut(i), 1000000, isCoinBase))
			}
		}
	}

	stepsSinceLastRemove := 0
	for waitingOnDependants.Len() > 0 {
		it := waitingOnDependants.Front()
		entry := it.Value.(*TxEntry)
		waitingOnDependants.Remove(it)
		spend := false
		for _, preOut := range entry.Tx.GetAllPreviousOut() {
			co := mempoolDuplicate.GetCoin(&preOut)
			if !(co != nil && !co.IsSpent()) {
				waitingOnDependants.PushBack(entry)
				stepsSinceLastRemove++
				if !(stepsSinceLastRemove < waitingOnDependants.Len()) {
					panic("the waitingOnDependants list have incorrect number ...")
				}
				spend = true
				break
			}
		}

		if spend {
			fCheckResult := entry.Tx.IsCoinBase()

			for _, preOut := range entry.Tx.GetAllPreviousOut() {
				co := mempoolDuplicate.GetCoin(&preOut)
				if !(co != nil && !co.IsSpent()) {
					fCheckResult = false
					break
				}
			}
			if !fCheckResult {
				panic("this transaction all parent have spent...")
			}
			for _, preout := range entry.Tx.GetAllPreviousOut() {
				mempoolDuplicate.SpendCoin(&preout)
			}
			isCoinBase := entry.Tx.IsCoinBase()
			for i := 0; i < entry.Tx.GetOutsCount(); i++ {
				a := outpoint.OutPoint{entry.Tx.GetHash(), uint32(i)}
				mempoolDuplicate.AddCoin(&a, utxo.NewCoin(entry.Tx.GetTxOut(i), 1000000, isCoinBase))
			}
			stepsSinceLastRemove = 0
		}
	}

	for _, entry := range m.nextTx {
		txid := entry.Tx.GetHash()
		if e, ok := m.poolData[txid]; !ok {
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

// LimitMempoolSize limit mempool size with time And limit size. when the noSpendsRemaining
// set, the function will return these have removed transaction's txin from mempool which use
// TrimToSize rule. Later, caller will remove these txin from uxto cache.
func (m *TxMempool) LimitMempoolSize() []*outpoint.OutPoint {
	m.Lock()
	defer m.Unlock()
	//todo, parse expire time from config
	m.expire(0)
	//todo, parse limit mempoolsize from config
	c := m.trimToSize(0)
	return c
}

// TrimToSize Remove transactions from the mempool until its dynamic size is <=
// sizelimit. noSpendsRemaining, if set, will be populated with the list
// of outpoints which are not in mempool which no longer have any spends in
// this mempool.
func (m *TxMempool) trimToSize(sizeLimit int64) []*outpoint.OutPoint {
	nTxnRemoved := 0
	ret := make([]*outpoint.OutPoint, 0)
	maxFeeRateRemove := int64(0)

	for len(m.poolData) > 0 && m.cacheInnerUsage > sizeLimit {
		removeIt := TxEntry(m.txByAncestorFeeRateSort.Min().(EntryAncestorFeeRateSort))
		rem := m.txByAncestorFeeRateSort.Delete(EntryAncestorFeeRateSort(removeIt)).(EntryAncestorFeeRateSort)
		if rem.Tx.GetHash() != removeIt.Tx.GetHash() {
			panic("the two element should have the same Txhash")
		}
		maxFeeRateRemove = util.NewFeeRateWithSize(removeIt.SumFeeWithDescendants, removeIt.SumSizeWithDescendants).SataoshisPerK
		stage := make(map[*TxEntry]struct{})
		m.calculateDescendants(&removeIt, stage)
		nTxnRemoved += len(stage)
		txn := make([]*tx.Tx, 0, len(stage))
		for iter := range stage {
			txn = append(txn, iter.Tx)
		}

		// here, don't update Descendant transaction state's reason :
		// all Descendant transaction of the removed tx also will be removed.
		m.removeStaged(stage, false, SIZELIMIT)
		for e := range stage {
			fmt.Printf("remove tx hash : %s, mempool size : %d\n", e.Tx.GetHash().String(), m.cacheInnerUsage)
		}
		for _, tx := range txn {
			for _, preout := range tx.GetAllPreviousOut() {
				if _, ok := m.poolData[preout.Hash]; ok {
					continue
				}
				if _, ok := m.nextTx[preout]; !ok {
					ret = append(ret, &preout)
				}
			}
		}

	}

	logs.SetLogger("mempool", fmt.Sprintf("removed %d txn, rolling minimum fee bumped : %d", nTxnRemoved, maxFeeRateRemove))
	return ret
}

// Expire all transaction (and their dependencies) in the memPool older
// than time. Return the number of removed transactions.
func (m *TxMempool) expire(time int64) int {
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
		m.calculateDescendants(removeIt, stage)
	}
	m.removeStaged(stage, false, EXPIRY)
	return len(stage)
}

func (m *TxMempool) removeStaged(entriesToRemove map[*TxEntry]struct{}, updateDescendants bool, reason PoolRemovalReason) {
	m.updateForRemoveFromMempool(entriesToRemove, updateDescendants)
	for rem := range entriesToRemove {
		if _, ok := m.rootTx[rem.Tx.GetHash()]; ok {
			delete(m.rootTx, rem.Tx.GetHash())
		}
		m.delTxentry(rem, reason)
		fmt.Println("remove one transaction late, the mempool size : ", m.cacheInnerUsage)
	}
}

func (m *TxMempool) removeConflicts(tx *tx.Tx) {
	// Remove transactions which depend on inputs of tx, recursively
	for _, preout := range tx.GetAllPreviousOut() {
		if flictEntry, ok := m.nextTx[preout]; ok {
			if flictEntry.Tx.GetHash() != tx.GetHash() {
				m.removeTxRecursive(flictEntry.Tx, CONFLICT)
			}
		}
	}
}

func (m *TxMempool) updateForRemoveFromMempool(entriesToRemove map[*TxEntry]struct{}, updateDescendants bool) {
	nNoLimit := uint64(math.MaxUint64)

	if updateDescendants {
		for removeIt := range entriesToRemove {
			setDescendants := make(map[*TxEntry]struct{})
			m.calculateDescendants(removeIt, setDescendants)
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
		ancestors, err := m.calculateMemPoolAncestors(removeIt.Tx, nNoLimit, nNoLimit, nNoLimit, nNoLimit, false)
		if err != nil {
			return
		}
		m.updateAncestorsOf(false, removeIt, ancestors)
	}
}

// removeTxRecursive remove this transaction And its all descent transaction from mempool.
func (m *TxMempool) removeTxRecursive(origTx *tx.Tx, reason PoolRemovalReason) {
	// Remove transaction from memory pool
	txToRemove := make(map[*TxEntry]struct{})

	if entry, ok := m.poolData[origTx.GetHash()]; ok {
		txToRemove[entry] = struct{}{}
	} else {
		// When recursively removing but origTx isn't in the mempool be sure
		// to remove any children that are in the pool. This can happen
		// during chain re-orgs if origTx isn't re-accepted into the mempool
		// for any reason.
		for i := 0; i < origTx.GetOutsCount(); i++ {
			outPoint := outpoint.OutPoint{Hash: origTx.GetHash(), Index: uint32(i)}
			if en, ok := m.nextTx[outPoint]; !ok {
				continue
			} else {
				if find, ok := m.poolData[en.Tx.GetHash()]; ok {
					txToRemove[find] = struct{}{}
				} else {
					panic("the transaction must in mempool, because NextTx struct of mempool have its data")
				}
			}
		}
	}
	allRemoves := make(map[*TxEntry]struct{})
	for it := range txToRemove {
		m.calculateDescendants(it, allRemoves)
	}
	m.removeStaged(allRemoves, false, reason)
}

// CalculateDescendants Calculates descendants of entry that are not already in setDescendants, and
// adds to setDescendants. Assumes entry it is already a tx in the mempool and
// setMemPoolChildren is correct for tx and all descendants. Also assumes that
// if an entry is in setDescendants already, then all in-mempool descendants of
// it are already in setDescendants as well, so that we can save time by not
// iterating over those entries.
func (m *TxMempool) calculateDescendants(entry *TxEntry, descendants map[*TxEntry]struct{}) {
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
			fmt.Println("tx will romove tx3's from its'parent, tx3 : ", txentry.Tx.GetHash().String(), ", tx1 : ", piter.Tx.GetHash().String())
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
		//fmt.Println("ancestor hash : ", ancestorit.Tx.GetHash().ToString())
		ancestorit.UpdateDescendantState(updateCount, updateSize, updateFee)
	}
}

func (m *TxMempool) updateEntryForAncestors(entry *TxEntry, setAncestors map[*TxEntry]struct{}) {
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
func (m *TxMempool) calculateMemPoolAncestors(tx *tx.Tx, limitAncestorCount uint64,
	limitAncestorSize uint64, limitDescendantCount uint64, limitDescendantSize uint64,
	searchForParent bool) (ancestors map[*TxEntry]struct{}, err error) {

	ancestors = make(map[*TxEntry]struct{})
	parent := make(map[*TxEntry]struct{})
	if searchForParent {
		for _, preout := range tx.GetAllPreviousOut() {
			if entry, ok := m.poolData[preout.Hash]; ok {
				parent[entry] = struct{}{}
				if uint64(len(parent))+1 > limitAncestorCount {
					return nil, errcode.New(errcode.ManyUnspendDepend)
				}
			}
		}
	} else {
		// If we're not searching for parents, we require this to be an entry in
		// the mempool already.
		if entry, ok := m.poolData[tx.GetHash()]; ok {
			parent = entry.ParentTx
		} else {
			panic("the tx must be in mempool")
		}
	}

	totalSizeWithAncestors := int64(tx.EncodeSize())
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
		if uint64(entry.SumSizeWithDescendants+int64(entry.TxSize)) > limitDescendantSize {
			return nil, errcode.New(errcode.ManyUnspendDepend)
		} else if uint64(entry.SumTxCountWithDescendants+1) > limitDescendantCount {
			return nil, errcode.New(errcode.ManyUnspendDepend)
		} else if uint64(totalSizeWithAncestors) > limitAncestorSize {
			return nil, errcode.New(errcode.ManyUnspendDepend)
		}

		graTxentrys := entry.ParentTx
		for gentry := range graTxentrys {
			if _, ok := ancestors[gentry]; !ok {
				paSLice = append(paSLice, gentry)
			}
			if uint64(len(parent)+len(ancestors)+1) > limitAncestorCount {
				return nil, errcode.New(errcode.ManyUnspendDepend)
			}
		}
	}

	return ancestors, nil
}

func (m *TxMempool) delTxentry(removeEntry *TxEntry, reason PoolRemovalReason) {
	// todo add signal for any subscriber

	for _, preout := range removeEntry.Tx.GetAllPreviousOut() {
		delete(m.nextTx, preout)
	}

	if _, ok := m.rootTx[removeEntry.Tx.GetHash()]; ok {
		delete(m.rootTx, removeEntry.Tx.GetHash())
	}
	m.cacheInnerUsage -= int64(removeEntry.usageSize) + int64(unsafe.Sizeof(removeEntry))
	m.TransactionsUpdated++
	m.totalTxSize -= uint64(removeEntry.TxSize)
	delete(m.poolData, removeEntry.Tx.GetHash())
	m.timeSortData.Delete(removeEntry)
	m.txByAncestorFeeRateSort.Delete(EntryAncestorFeeRateSort(*removeEntry))
}

func checkSequenceLocks(tx *tx.Tx, tip *blockindex.BlockIndex, flags int, lp *LockPoints, useExistingLockPoints bool, coins []*utxo.Coin) bool {
	//TODO:AssertLockHeld(cs_main) and AssertLockHeld(mempool.cs) not finish
	var index *blockindex.BlockIndex
	index.Prev = tip
	// CheckSequenceLocks() uses chainActive.Height()+1 to evaluate height based
	// locks because when SequenceLocks() is called within ConnectBlock(), the
	// height of the block *being* evaluated is what is used. Thus if we want to
	// know if a transaction can be part of the *next* block, we need to use one
	// more than chainActive.Height()
	index.Height = tip.Height + 1
	lockPair := make(map[int]int64)

	if useExistingLockPoints {
		if lp == nil {
			panic("the mempool lockPoints is nil")
		}
		lockPair[lp.Height] = lp.Time
	} else {
		var prevheights []int
		for txinIndex, coin := range coins {
			if coin.IsMempoolCoin() {
				prevheights[txinIndex] = tip.Height + 1
			} else {
				prevheights[txinIndex] = int(coin.GetHeight())
			}
		}

		lockPair = ltx.CalculateSequenceLocks(*tx, flags, prevheights, index)
		if lp != nil {
			lockPair[lp.Height] = lp.Time
			// Also store the hash of the block with the highest height of all
			// the blocks which have sequence locked prevouts. This hash needs
			// to still be on the chain for these LockPoint calculations to be
			// valid.
			// Note: It is impossible to correctly calculate a maxInputBlock if
			// any of the sequence locked inputs depend on unconfirmed txs,
			// except in the special case where the relative lock time/height is
			// 0, which is equivalent to no sequence lock. Since we assume input
			// height of tip+1 for mempool txs and test the resulting lockPair
			// from CalculateSequenceLocks against tip+1. We know
			// EvaluateSequenceLocks will fail if there was a non-zero sequence
			// lock on a mempool input, so we can use the return value of
			// CheckSequenceLocks to indicate the LockPoints validity
			maxInputHeight := 0
			for height := range prevheights {
				// Can ignore mempool inputs since we'll fail if they had non-zero locks
				if height != tip.Height+1 {
					maxInputHeight = int(math.Max(float64(maxInputHeight), float64(height)))
				}
			}
			lp.MaxInputBlock = tip.GetAncestor(maxInputHeight)
		}
	}
	return EvaluateSequenceLocks(index, lockPair)
}

func EvaluateSequenceLocks(block *blockindex.BlockIndex, lockPair map[int]int64) bool {
	if block.Prev == nil {
		panic("the block's pprev is nil, Please check.")
	}
	nBlocktime := block.Prev.GetMedianTimePast()
	for key, value := range lockPair {
		if int32(key) >= block.Height || value >= nBlocktime {
			return false
		}
	}
	return true
}

func (m *TxMempool) TxInfoAll() []*TxMempoolInfo {
	m.RLock()
	defer m.RUnlock()

	ret := make([]*TxMempoolInfo, len(m.poolData))
	index := 0
	m.txByAncestorFeeRateSort.Ascend(func(i btree.Item) bool {
		entry := TxEntry(i.(EntryAncestorFeeRateSort))
		ret[index] = entry.GetInfo()
		index++
		return true
	})

	return ret
}

func NewTxMempool() *TxMempool {
	t := &TxMempool{}
	t.feeRate = util.FeeRate{SataoshisPerK: 1}
	t.nextTx = make(map[outpoint.OutPoint]*TxEntry)
	t.poolData = make(map[util.Hash]*TxEntry)
	t.timeSortData = *btree.New(32)
	t.rootTx = make(map[util.Hash]*TxEntry)
	t.txByAncestorFeeRateSort = *btree.New(32)
	return t
}

func InitMempool() {
	Gpool = NewTxMempool()
}

const (
	OrphanTxExpireTime          = 20 * 60
	OrphanTxExpireInterval      = 5 * 60
	DefaultMaxOrphanTransaction = 100
)

type OrphanTx struct {
	Tx         *tx.Tx
	NodeID     int64
	Expiration int
}

func (m *TxMempool) AddOrphanTx(orphantx *tx.Tx, nodeID int64) {
	if _, ok := m.OrphanTransactions[orphantx.GetHash()]; ok {
		return
	}
	sz := orphantx.EncodeSize()
	if sz >= consensus.MaxTxSize {
		return
	}
	o := OrphanTx{Tx: orphantx, NodeID: nodeID, Expiration: time.Now().Second() + OrphanTxExpireTime}
	m.OrphanTransactions[orphantx.GetHash()] = o
	for _, preout := range orphantx.GetAllPreviousOut() {
		if exsit, ok := m.OrphanTransactionsByPrev[preout]; ok {
			exsit[orphantx.GetHash()] = o
		} else {
			mi := make(map[util.Hash]OrphanTx)
			mi[orphantx.GetHash()] = o
			m.OrphanTransactionsByPrev[preout] = mi
		}
	}
}

func (m *TxMempool) EraseOrphanTx(txHash util.Hash, removeRedeemers bool) {

	if orphanTx, ok := m.OrphanTransactions[txHash]; ok {
		for _, preout := range orphanTx.Tx.GetAllPreviousOut() {
			if orphans, exsit := m.OrphanTransactionsByPrev[preout]; exsit {
				delete(orphans, txHash)
				if len(orphans) == 0 {
					delete(m.OrphanTransactionsByPrev, preout)
				}
			}
		}
	}
	if removeRedeemers {
		preout := outpoint.OutPoint{Hash: txHash}
		orphan := m.OrphanTransactions[txHash]
		for i := 0; i < orphan.Tx.GetOutsCount(); i++ {
			preout.Index = uint32(i)
			for _, orphan := range m.OrphanTransactionsByPrev[preout] {
				m.EraseOrphanTx(orphan.Tx.GetHash(), true)
			}
		}
	}
	delete(m.OrphanTransactions, txHash)
}

func (m *TxMempool) LimitOrphanTx() int {

	removeNum := 0
	now := time.Now().Second()
	if m.nextSweep <= now {
		minExpTime := now + OrphanTxExpireTime - OrphanTxExpireInterval
		for hash, orphan := range m.OrphanTransactions {
			if orphan.Expiration <= now {
				m.EraseOrphanTx(hash, true)
			} else {
				if minExpTime > orphan.Expiration {
					minExpTime = orphan.Expiration
				}
			}
		}
		m.nextSweep = minExpTime + OrphanTxExpireInterval
	}

	for {
		if len(m.OrphanTransactions) < DefaultMaxOrphanTransaction {
			break
		}
		for hash := range m.OrphanTransactions {
			m.EraseOrphanTx(hash, true)
		}
	}
	return removeNum
}

func (m *TxMempool)RemoveOrphansByTag(nodeID int64) int {
	numEvicted := 0
	m.Lock()
	for _, otx := range m.OrphanTransactions{
		if otx.NodeID == nodeID{
			m.EraseOrphanTx(otx.Tx.GetHash(), true)
			numEvicted++
		}
	}
	m.Unlock()
	return numEvicted
}