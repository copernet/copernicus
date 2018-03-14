package mempool

import (
	"sync"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
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

func (m *TxMempool) DelTx(tx *core.Tx) bool {
	return true
}

func (m *TxMempool) FindTx(hash utils.Hash) *core.Tx {
	return nil
}

func (m *TxMempool) AcceptTx(tx *core.Tx) bool {
	return true
}

func (m *TxMempool) RejectTx(tx *core.Tx) bool {
	return true
}
