package mempool

import (
	"unsafe"

	"fmt"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/google/btree"
)

// TxEntry are not safe for concurrent write and read access .
type TxEntry struct {
	Tx     *core.Tx
	TxSize int
	// txFee tis transaction fee
	TxFee    int64
	TxHeight int
	// sigOpCount sigop plus P2SH sigops count
	SigOpCount int
	// time Local time when entering the memPool
	time int64
	// usageSize and total memory usage
	usageSize int
	// childTx the tx's all Descendants transaction
	ChildTx map[*TxEntry]struct{}
	// parentTx the tx's all Ancestors transaction
	ParentTx map[*TxEntry]struct{}
	// lp Track the height and time at which tx was final
	lp core.LockPoints
	// spendsCoinBase keep track of transactions that spend a coinBase
	spendsCoinbase bool
	//Statistics Information for every txentry with its ancestors And descend.
	StatisInformation
}

type StatisInformation struct {
	// sumTxCountWithDescendants is this tx and all Descendants transaction's number. init number is 1.
	SumTxCountWithDescendants int64
	// sumFeeWithDescendants is calculated by this tx and all Descendants transaction;
	SumFeeWithDescendants int64
	// sumSizeWithDescendants size calculated by this tx and all Descendants transaction;
	SumSizeWithDescendants int64

	SumTxCountWithAncestors    int64
	SumSizeWitAncestors        int64
	SumSigOpCountWithAncestors int64
	SumFeeWithAncestors        int64
}

func (t *TxEntry) GetSigOpCountWithAncestors() int64 {
	return t.SumSigOpCountWithAncestors
}

func (t *TxEntry) GetUsageSize() int64 {
	return int64(t.usageSize)
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
		t.ParentTx[parent] = struct{}{}
		fmt.Printf("should add tx1 to tx3 parent, ---------------- tx1 : %s, tx3 : %s\n", parent.Tx.Hash.ToString(), t.Tx.Hash.ToString())
		*innerUsage += int64(unsafe.Sizeof(parent))
		return
	}
	delete(t.ParentTx, parent)
	*innerUsage -= int64(unsafe.Sizeof(parent))
	fmt.Printf("should remove tx1 from tx3 parent, *************** tx1 : %s, tx3 : %s\n", parent.Tx.Hash.ToString(), t.Tx.Hash.ToString())
}

func (t *TxEntry) UpdateChild(child *TxEntry, innerUsage *int64, add bool) {
	if add {
		t.ChildTx[child] = struct{}{}
		fmt.Printf("should add tx3 to tx1 child ----------------, tx1 : %s, tx3 : %s\n", t.Tx.Hash.ToString(), child.Tx.Hash.ToString())
		*innerUsage += int64(unsafe.Sizeof(child))
		return
	}
	delete(t.ChildTx, child)
	fmt.Printf("should remove tx3 from tx1 child ***************, tx1 : %s, tx3 : %s\n", t.Tx.Hash.ToString(), child.Tx.Hash.ToString())
	*innerUsage -= int64(unsafe.Sizeof(child))
}

func (t *TxEntry) UpdateDescendantState(updateCount, updateSize int, updateFee int64) {
	t.SumTxCountWithDescendants += int64(updateCount)
	t.SumSizeWithDescendants += int64(updateSize)
	t.SumFeeWithDescendants += updateFee
}

func (t *TxEntry) UpdateAncestorState(updateCount, updateSize, updateSigOps int, updateFee int64) {
	t.SumSizeWitAncestors += int64(updateSize)
	t.SumTxCountWithAncestors += int64(updateCount)
	t.SumSigOpCountWithAncestors += int64(updateSigOps)
	t.SumFeeWithAncestors += updateFee
}

func (t *TxEntry) Less(than btree.Item) bool {
	th := than.(*TxEntry)
	if t.time == th.time {
		return t.Tx.Hash.Cmp(&th.Tx.Hash) > 0
	}
	return t.time < th.time
}

func NewTxentry(tx *core.Tx, txFee int64, acceptTime int64, height int, lp core.LockPoints, sigOpsCount int, spendCoinbase bool) *TxEntry {
	t := new(TxEntry)
	t.Tx = tx
	t.time = acceptTime
	t.TxSize = tx.SerializeSize()
	t.TxFee = txFee
	t.usageSize = t.TxSize + int(unsafe.Sizeof(t.lp))
	t.spendsCoinbase = spendCoinbase
	t.lp = lp
	t.TxHeight = height

	t.SumSizeWithDescendants = int64(t.TxSize)
	t.SumFeeWithDescendants = txFee
	t.SumTxCountWithDescendants = 1

	t.SumFeeWithAncestors = txFee
	t.SumSizeWitAncestors = int64(t.TxSize)
	t.SumTxCountWithAncestors = 1
	t.SumSigOpCountWithAncestors = int64(sigOpsCount)

	t.ParentTx = make(map[*TxEntry]struct{})
	t.ChildTx = make(map[*TxEntry]struct{})

	return t
}

func (t *TxEntry) GetFeeRate() *utils.FeeRate {
	return utils.NewFeeRateWithSize(t.TxFee, int64(t.TxSize))
}

func (t *TxEntry) GetInfo() *TxMempoolInfo {
	return &TxMempoolInfo{
		Tx:      t.Tx,
		Time:    t.time,
		FeeRate: *t.GetFeeRate(),
	}
}

type EntryFeeSort TxEntry

func (e EntryFeeSort) Less(than btree.Item) bool {
	t := than.(EntryFeeSort)
	if e.SumFeeWithAncestors == t.SumFeeWithAncestors {
		return e.Tx.Hash.Cmp(&t.Tx.Hash) > 0
	}
	return e.SumFeeWithAncestors > than.(EntryFeeSort).SumFeeWithAncestors
}

type EntryAncestorFeeRateSort TxEntry

func (r EntryAncestorFeeRateSort) Less(than btree.Item) bool {
	t := than.(EntryAncestorFeeRateSort)
	b1 := utils.NewFeeRateWithSize((r).SumFeeWithAncestors, r.SumSizeWitAncestors).SataoshisPerK
	b2 := utils.NewFeeRateWithSize(t.SumFeeWithAncestors, t.SumSizeWitAncestors).SataoshisPerK
	if b1 == b2 {
		return r.Tx.Hash.Cmp(&t.Tx.Hash) > 0
	}
	return b1 > b2
}
