package mempool

import (
	"github.com/btcboost/copernicus/core"

	"time"
	"unsafe"

	"btree"
	"github.com/btcboost/copernicus/utils"
)

//TxEntry are not safe for concurrent write and read access .
type TxEntry struct {
	tx       *core.Tx
	txHeight int
	txSize   int
	// txFee tis transaction fee
	txFee     int64
	txFeeRate utils.FeeRate
	// sumTxCount is this tx and all Descendants transaction's number.
	sumTxCountWithDescendants uint64
	// sumFee is calculated by this tx and all Descendants transaction;
	sumFeeWithDescendants int64
	// sumSize size calculated by this tx and all Descendants transaction;
	sumSizeWithDescendants     uint64
	sumTxCountWithAncestors    uint64
	sumSizeWitAncestors        uint64
	sumSigOpCountWithAncestors uint64
	// time Local time when entering the mempool
	time int64
	// usageSize and total memory usage
	usageSize int64
	// childTx the tx's all Descendants transaction
	childTx map[*TxEntry]struct{}
	// parentTx the tx's all Ancestors transaction
	parentTx map[*TxEntry]struct{}
	//lp Track the height and time at which tx was final
	lp core.LockPoints
	//spendsCoinbase keep track of transactions that spend a coinbase
	spendsCoinbase bool
}

func (t *TxEntry) GetTxFromTxEntry() *core.Tx {
	return t.tx
}

func (t *TxEntry) SetLockPointFromTxEntry(lp core.LockPoints) {
	t.lp = lp
}

func (t *TxEntry) GetLockPointFromTxEntry() core.LockPoints {
	return t.lp
}

func (t *TxEntry) UpdateForDescendants(addTx *TxEntry) {

}

func (t *TxEntry) UpdateEntryForAncestors() {

}

func (t *TxEntry) GetSpendsCoinbase() bool {
	return t.spendsCoinbase
}

//UpdateParent update the tx's parent transaction.
func (t *TxEntry) UpdateParent(parent *TxEntry, add bool) {
	if add {
		t.parentTx[parent] = struct{}{}
	}
	delete(t.parentTx, parent)
}

func (t *TxEntry) Less(than btree.Item) bool {
	return t.time < than.(*TxEntry).time
}

func NewTxentry(tx *core.Tx, txFee int64) *TxEntry {
	t := new(TxEntry)
	t.tx = tx
	t.time = time.Now().Unix()
	t.txSize = tx.SerializeSize()
	t.sumSizeWithDescendants = uint64(t.txSize)
	t.txFee = txFee
	t.txFeeRate = utils.NewFeeRateWithSize(txFee, t.txSize)
	t.usageSize = int64(t.txSize) + int64(unsafe.Sizeof(t.txSize)*2+
		unsafe.Sizeof(t.sumTxCountWithDescendants)*4+unsafe.Sizeof(t.txFeeRate))
	t.parentTx = make(map[*TxEntry]struct{})
	t.childTx = make(map[*TxEntry]struct{})
	t.sumFeeWithDescendants += txFee
	t.sumTxCountWithDescendants++

	return t
}
