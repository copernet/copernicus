package mempool

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/algorithm/mapcontainer"
	"github.com/copernet/copernicus/util/algorithm/mapcontainer/skiplist"
)

const (
	RollingFeeHalfLife = 12 * 60 * 60
)

var gpool *TxMempool

func GetInstance() *TxMempool {
	if gpool == nil {
		gpool = NewTxMempool()
	}

	return gpool
}

func SetInstance(p *TxMempool) {
	gpool = p
}

// Close FIXME this is only for test. We must do it in a graceful way
func Close() {
	gpool = nil
}

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
	lck sync.RWMutex
	// current mempool best feerate for one transaction.
	feeRate util.FeeRate
	// poolData store the tx in the mempool
	poolData map[util.Hash]*TxEntry
	//NextTx key is txPrevout, value is tx.
	nextTx map[outpoint.OutPoint]*TxEntry
	//RootTx contain all root transaction in mempool.
	rootTx map[util.Hash]*TxEntry
	// txByAncestorFeeRateSort btree.BTree
	// timeSortData            btree.BTree
	txByAncestorFeeRateSort mapcontainer.MapContainer
	timeSortData            mapcontainer.MapContainer

	//
	usageSize int64
	// sum of all mempool tx's size.
	totalTxSize uint64
	//transactionsUpdated mempool update transaction total number when create mempool late.
	TransactionsUpdated      uint64
	OrphanTransactionsByPrev map[outpoint.OutPoint]map[util.Hash]OrphanTx
	OrphanTransactions       map[util.Hash]OrphanTx

	RejectedTxs map[util.Hash]struct{}

	nextSweep int

	//MaxMemPoolSize               int64
	incrementalRelayFee          util.FeeRate //
	rollingMinimumFeeRate        int64
	blockSinceLastRollingFeeBump bool
	lastRollingFeeUpdate         int64
}

func (m *TxMempool) Lock() {
	m.lck.Lock()
}
func (m *TxMempool) Unlock() {
	m.lck.Unlock()
}
func (m *TxMempool) RLock() {
	m.lck.RLock()
}
func (m *TxMempool) RUnlock() {
	m.lck.RUnlock()
}

func (m *TxMempool) GetCheckFrequency() uint64 {
	return conf.Cfg.Mempool.CheckFrequency
}

func (m *TxMempool) GetMinFeeRate() util.FeeRate {
	m.RLock()
	feeRate := m.GetMinFee(conf.Cfg.Mempool.MaxPoolSize)
	m.RUnlock()
	return feeRate
}

// AddTx operator is safe for concurrent write And read access.
// this function is used to add tx to the memPool, and now the tx should
// be passed all appropriate checks.
func (m *TxMempool) AddTx(txEntry *TxEntry, ancestors map[*TxEntry]struct{}) error {
	// insert new txEntry to the memPool; and update the memPool's memory consume.
	m.timeSortData.ReplaceOrInsert(txEntry)
	m.poolData[txEntry.Tx.GetHash()] = txEntry
	m.usageSize += int64(txEntry.usageSize)

	// Update ancestors with information about this tx
	m.updateNextTx(txEntry)

	txEntry.AssociateRelationship(m)
	m.StatisticIncrease(txEntry, ancestors, nil)

	m.totalTxSize += uint64(txEntry.TxSize)
	m.TransactionsUpdated++
	m.txByAncestorFeeRateSort.ReplaceOrInsert((*EntryAncestorFeeRateSort)(txEntry))
	if txEntry.SumTxCountWithAncestors == 1 {
		m.rootTx[txEntry.Tx.GetHash()] = txEntry
	}
	m.LimitMempoolSize(conf.Cfg.Mempool.MaxPoolSize, int64(conf.Cfg.Mempool.MaxPoolExpiry)*60*60)
	return nil
}

func (m *TxMempool) HasSpentOut(out *outpoint.OutPoint) bool {
	m.RLock()
	defer m.RUnlock()

	_, ok := m.nextTx[*out]
	return ok
}

func (m *TxMempool) CleanOrphan() {
	m.OrphanTransactionsByPrev = make(map[outpoint.OutPoint]map[util.Hash]OrphanTx)
	m.OrphanTransactions = make(map[util.Hash]OrphanTx)
	log.Debug("mempool.CleanOrphan clean txn: %v", m.OrphanTransactions)
}

