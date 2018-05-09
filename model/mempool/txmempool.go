package mempool

import (
	"sync"
	"github.com/google/btree"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/astaxie/beego/logs"
	"fmt"
	"container/list"
	"github.com/btcboost/copernicus/model/tx"
	"unsafe"
	"math"
	"github.com/btcboost/copernicus/model/txin"
	"time"
)

var Gpool *TxMempool

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
	fee 					util.FeeRate
	// poolData store the tx in the mempool
	poolData 				map[util.Hash]*TxEntry
	//NextTx key is txPrevout, value is tx.
	nextTx 					map[outpoint.OutPoint]*TxEntry
	//RootTx contain all root transaction in mempool.
	rootTx                  map[util.Hash]*TxEntry
	txByAncestorFeeRateSort btree.BTree
	timeSortData            btree.BTree
	cacheInnerUsage         int64
	checkFrequency          float64
	// sum of all mempool tx's size.
	totalTxSize 			uint64
	//transactionsUpdated mempool update transaction total number when create mempool late.
	transactionsUpdated 	uint64
	pennyTotal				float64
	lastPennyUnix			int64
	FreeTxRelayLimit		float64
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
	m.poolData[txentry.Tx.Hash] = txentry
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
	m.transactionsUpdated++
	m.txByAncestorFeeRateSort.ReplaceOrInsert(EntryAncestorFeeRateSort(*txentry))
	if txentry.SumTxCountWithAncestors == 1 {
		m.rootTx[txentry.Tx.Hash] = txentry
	}
	return nil
}

func (m *TxMempool) HasSpentOut(out *outpoint.OutPoint) bool {
	m.RLock()
	m.RUnlock()

	if _, ok := m.nextTx[*out]; ok {
		return true
	}
	return false
}

