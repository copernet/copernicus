package mempool

import (
	"sync"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

// TxMempool is safe for concurrent write And read access.
type TxMempool struct {
	sync.RWMutex
	// fee the transaction enter the mempool mininal fee.
	fee int64
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

	return true
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