func (m *TxMempool) HasSPentOutWithoutLock(out *outpoint.OutPoint) *TxEntry {
	if e, ok := m.nextTx[*out]; ok {
		return e
	}
	return nil
}

func (m *TxMempool) GetPoolAllTxSize(lock bool) uint64 {
	if lock {
		m.RLock()
		defer m.RUnlock()
	}
	size := m.totalTxSize
	return size
}

func (m *TxMempool) GetPoolUsage() int64 {
	m.RLock()
	size := m.usageSize
	m.RUnlock()
	return size
}

func (m *TxMempool) CalculateDescendantsWithLock(txHash *util.Hash) map[*TxEntry]struct{} {
	m.RLock()
	defer m.RUnlock()
	entry, ok := m.poolData[*txHash]
	if !ok {
		return nil
	}
	return m.CalculateDescendants(entry)
}

func (m *TxMempool) CalculateMemPoolAncestorsWithLock(txhash *util.Hash) map[*TxEntry]struct{} {
	m.RLock()
	defer m.RUnlock()
	entry, ok := m.poolData[*txhash]
	if !ok {
		return nil
	}
	noLimit := uint64(math.MaxUint64)
	ancestors, _ := m.CalculateMemPoolAncestors(entry.Tx, noLimit, noLimit, noLimit, noLimit, true)
	return ancestors
}

func (m *TxMempool) RemoveTxRecursive(origTx *tx.Tx, reason PoolRemovalReason) {
	m.Lock()
	m.removeTxRecursive(origTx, reason)
	m.Unlock()
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
	for k, v := range m.poolData {
		ret[k] = v
	}
	m.RUnlock()
	return ret
}

func (m *TxMempool) GetAllTxEntryWithoutLock() map[util.Hash]*TxEntry {
	ret := make(map[util.Hash]*TxEntry, len(m.poolData))
	for k, v := range m.poolData {
		ret[k] = v
	}
	return ret
}

func (m *TxMempool) GetAllSpentOutWithoutLock() map[outpoint.OutPoint]*TxEntry {
	ret := make(map[outpoint.OutPoint]*TxEntry, len(m.nextTx))
	for k, v := range m.nextTx {
		ret[k] = v
	}
	return ret
}

// RemoveTxSelf will only remove these transaction self.
func (m *TxMempool) RemoveTxSelf(txs []*tx.Tx) {
	m.Lock()
	defer m.Unlock()

	// todo base on entries to set the new feerate for mempool.
	for _, tx := range txs {
		if entry, ok := m.poolData[tx.GetHash()]; ok {
			stage := make(map[*TxEntry]struct{})
			stage[entry] = struct{}{}
			m.RemoveStaged(stage, true, BLOCK)
		}
		m.removeConflicts(tx)
	}
	m.lastRollingFeeUpdate = util.GetTimeSec()
	m.blockSinceLastRollingFeeBump = true
}

func (m *TxMempool) FindTx(hash util.Hash) *TxEntry {
	m.RLock()
	defer m.RUnlock()
	if find, ok := m.poolData[hash]; ok {
		return find
	}
	return nil
}

