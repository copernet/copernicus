package mempool

import (
	"unsafe"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/google/btree"
)

// TxEntry are not safe for concurrent write and read access .
type TxEntry struct {
	tx     *core.Tx
	txSize int
	// txFee tis transaction fee
	txFee int64
	// sumTxCountWithDescendants is this tx and all Descendants transaction's number. init number is 1.
	sumTxCountWithDescendants int64
	// sumFeeWithDescendants is calculated by this tx and all Descendants transaction;
	sumFeeWithDescendants int64
	// sumSizeWithDescendants size calculated by this tx and all Descendants transaction;
	sumSizeWithDescendants int64

	sumTxCountWithAncestors    int64
	sumSizeWitAncestors        int64
	sumSigOpCountWithAncestors int64
	sumFeeWithAncestors        int64
	// sigOpCount sigop plus P2SH sigops count
	sigOpCount int
	// time Local time when entering the memPool
	time int64
	// usageSize and total memory usage
	usageSize int
	// childTx the tx's all Descendants transaction
	childTx map[*TxEntry]struct{}
	// parentTx the tx's all Ancestors transaction
	parentTx map[*TxEntry]struct{}
	// lp Track the height and time at which tx was final
	lp core.LockPoints
	// spendsCoinBase keep track of transactions that spend a coinBase
	spendsCoinbase bool
}

func (t *TxEntry) GetTxCountWithDescendants() int64 {
	return t.sumTxCountWithDescendants
}

func (t *TxEntry) GetTxCountWithAncestors() int64 {
	return t.sumTxCountWithAncestors
}

func (t *TxEntry) GetSizeWithAncestors() int64 {
	return t.sumSizeWitAncestors
}

func (t *TxEntry) GetSizeWithDescendants() int64 {
	return t.sumSizeWithDescendants
}

func (t *TxEntry) GetSigOpCount() int {
	return t.sigOpCount
}

func (t *TxEntry) GetSigOpCountWithAncestors() int64 {
	return t.sumSigOpCountWithAncestors
}

func (t *TxEntry) GetTxSize() int {
	return t.txSize
}

func (t *TxEntry) GetUsageSize() int64 {
	return int64(t.usageSize)
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

func (t *TxEntry) GetSpendsCoinbase() bool {
	return t.spendsCoinbase
}

// UpdateParent update the tx's parent transaction.
func (t *TxEntry) UpdateParent(parent *TxEntry, innerUsage *int64, add bool) {
	if add {
		t.parentTx[parent] = struct{}{}
		*innerUsage += int64(unsafe.Sizeof(parent))
		return
	}
	delete(t.parentTx, parent)
	*innerUsage -= int64(unsafe.Sizeof(parent))
}

func (t *TxEntry) UpdateChild(child *TxEntry, innerUsage *int64, add bool) {
	if add {
		t.childTx[child] = struct{}{}
		*innerUsage += int64(unsafe.Sizeof(child))
		return
	}
	delete(t.childTx, child)
	*innerUsage -= int64(unsafe.Sizeof(child))
}

func (t *TxEntry) UpdateDescendantState(updateCount, updateSize int, updateFee int64) {
	t.sumTxCountWithDescendants += int64(updateCount)
	t.sumSizeWithDescendants += int64(updateSize)
	t.sumFeeWithDescendants += updateFee
}

func (t *TxEntry) UpdateAncestorState(updateCount, updateSize, updateSigOps int, updateFee int64) {
	t.sumSizeWitAncestors += int64(updateSize)
	t.sumTxCountWithAncestors += int64(updateCount)
	t.sumSigOpCountWithAncestors += int64(updateSigOps)
	t.sumFeeWithAncestors += updateFee
}

func (t *TxEntry) Less(than btree.Item) bool {
	return t.time < than.(*TxEntry).time
}

func NewTxentry(tx *core.Tx, txFee int64, acceptTime int64, lp core.LockPoints, sigOpsCount int, spendCoinbase bool) *TxEntry {
	t := new(TxEntry)
	t.tx = tx
	t.time = acceptTime
	t.txSize = tx.SerializeSize()
	t.txFee = txFee
	t.usageSize = t.txSize + int(unsafe.Sizeof(t.lp))
	t.spendsCoinbase = spendCoinbase
	t.lp = lp

	t.sumSizeWithDescendants = int64(t.txSize)
	t.sumFeeWithDescendants = txFee
	t.sumTxCountWithDescendants = 1

	t.sumFeeWithAncestors = txFee
	t.sumSizeWitAncestors = int64(t.txSize)
	t.sumTxCountWithAncestors = 1
	t.sumSigOpCountWithAncestors = int64(sigOpsCount)

	t.parentTx = make(map[*TxEntry]struct{})
	t.childTx = make(map[*TxEntry]struct{})

	return t
}

type EntryFeeSort TxEntry

func (e EntryFeeSort) Less(than btree.Item) bool {
	return e.sumFeeWithDescendants > than.(EntryFeeSort).sumFeeWithDescendants
}

type EntryFeeRateSort TxEntry

func (r EntryFeeRateSort) Less(than btree.Item) bool {
	t := than.(EntryFeeRateSort)
	return utils.NewFeeRateWithSize((r).sumFeeWithDescendants, r.sumSizeWithDescendants).SataoshisPerK >
		utils.NewFeeRateWithSize(t.sumFeeWithDescendants, t.sumSizeWithDescendants).SataoshisPerK
}