func (m *TxMempool) RemoveTxRecursive(origTx *tx.Tx, reason PoolRemovalReason)  {
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

func (m *TxMempool)GetAllTxEntry() map[util.Hash]*TxEntry {
	return m.poolData
}

func (m *TxMempool)RemoveUnFinalTx(chain *Chain, pcoins *utxo.CoinsViewCache, nMemPoolHeight int, flag int) {
	m.Lock()
	defer m.Unlock()

	// Remove transactions spending a coinbase which are now immature and
	// no-longer-final transactions
	txToRemove := make(map[*TxEntry]struct{})
	tip := chain.Tip()
	for _, entry := range m.poolData {
		lp := entry.GetLockPointFromTxEntry()
		validLP := entry.CheckLockPointValidity(chain)
		state := NewValidationState()

		tx := entry.Tx
		allPreout := tx.GetAllPreviousOut()
		coins := make([]*Coin, len(allPreout))
		for i, preout := range allPreout{
			if coin := pcoins.GetCoin(preout); coin != nil{
				coins[i] = coin
			} else {
				if coin := m.GetCoin(&preout); coin != nil{
					coins[i] = coin
				}else {
					panic("the transaction in mempool, not found its parent " +
						"transaction in local node and utxo")
				}
			}
		}
		if !tx.ContextualCheckTransactionForCurrentBlock(state, msg.ActiveNetParams, uint(flag)) ||
			!checkSequenceLocks(tx, tip, flag, &lp, validLP, coins) {
			txToRemove[entry] = struct{}{}
		} else if entry.GetSpendsCoinbase() {
			for _, preout := range tx.GetAllPreviousOut() {
				if _, ok := m.poolData[preout.Hash]; ok {
					continue
				}

				coin := pcoins.AccessCoin(preout)
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
		if entry, ok := m.poolData[tx.Hash]; ok {
			entries = append(entries, entry)
		}
	}

	// todo base on entries to set the new feerate for mempool.

	for _, tx := range txs {
		if entry, ok := m.poolData[tx.Hash]; ok {
			stage := make(map[*TxEntry]struct{})
			stage[entry] = struct{}{}
			m.removeStaged(stage, true, BLOCK)
		}
		m.removeConflicts(tx)
	}
}


func (m *TxMempool) FindTx(hash util.Hash) *tx.Tx {
	m.RLock()
	m.RUnlock()
	if find, ok := m.poolData[hash]; ok {
		return find.Tx
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
	if out != nil{
		coin := utxo.NewCoin(out, consensus.MempoolHeight, false)
		return coin
	}

	return nil
}

func CalcPriority(tx *tx.Tx, nextBlockHeight int, coins []*Coin) float64 {
	overhead := 0
	for _, txIn := range tx.GetAllPreviousOut() {
		// Max inputs + size can't possibly overflow here.
		overhead += 41 + minInt(110, len(txIn.SignatureScript))
	}

	serializedTxSize := tx.SerializeSize()
	if overhead >= int(serializedTxSize) {
		return 0.0
	}

	var totalInputAge float64
	for _, coin := range coins{
		height := coin.GetHeight()
		inputAge := 0
		if height == MempoolHeight {
			inputAge = 0
		}else {
			inputAge = nextBlockHeight - height
		}

		// Sum the input value times age.
		inputValue := coin.GetValue()
		totalInputAge += inputAge * inputValue
	}
	ret :=  totalInputAge / float64(serializedTxSize - uint(overhead))
	return ret
}


// calcMinRequiredTxRelayFee returns the minimum transaction fee required for a
// transaction with the passed serialized size to be accepted into the memory
// pool and relayed.
func calcMinRequiredTxRelayFee(serializedSize int64, minRelayTxFee int64) int64 {
	// Calculate the minimum fee for a transaction to be allowed into the
	// mempool and relayed by scaling the base fee (which is the minimum
	// free transaction relay fee).  minTxRelayFee is in Satoshi/kB so
	// multiply by serializedSize (which is in bytes) and divide by 1000 to
	// get minimum Satoshis.
	minFee := (serializedSize * minRelayTxFee) / 1000

	if minFee == 0 && minRelayTxFee > 0 {
		minFee = int64(minRelayTxFee)
	}

	// Set the minimum fee to the maximum possible value if the calculated
	// fee is not in the valid range for monetary amounts.
	if minFee < 0 || minFee > MaxSatoshi {
		minFee = MaxSatoshi
	}

	return minFee
}

func (m *TxMempool)IsAcceptTx(tx *tx.Tx, txfee int64, mpHeight int, coins []*Coin,
		tip *BlockIndex) (map[*TxEntry]struct{}, LockPoints, bool) {

	lp := LockPoints{}
	if _, ok := m.poolData[tx.Hash]; ok{
		return nil, lp, false
	}

	if !checkSequenceLocks(tx, tip, STANDARD_LOCKTIME_VERIFY_FLAGS, &lp, false, coins){
		return nil, lp, false
	}

	limitAncestors := 0
	limitAncestorSize := 0
	limitDescendants := 0
	limitDescendantSize := 0
	ancestors, ok := m.calculateMemPoolAncestors(tx, uint64(limitAncestors), uint64(limitAncestorSize),
		uint64(limitDescendants), uint64(limitDescendantSize), true)
	if ok != nil{
		return nil, lp, false
	}

	txsize := int64(tx.SerializeSize())
	// compare the transaction feeRate with enter mempool min txfee, And
	// relay fee with the txfee.
	minFee := calcMinRequiredTxRelayFee(txsize, m.fee.SataoshisPerK)
	if txfee < minFee {
		currentPriority := CalcPriority(tx, mpHeight, coins)
		if currentPriority <= MinHighPriority{
			return nil, lp, false
		}
	}

	// Free-to-relay transactions are rate limited here to prevent
	// penny-flooding with tiny transactions as a form of attack.
	if txfee < minFee{
		nowUnix := time.Now().Unix()
		// Decay passed data with an exponentially decaying ~10 minute
		// window - matches bitcoind handling.
		m.pennyTotal *= math.Pow(1.0-1.0/600.0,
			float64(nowUnix-m.lastPennyUnix))
		m.lastPennyUnix = nowUnix
		m.pennyTotal += float64(txsize)

		// Are we still over the limit?
		if m.pennyTotal >= m.FreeTxRelayLimit*10*1000 {
			return nil, lp, false
		}
		oldTotal := m.pennyTotal
		_ = oldTotal

		//todo add log
	}

	return ancestors, lp, true
}

// Check If sanity-checking is turned on, check makes sure the pool is consistent
// (does not contain two transactions that spend the same inputs, all inputs
// are in the mapNextTx array). If sanity-checking is turned off, check does
// nothing.
func (m *TxMempool) Check(coins *utxo.CoinsViewCache, bestHeight int) {
	if m.checkFrequency == 0 {
		return
	}
	if float64(util.GetRand(math.MaxUint32)) >= m.checkFrequency {
		return
	}

	logs.SetLogger("mempool", fmt.Sprintf("checking mempool with %d transaction and %d inputs ...", len(m.poolData), len(m.nextTx)))
	checkTotal := uint64(0)
	m.Lock()
	defer m.Unlock()

	waitingOnDependants := list.New()
	// foreach every txentry in mempool, and check these txentry correctness.
	for _, entry := range m.poolData {

		checkTotal += uint64(entry.TxSize)
		fDependsWait := false
		setParentCheck := make(map[util.Hash]struct{})

		for _, preout := range entry.Tx.GetAllPreviousOut() {
			if entry, ok := m.poolData[preout.Hash]; ok {
				tx2 := entry.Tx
				if !len(tx2.Outs) > int(preout.Index){
					if !tx2.GetTxOut(int(preout.Index)).IsNull(){
						panic("the tx introduced input dose not exist, or the input amount is nil ")
					}
				}

				fDependsWait = true
				setParentCheck[tx2.Hash] = struct{}{}
			} else {
				if !coins.HaveCoin(txin.PreviousOutPoint) {
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
		for i := range entry.Tx.Outs {
			o := outpoint.OutPoint{Hash: entry.Tx.Hash, Index: uint32(i)}
			if e, ok := m.nextTx[o]; ok {
				if _, ok := m.poolData[e.Tx.Hash]; !ok {
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
			var state ValidationState
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

	for _, entry := range m.nextTx {
		txid := entry.Tx.Hash
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
func (m *TxMempool)LimitMempoolSize() []*outpoint.OutPoint {
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
		if rem.Tx.Hash != removeIt.Tx.Hash {
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
			fmt.Printf("remove tx hash : %s, mempool size : %d\n", e.Tx.Hash.ToString(), m.cacheInnerUsage)
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
		if _, ok := m.rootTx[rem.Tx.Hash]; ok {
			delete(m.rootTx, rem.Tx.Hash)
		}
		m.delTxentry(rem, reason)
		fmt.Println("remove one transaction late, the mempool size : ", m.cacheInnerUsage)
	}
}

func (m *TxMempool) removeConflicts(tx *tx.Tx) {
	// Remove transactions which depend on inputs of tx, recursively
	for _, preout := range tx.GetAllPreviousOut() {
		if flictEntry, ok := m.nextTx[preout]; ok {
			if flictEntry.Tx.Hash != tx.Hash {
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

	if entry, ok := m.poolData[origTx.Hash]; ok {
		txToRemove[entry] = struct{}{}
	} else {
		// When recursively removing but origTx isn't in the mempool be sure
		// to remove any children that are in the pool. This can happen
		// during chain re-orgs if origTx isn't re-accepted into the mempool
		// for any reason.
		for i := range origTx.Outs {
			outPoint := outpoint.OutPoint{Hash: origTx.Hash, Index: uint32(i)}
			if en, ok := m.nextTx[outPoint]; !ok {
				continue
			} else {
				if find, ok := m.poolData[en.Tx.Hash]; ok {
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
					return nil,
						fmt.Errorf("too many unconfirmed parents [limit: %d]", limitAncestorCount)
				}
			}
		}
	} else {
		// If we're not searching for parents, we require this to be an entry in
		// the mempool already.
		if entry, ok := m.poolData[tx.Hash]; ok {
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

	for _, preout := range removeEntry.Tx.GetAllPreviousOut() {
		delete(m.nextTx, preout)
	}

	if _, ok := m.rootTx[removeEntry.Tx.Hash]; ok {
		delete(m.rootTx, removeEntry.Tx.Hash)
	}
	m.cacheInnerUsage -= int64(removeEntry.usageSize) + int64(unsafe.Sizeof(removeEntry))
	m.transactionsUpdated++
	m.totalTxSize -= uint64(removeEntry.TxSize)
	delete(m.poolData, removeEntry.Tx.Hash)
	m.timeSortData.Delete(removeEntry)
	m.txByAncestorFeeRateSort.Delete(EntryAncestorFeeRateSort(*removeEntry))
}


func checkSequenceLocks(tx *tx.Tx, tip *BlockIndex, flags int, lp *LockPoints, useExistingLockPoints bool, coins []*Coin) bool {
	//TODO:AssertLockHeld(cs_main) and AssertLockHeld(mempool.cs) not finish
	var index *BlockIndex
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
		for _, coin := range coins{
			if coin.GetHeight() == consensus.MEMPOOL_HEIGHT {
				// Assume all mempool transaction confirm in the next block
				prevheights[txinIndex] = tip.Height + 1
			} else {
				prevheights[txinIndex] = int(coin.GetHeight())
			}
		}

		lockPair = tx.CalculateSequenceLocks(flags, prevheights, index)
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

func EvaluateSequenceLocks(block *BlockIndex, lockPair map[int]int64) bool {
	if block.Prev == nil {
		panic("the block's pprev is nil, Please check.")
	}
	nBlocktime := block.Prev.GetMedianTimePast()
	for key, value := range lockPair {
		if key >= block.Height || value >= nBlocktime {
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
	t.fee = util.FeeRate{SataoshisPerK: 1}
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

type LockPoints struct {
	// Height and Time will be set to the blockChain height and median time past values that
	// would be necessary to satisfy all relative lockTime constraints (BIP68)
	// of this tx given our view of block chain history
	Height int
	Time   int64
	// MaxInputBlock as long as the current chain descends from the highest height block
	// containing one of the inputs used in the calculation, then the cached
	// values are still valid even after a reOrg.
	MaxInputBlock *blockindex
}

func NewLockPoints() *LockPoints {
	lockPoints := LockPoints{}
	return &lockPoints
}





