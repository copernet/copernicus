package mempool

import (
	"github.com/btcboost/copernicus/core"

	"time"
	"unsafe"

	"github.com/btcboost/copernicus/utils"
)

//TxEntry are not safe for concurrent write and read access .
type TxEntry struct {
	tx       *core.Tx
	txHeight int
	txSize   int
	// txFee tis transaction fee
	txFeeRate utils.FeeRate
	// sumTxCount is this tx and all Descendants transaction's number.
	sumTxCount uint64
	// sumFee is calculated by this tx and all Descendants transaction;
	sumFee utils.FeeRate
	// sumSize size calculated by this tx and all Descendants transaction;
	sumSize uint64
	// time Local time when entering the mempool
	time int64
	// usageSize and total memory usage
	usageSize int64
	// childTx the tx's all Descendants transaction
	childTx []*TxEntry
	// childTx the tx's all Ancestors transaction
	parentTx []*TxEntry
}

func (t *TxEntry) Less(c *TxEntry) bool {
	return t.txFeeRate.Less(c.txFeeRate)
}

func (t *TxEntry) UpdateForDescendants(addTx *TxEntry) {

}

func (t *TxEntry) UpdateEntryForAncestors() {

}

//UpdateParent update the tx's parent transaction.
func (t *TxEntry) UpdateParent(parent *TxEntry, add bool) {
	t.parentTx = append(t.parentTx, parent)
}

func NewTxentry(tx *core.Tx, txFee int64) *TxEntry {
	t := new(TxEntry)
	t.tx = tx
	t.time = time.Now().Unix()
	t.txSize = tx.SerializeSize()
	t.sumSize = uint64(t.txSize)
	t.txFeeRate = utils.NewFeeRateWithSize(txFee, t.txSize)
	t.usageSize = int64(t.txSize) + int64(unsafe.Sizeof(t.txSize)*2+
		unsafe.Sizeof(t.sumTxCount)*4+unsafe.Sizeof(t.txFeeRate))
	return t
}