func (m *TxMempool) GetCoin(outpoint *outpoint.OutPoint) *utxo.Coin {
	// m.RLock()
	// defer m.RUnlock()

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

// LimitMempoolSize limit mempool size with time And limit size. when the noSpendsRemaining
// set, the function will return these have removed transaction's txin from mempool which use
// TrimToSize rule. Later, caller will remove these txin from uxto cache.
func (m *TxMempool) LimitMempoolSize(sizeLimit, age int64) {
	expired := m.expire(util.GetTimeSec() - age)
	if expired != 0 {
		log.Debug("Expired %d transactions from the memory pool", expired)
	}
	outPoints := m.trimToSize(sizeLimit)
	view := utxo.GetUtxoCacheInstance()
	for _, outPoint := range outPoints {
		view.RemoveCoins(outPoint)
	}
}

func (m *TxMempool) trackPackageRemoved(rate util.FeeRate) {
	if rate.GetFeePerK() > m.rollingMinimumFeeRate {
		m.rollingMinimumFeeRate = rate.GetFeePerK()
		m.blockSinceLastRollingFeeBump = false
	}
}

func (m *TxMempool) GetMinFee(sizeLimit int64) util.FeeRate {
	if !m.blockSinceLastRollingFeeBump || m.rollingMinimumFeeRate == 0 {
		return *util.NewFeeRate(m.rollingMinimumFeeRate)
	}

	timeTmp := util.GetTimeSec()
	if timeTmp > m.lastRollingFeeUpdate+10 {
		halfLife := RollingFeeHalfLife
		if m.usageSize < sizeLimit/4 {
			halfLife /= 4
		} else if m.usageSize < sizeLimit/2 {
			halfLife /= 2
		}
		m.rollingMinimumFeeRate = m.rollingMinimumFeeRate / int64(math.Pow(2.0, float64(timeTmp-m.lastRollingFeeUpdate))/float64(halfLife))
		m.lastRollingFeeUpdate = timeTmp
		if m.rollingMinimumFeeRate < m.incrementalRelayFee.GetFeePerK()/2 {
			m.rollingMinimumFeeRate = 0
			return *util.NewFeeRate(0)
		}
	}
	rate := util.NewFeeRate(m.rollingMinimumFeeRate)
	if rate.SataoshisPerK > m.incrementalRelayFee.SataoshisPerK {
		return *rate
	}
	return m.incrementalRelayFee
}

// TrimToSize Remove transactions from the mempool until its dynamic size is <=
// sizelimit. noSpendsRemaining, if set, will be populated with the list
// of outpoints which are not in mempool which no longer have any spends in
// this mempool.
func (m *TxMempool) trimToSize(sizeLimit int64) []*outpoint.OutPoint {
	nTxnRemoved := 0
	ret := make([]*outpoint.OutPoint, 0)
	maxFeeRateRemove := int64(0)

	for len(m.poolData) > 0 && m.usageSize > sizeLimit {
		less, _ := m.txByAncestorFeeRateSort.Min()
		removeIt := less.(*EntryAncestorFeeRateSort)

		rmless, _ := m.txByAncestorFeeRateSort.Delete(removeIt)
		rem := rmless.(*EntryAncestorFeeRateSort)
		if rem.Tx.GetHash() != removeIt.Tx.GetHash() {
			panic("the two element should have the same Txhash")
		}
		removed := util.NewFeeRateWithSize(rem.TxFee, int64(rem.TxSize))
		removed.SataoshisPerK += m.incrementalRelayFee.SataoshisPerK

		maxFeeRateRemove = util.NewFeeRateWithSize(removeIt.SumTxFeeWithDescendants, removeIt.SumTxSizeWithDescendants).SataoshisPerK
		stage := m.CalculateDescendants((*TxEntry)(removeIt))
		stage[(*TxEntry)(removeIt)] = struct{}{}
		nTxnRemoved += len(stage)
		txn := make([]*tx.Tx, 0, len(stage))
		for iter := range stage {
			txn = append(txn, iter.Tx)
		}

		// here, don't update Descendant transaction state's reason :
		// all Descendant transaction of the removed tx also will be removed.
		m.RemoveStaged(stage, false, SIZELIMIT)
		for e := range stage {
			log.Debug("remove tx hash : %s, mempool size : %d\n", e.Tx.GetHash(), m.usageSize)
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

	log.Debug("mempool", fmt.Sprintf("removed %d txn, rolling minimum fee bumped : %d", nTxnRemoved, maxFeeRateRemove))
	return ret
}

// Expire all transaction (and their dependencies) in the memPool older
// than time. Return the number of removed transactions.
func (m *TxMempool) expire(time int64) int {
	toremove := make(map[*TxEntry]struct{}, 100)
	m.timeSortData.Ascend(func(i mapcontainer.Lesser) bool {
		entry := i.(*TxEntry)
		if entry.time < time {
			toremove[entry] = struct{}{}
			return true
		}
		return false
	})

	var expireCnt int
	for entry := range toremove {
		stage := m.CalculateDescendants(entry)
		expireCnt += len(stage)
		m.RemoveStaged(stage, false, EXPIRY)
	}

	return expireCnt
}

func (m *TxMempool) RemoveStaged(entriesToRemove map[*TxEntry]struct{}, updateDescendants bool, reason PoolRemovalReason) {
	m.updateForRemoveFromMempool(entriesToRemove, updateDescendants)
	for rem := range entriesToRemove {
		if _, ok := m.rootTx[rem.Tx.GetHash()]; ok {
			delete(m.rootTx, rem.Tx.GetHash())
		}
		m.delTxentry(rem, reason)
		log.Debug("remove one transaction late, the mempool size : ", m.usageSize)
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

func (m *TxMempool) addToFeeRateContainer(entries map[*TxEntry]struct{}) {
	for entry := range entries {
		m.txByAncestorFeeRateSort.ReplaceOrInsert((*EntryAncestorFeeRateSort)(entry))
	}
}

func (m *TxMempool) delFromFeeRateContainer(entries map[*TxEntry]struct{}) {
	for entry := range entries {
		m.txByAncestorFeeRateSort.Delete((*EntryAncestorFeeRateSort)(entry))
	}
}

func (m *TxMempool) updateForRemoveFromMempool(entriesToRemove map[*TxEntry]struct{}, updateDescendants bool) {
	noLimit := uint64(math.MaxUint64)

	for entry := range entriesToRemove {
		ancestors, _ := m.CalculateMemPoolAncestors(entry.Tx, noLimit, noLimit, noLimit, noLimit, false)
		var descendants map[*TxEntry]struct{}
		if updateDescendants {
			descendants = m.CalculateDescendants(entry)
		}
		entry.UnassociateRelationship()

		m.StatisticDescrease(entry, ancestors, descendants)
	}
}

// StatisticIncrease is used to update txEntry, ancestors and descendants's property SumTx<property>With<Ancestor/Descendant>. And also update entry's position in @txByAncestorFeeRateSort.
// NOTE: It only update ancestors and descendants's position in @txByAncestorFeeRateSort.
// It will not deal with txEntry's.
func (m *TxMempool) StatisticIncrease(txEntry *TxEntry,
	ancestors, descendants map[*TxEntry]struct{}) {

	// Update txEntry's 'descendant' and 'ancestor' related properties.
	txEntry.UpdateAncestorState(statisticSum(ancestors))
	txEntry.UpdateDescendantState(statisticSum(descendants))

	// Update @ancestors's 'descendant' related properties.
	statisticIncrementalUpdateDescendant(ancestors, 1,
		txEntry.TxSize, txEntry.SigOpCount, txEntry.TxFee)

	// Update @descendants's 'ancestor' related properties.
	m.delFromFeeRateContainer(descendants)
	statisticIncrementalUpdateAncestors(descendants, 1,
		txEntry.TxSize, txEntry.SigOpCount, txEntry.TxFee)
	m.addToFeeRateContainer(descendants)
}

// StatisticDescrease work as opposite to StatisticIncrease
func (m *TxMempool) StatisticDescrease(txEntry *TxEntry, ancestors, descendants map[*TxEntry]struct{}) {
	// Since txEntry will be removed, there is no need to update @txEntrys' properties.

	// Update @ancestors's 'descendant' related properties.
	statisticIncrementalUpdateDescendant(ancestors, -1, -txEntry.TxSize, -txEntry.SigOpCount, -txEntry.TxFee)

	// Update @descendants's 'ancestor' related properties.
	m.delFromFeeRateContainer(descendants)
	statisticIncrementalUpdateAncestors(descendants, -1, -txEntry.TxSize, -txEntry.SigOpCount, -txEntry.TxFee)
	m.addToFeeRateContainer(descendants)
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
			if en, found := m.nextTx[outPoint]; found {
				if find, ok := m.poolData[en.Tx.GetHash()]; ok {
					txToRemove[find] = struct{}{}
				} else {
					panic("the transaction must in mempool, because NextTx struct of mempool have its data")
				}
			}
		}
	}

	for e := range txToRemove {
		stage := m.CalculateDescendants(e)
		stage[e] = struct{}{}
		m.RemoveStaged(stage, false, reason)
	}
}

// CalculateDescendants Calculates descendants of entry that are not already in setDescendants, and
// adds to setDescendants. Assumes entry it is already a tx in the mempool and
// setMemPoolChildren is correct for tx and all descendants. Also assumes that
// if an entry is in setDescendants already, then all in-mempool descendants of
// it are already in setDescendants as well, so that we can save time by not
// iterating over those entries.
func (m *TxMempool) CalculateDescendants(entry *TxEntry) map[*TxEntry]struct{} {
	descendants := make(map[*TxEntry]struct{})

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

	delete(descendants, entry)
	return descendants
}

func (m *TxMempool) updateNextTx(txEntry *TxEntry) {
	for _, in := range txEntry.Tx.GetIns() {
		m.nextTx[*in.PreviousOutPoint] = txEntry
	}
}

func (m *TxMempool) getParents(txEntry *TxEntry) map[util.Hash]*TxEntry {
	parents := make(map[util.Hash]*TxEntry)
	for _, in := range txEntry.Tx.GetIns() {
		if txn, ok := m.poolData[in.PreviousOutPoint.Hash]; ok {
			parents[in.PreviousOutPoint.Hash] = txn
		}
	}
	return parents
}

func (m *TxMempool) getChildren(txEntry *TxEntry) map[util.Hash]*TxEntry {
	children := make(map[util.Hash]*TxEntry)
	op := outpoint.OutPoint{Hash: txEntry.Tx.GetHash()}
	for op.Index = 0; op.Index < uint32(txEntry.Tx.GetOutsCount()); op.Index++ {
		if child, ok := m.nextTx[op]; ok {
			children[child.Tx.GetHash()] = child
		}
	}
	return children
}

// CalculateMemPoolAncestors get tx all ancestors transaction in mempool.
// when the find is false: the tx must in mempool, so directly get his parent.
func (m *TxMempool) CalculateMemPoolAncestors(tx *tx.Tx, limitAncestorCount uint64,
	limitAncestorSize uint64, limitDescendantCount uint64, limitDescendantSize uint64,
	searchForParent bool) (ancestors map[*TxEntry]struct{}, err error) {

	parents := make(map[*TxEntry]struct{})
	txIns := tx.GetIns()
	if searchForParent {
		for _, txIn := range txIns {
			if entry, ok := m.poolData[txIn.PreviousOutPoint.Hash]; ok {
				parents[entry] = struct{}{}
				if uint64(len(parents))+1 > limitAncestorCount {
					return nil, errcode.New(errcode.ManyUnspendDepend)
				}
			}
		}
	} else {
		// If we're not searching for parents, we require this to be an entry in
		// the mempool already.
		if entry, ok := m.poolData[tx.GetHash()]; ok {
			parents = entry.ParentTx
		} else {
			panic("the tx must be in mempool")
		}
	}

	tempParents := make([]*TxEntry, len(parents))
	j := 0
	for entry := range parents {
		tempParents[j] = entry
		j++
	}

	totalSizeWithAncestors := int64(tx.EncodeSize())
	ancestors = make(map[*TxEntry]struct{})
	for len(tempParents) > 0 {
		entry := tempParents[0]
		tempParents = tempParents[1:]

		ancestors[entry] = struct{}{}
		if uint64(entry.SumTxSizeWithDescendants+int64(entry.TxSize)) > limitDescendantSize {
			return nil, errcode.New(errcode.ManyUnspendDepend)
		}

		if uint64(entry.SumTxCountWithDescendants+1) > limitDescendantCount {
			return nil, errcode.New(errcode.ManyUnspendDepend)
		}

		totalSizeWithAncestors += int64(entry.TxSize)
		if uint64(totalSizeWithAncestors) > limitAncestorSize {
			return nil, errcode.New(errcode.ManyUnspendDepend)
		}

		grandTxentrys := entry.ParentTx
		for grandEntry := range grandTxentrys {
			if _, ok := ancestors[grandEntry]; !ok {
				tempParents = append(tempParents, grandEntry)
			}
			if uint64(len(tempParents)+len(ancestors)+1) > limitAncestorCount {
				return nil, errcode.New(errcode.ManyUnspendDepend)
			}
		}
	}

	return ancestors, nil
}

func (m *TxMempool) delTxentry(removeEntry *TxEntry, reason PoolRemovalReason) {
	// todo add signal for any subscriber

	for _, preout := range removeEntry.Tx.GetAllPreviousOut() {
		if txe, ok := m.nextTx[preout]; ok && txe.Tx.GetHash() == removeEntry.Tx.GetHash() {
			delete(m.nextTx, preout)
		}

	}

	if _, ok := m.rootTx[removeEntry.Tx.GetHash()]; ok {
		delete(m.rootTx, removeEntry.Tx.GetHash())
	}
	m.usageSize -= int64(removeEntry.usageSize)
	m.TransactionsUpdated++
	m.totalTxSize -= uint64(removeEntry.TxSize)
	delete(m.poolData, removeEntry.Tx.GetHash())
	m.timeSortData.Delete(removeEntry)
	m.txByAncestorFeeRateSort.Delete((*EntryAncestorFeeRateSort)(removeEntry))
}

func (m *TxMempool) TxInfoAll() []*TxMempoolInfo {
	m.RLock()
	defer m.RUnlock()

	ret := make([]*TxMempoolInfo, len(m.poolData))
	index := 0
	m.txByAncestorFeeRateSort.Ascend(func(i mapcontainer.Lesser) bool {
		entry := TxEntry(*i.(*EntryAncestorFeeRateSort))
		ret[index] = entry.GetInfo()
		index++
		return true
	})

	return ret
}

func NewTxMempool() *TxMempool {
	return &TxMempool{
		feeRate:                 util.FeeRate{SataoshisPerK: 1},
		poolData:                make(map[util.Hash]*TxEntry),
		nextTx:                  make(map[outpoint.OutPoint]*TxEntry),
		rootTx:                  make(map[util.Hash]*TxEntry),
		txByAncestorFeeRateSort: skiplist.New(30000),
		timeSortData:            skiplist.New(30000),
		incrementalRelayFee:     *util.NewFeeRate(1),

		OrphanTransactionsByPrev: make(map[outpoint.OutPoint]map[util.Hash]OrphanTx),
		OrphanTransactions:       make(map[util.Hash]OrphanTx),
		RejectedTxs:              make(map[util.Hash]struct{}),
	}
}

func InitMempool() {
	gpool = NewTxMempool()
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
		if exist, ok := m.OrphanTransactionsByPrev[preout]; ok {
			exist[o.Tx.GetHash()] = o
		} else {
			mi := make(map[util.Hash]OrphanTx)
			mi[o.Tx.GetHash()] = o
			m.OrphanTransactionsByPrev[preout] = mi
		}
	}

	evicted := m.limitOrphanTx()
	if evicted > 0 {
		log.Debug("Orphan transaction overflow, removed %d orphan tx", evicted)
	}
}

func (m *TxMempool) IsTransactionInPool(tx *tx.Tx) bool {
	_, exists := m.poolData[tx.GetHash()]
	return exists
}

func (m *TxMempool) IsOrphanInPool(tx *tx.Tx) bool {
	_, exists := m.OrphanTransactions[tx.GetHash()]
	return exists
}

func (m *TxMempool) HaveTransaction(tx *tx.Tx) bool {
	return m.IsTransactionInPool(tx) || m.IsOrphanInPool(tx)
}

func (m *TxMempool) EraseOrphanTx(txHash util.Hash, removeRedeemers bool) {

	if orphanTx, ok := m.OrphanTransactions[txHash]; ok {
		for _, preout := range orphanTx.Tx.GetAllPreviousOut() {
			if orphans, exist := m.OrphanTransactionsByPrev[preout]; exist {
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

func (m *TxMempool) limitOrphanTx() (removeNum int) {
	now := time.Now().Second()
	if m.nextSweep <= now {
		minExpTime := now + OrphanTxExpireTime - OrphanTxExpireInterval
		for hash, orphan := range m.OrphanTransactions {
			if orphan.Expiration <= now {
				m.EraseOrphanTx(hash, true)
				removeNum++
			} else {
				if minExpTime > orphan.Expiration {
					minExpTime = orphan.Expiration
				}
			}
		}
		m.nextSweep = minExpTime + OrphanTxExpireInterval
	}

	if len(m.OrphanTransactions) <= DefaultMaxOrphanTransaction {
		return
	}

	for hash := range m.OrphanTransactions {
		m.EraseOrphanTx(hash, true)
		removeNum++
		break
	}
	return
}

func (m *TxMempool) RemoveOrphansByTag(nodeID int64) int {
	numEvicted := 0
	m.Lock()
	for _, otx := range m.OrphanTransactions {
		if otx.NodeID == nodeID {
			m.EraseOrphanTx(otx.Tx.GetHash(), true)
			numEvicted++
		}
	}
	m.Unlock()
	return numEvicted
}
