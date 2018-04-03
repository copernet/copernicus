package mempool

import (
	"fmt"
	"math"
	"sync"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
	"github.com/google/btree"
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

// TxMempool is safe for concurrent write And read access.
type TxMempool struct {
	sync.RWMutex
	// current mempool best feerate for one transaction.
	fee utils.FeeRate
	// poolData : store the tx in the mempool
	PoolData        map[utils.Hash]*TxEntry
	NextTx          map[core.OutPoint]*TxEntry
	timeSortData    btree.BTree
	cacheInnerUsage int64
	checkFrequency  float64
}

func (m *TxMempool) GetCheckFreQuency() float64 {
	return m.checkFrequency
}

// RemoveForBlock when a new valid block is received, so all the transaction
// in the block should removed from memPool.
func (m *TxMempool) RemoveForBlock(txs []*core.Tx, txHeight int) {
	m.Lock()
	defer m.Unlock()

	entries := make([]*TxEntry, 0, 2000)
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
func (m *TxMempool) AddTx(tx *core.Tx, txFee int64) bool {
	// todo: send signal to all interesting the caller.
	m.Lock()
	defer m.Unlock()
	nNoLimit := uint64(math.MaxUint64)
	ancestors, _ := m.CalculateMemPoolAncestors(tx, nNoLimit, nNoLimit, nNoLimit, nNoLimit, true)

	// insert new txEntry to the memPool; and update the memPool's memory consume.
	newTxEntry := NewTxentry(tx, txFee)
	m.PoolData[tx.Hash] = newTxEntry
	m.cacheInnerUsage += newTxEntry.usageSize
	m.timeSortData.ReplaceOrInsert(newTxEntry)

	// Update ancestors with information about this tx
	setParentTransactions := make(map[utils.Hash]struct{})
	for _, txin := range tx.Ins {
		m.NextTx[*txin.PreviousOutPoint] = newTxEntry
		setParentTransactions[txin.PreviousOutPoint.Hash] = struct{}{}
	}

	for hash := range setParentTransactions {
		if parent, ok := m.PoolData[hash]; ok {

			newTxEntry.UpdateParent(parent, true)
		}
	}

	m.updateAncestorsOf(true, newTxEntry, ancestors)

	return true
}

func (m *TxMempool) TrimToSize(sizeLimit int64, noSpendsRemaining *[]*core.OutPoint) {
	m.Lock()
	defer m.Unlock()

	nTxnRemoved := 0
	for _, remove := range m.PoolData {
		if m.cacheInnerUsage > sizeLimit {
			stage := make(map[*TxEntry]struct{})
			m.CalculateDescendants(remove, stage)
			nTxnRemoved += len(stage)

			txn := make([]*core.Tx, 0, len(stage))
			if noSpendsRemaining != nil {
				for iter := range stage {
					txn = append(txn, iter.tx)
				}
			}

			m.RemoveStaged(stage, false, SIZELIMIT)
			if noSpendsRemaining != nil {
				for _, tx := range txn {
					for _, txin := range tx.Ins {
						if m.FindTx(txin.PreviousOutPoint.Hash) != nil {
							continue
						}
						if _, ok := m.NextTx[*txin.PreviousOutPoint]; !ok {
							*noSpendsRemaining = append(*noSpendsRemaining, txin.PreviousOutPoint)
						}
					}
				}
			}
		}
	}
	logs.Debug("mempool remove %d transactions with SIZELIMIT reason. \n", nTxnRemoved)

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

// Check If sanity-checking is turned on, check makes sure the pool is consistent
// (does not contain two transactions that spend the same inputs, all inputs
// are in the mapNextTx array). If sanity-checking is turned off, check does
// nothing.
func (m *TxMempool) Check(coins *utxo.CoinsViewCache) {
	if m.checkFrequency == 0 {
		return
	}
	if float64(utils.GetRand(math.MaxUint32)) >= m.checkFrequency {
		return
	}

	logs.Debug("mempool Checking mempool with %d transactions and %d inputs", len(m.PoolData), len(m.NextTx))

	//checkTotal := uint64(0)
	//innerUsage := uint64(0)

}

func (m *TxMempool) RemoveStaged(entriesToRemove map[*TxEntry]struct{}, updateDescendants bool, reason PoolRemovalReason) {

	nNoLimit := uint64(math.MaxUint64)
	for removeIt := range entriesToRemove {
		ancestors, err := m.CalculateMemPoolAncestors(removeIt.tx, nNoLimit, nNoLimit, nNoLimit, nNoLimit, false)
		if err != nil {
			return
		}
		m.updateAncestorsOf(false, removeIt, ancestors)
	}

	for rem := range entriesToRemove {
		m.delTxentry(rem, reason)
	}
}

func (m *TxMempool) removeConflicts(tx *core.Tx) {
	// Remove transactions which depend on inputs of tx, recursively
	for _, txin := range tx.Ins {
		if flictEntry, ok := m.NextTx[*txin.PreviousOutPoint]; ok {
			if flictEntry.tx.Hash != tx.Hash {
				m.removeRecursive(flictEntry.tx, CONFLICT)
			}
		}
	}
}

func (m *TxMempool) removeRecursive(origTx *core.Tx, reason PoolRemovalReason) {
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
				if find, ok := m.PoolData[en.tx.Hash]; ok {
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
	stage := make(map[*TxEntry]struct{})
	if _, ok := descendants[entry]; !ok {
		stage[entry] = struct{}{}
	}

	// Traverse down the children of entry, only adding children that are not
	// accounted for in setDescendants already (because those children have
	// either already been walked, or will be walked in this iteration).
	for desEntry := range stage {
		descendants[desEntry] = struct{}{}
		delete(stage, desEntry)

		for child := range desEntry.childTx {
			if _, ok := descendants[child]; !ok {
				stage[child] = struct{}{}
			}
		}
	}

}

// updateAncestorsOf update each of ancestors transaction state; add or remove this
// txentry txfee, txsize, txcount.
func (m *TxMempool) updateAncestorsOf(add bool, txentry *TxEntry, ancestors map[*TxEntry]struct{}) {

	// update the parent's child transaction set;
	for piter := range txentry.parentTx {
		if add {
			piter.childTx[txentry] = struct{}{}
		}
		delete(piter.childTx, txentry)
	}

	// update each of ancestors transaction state;
	for ancestorit := range ancestors {
		m.updateDescendant(add, ancestorit, txentry)
	}
}

func (m *TxMempool) updateDescendant(add bool, ancestor *TxEntry, entry *TxEntry) {
	if add {
		ancestor.sumFeeWithDescendants += entry.txFee
		ancestor.sumTxCountWithDescendants++
		ancestor.sumSizeWithDescendants += uint64(entry.txSize)
	}
	ancestor.sumFeeWithDescendants -= entry.txFee
	ancestor.sumTxCountWithDescendants--
	ancestor.sumSizeWithDescendants -= uint64(entry.txSize)
}

// CalculateMemPoolAncestors get tx all ancestors transaction in mempool.
// when the find is false: the tx must in mempool, so directly get his parent.
func (m *TxMempool) CalculateMemPoolAncestors(tx *core.Tx, limitAncestorCount uint64,
	limitAncestorSize uint64, limitDescendantCount uint64, limitDescendantSize uint64,
	find bool) (ancestors map[*TxEntry]struct{}, err error) {

	ancestors = make(map[*TxEntry]struct{})
	parent := make(map[*TxEntry]struct{})
	if find {
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
			parent = entry.parentTx
		} else {
			panic("the tx must be in mempool")
		}
	}

	totalSizeWithAncestors := uint64(tx.SerializeSize())

	for entry := range parent {
		delete(parent, entry)
		ancestors[entry] = struct{}{}
		totalSizeWithAncestors += uint64(entry.txSize)
		hash := entry.tx.Hash
		if entry.sumSizeWithDescendants+uint64(entry.txSize) > limitDescendantSize {
			return nil,
				fmt.Errorf("exceeds descendant size limit for tx %s [limit: %d]", hash.ToString(), limitDescendantSize)
		} else if entry.sumTxCountWithDescendants+1 > limitDescendantCount {
			return nil,
				fmt.Errorf("too many descendants for tx %s [limit: %d]", hash.ToString(), limitDescendantCount)
		} else if totalSizeWithAncestors > limitAncestorSize {
			return nil,
				fmt.Errorf("exceeds ancestor size limit [limit: %d]", limitAncestorSize)
		}

		graTxentrys := entry.parentTx
		for gentry := range graTxentrys {
			if _, ok := ancestors[gentry]; !ok {
				parent[gentry] = struct{}{}
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

	for _, txin := range removeEntry.tx.Ins {
		delete(m.NextTx, *txin.PreviousOutPoint)
	}

	m.cacheInnerUsage -= removeEntry.usageSize
	delete(m.PoolData, removeEntry.tx.Hash)
}

func (m *TxMempool) FindTx(hash utils.Hash) *core.Tx {
	m.RLock()
	m.RUnlock()
	if find, ok := m.PoolData[hash]; ok {
		return find.tx
	}
	return nil
}

func (m *TxMempool) xAcceptTx(tx *core.Tx) bool {
	return true
}

func (m *TxMempool) RejectTx(tx *core.Tx) bool {
	return true
}

func (m *TxMempool) GetDescendant(tx *core.Tx) int {
	return 0
}

func (m *TxMempool) GetAncestor(tx *core.Tx) int {
	return 0
}
