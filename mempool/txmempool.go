package mempool

import (
	"sync"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

type PoolRemovalReason int

//Reason why a transaction was removed from the mempool, this is passed to the
//* notification signal.
const (
	//UNKNOWN Manually removed or unknown reason
	UNKNOWN PoolRemovalReason = iota
	//EXPIRY Expired from mempool
	EXPIRY
	//SIZELIMIT Removed in size limiting
	SIZELIMIT
	//REORG Removed for reorganization
	REORG
	//BLOCK Removed for block
	BLOCK
	//CONFLICT Removed for conflict with in-block transaction
	CONFLICT
	//REPLACED Removed for replacement
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
	cacheInnerUsage int64
	checkFrequency  float64
}

func (m *TxMempool) GetCheckFreQuency() float64 {
	return m.checkFrequency
}

// RemoveForBlock when a new valid block is received, so all the transaction
// in the block should removed from mempool.
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
// this function is used to add tx to the mempool, and now the tx should
// be passed all appropriate checks.
func (m *TxMempool) AddTx(tx *core.Tx, txFee int64) bool {
	//todo; send signal to all interesting the caller.
	m.Lock()
	defer m.Unlock()

	ancestors := m.calculateMemPoolAncestors(tx, true)

	// insert new txentry to the mempool; and update the mempool's memory consume.
	newTxEntry := NewTxentry(tx, txFee)
	m.PoolData[tx.Hash] = newTxEntry
	m.cacheInnerUsage += newTxEntry.usageSize

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

func (m *TxMempool) RemoveStaged(entriesToRemove map[*TxEntry]struct{}, updateDescendants bool, reason PoolRemovalReason) {

	for removeIt := range entriesToRemove {
		ancestors := m.calculateMemPoolAncestors(removeIt.tx, false)
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
// adds to setDescendants. Assumes entryit is already a tx in the mempool and
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
		ancestor.sumFee += entry.txFee
		ancestor.sumTxCount++
		ancestor.sumSize += uint64(entry.txSize)
	}
	ancestor.sumFee -= entry.txFee
	ancestor.sumTxCount--
	ancestor.sumSize -= uint64(entry.txSize)
}

// calculateMemPoolAncestors get tx all ancestors transaction in mempool.
// when the find is false: the tx must in mempool, so directly get his parent.
func (m *TxMempool) calculateMemPoolAncestors(tx *core.Tx, find bool) (ancestors map[*TxEntry]struct{}) {
	ancestors = make(map[*TxEntry]struct{})
	parentHash := make(map[*TxEntry]struct{})
	if find {
		for _, txin := range tx.Ins {
			if entry, ok := m.PoolData[txin.PreviousOutPoint.Hash]; ok {
				parentHash[entry] = struct{}{}
			}
		}
	} else {
		// If we're not searching for parents, we require this to be an entry in
		// the mempool already.
		if entry, ok := m.PoolData[tx.Hash]; ok {
			parentHash = entry.parentTx
		} else {
			panic("the tx must be in mempool")
		}
	}

	for entry := range parentHash {
		delete(parentHash, entry)
		ancestors[entry] = struct{}{}

		graTxentrys := entry.parentTx
		for gentry := range graTxentrys {
			if _, ok := ancestors[gentry]; !ok {
				parentHash[gentry] = struct{}{}
			}
		}
	}

	return
}

func (m *TxMempool) delTxentry(removeEntry *TxEntry, reason PoolRemovalReason) {
	//todo add signal for any subscriber

	for _, txin := range removeEntry.tx.Ins {
		delete(m.NextTx, *txin.PreviousOutPoint)
	}

	m.cacheInnerUsage -= removeEntry.usageSize
	delete(m.PoolData, removeEntry.tx.Hash)
}

func (m *TxMempool) FindTx(hash utils.Hash) *core.Tx {
	m.RLock()
	m.RUnlock()
	if find, ok := m.PoolData[hash]; ok{
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
