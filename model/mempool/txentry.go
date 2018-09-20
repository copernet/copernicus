package mempool

import (
	"unsafe"

	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
	"github.com/google/btree"
)

type TxEntry struct {
	Tx     *tx.Tx
	TxSize int
	// txFee tis transaction fee
	TxFee    int64
	TxHeight int32
	// sigOpCount sigop plus P2SH sigops count
	SigOpCount int
	// time Local time when entering the memPool
	time int64
	// usageSize and total memory usage;
	usageSize int
	// childTx the tx's all Descendants transaction
	ChildTx map[*TxEntry]struct{}
	// parentTx the tx's all Ancestors transaction
	ParentTx map[*TxEntry]struct{}
	// lp Track the height and time at which tx was final
	lp LockPoints
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

func (t *TxEntry) SetLockPointFromTxEntry(lp LockPoints) {
	t.lp = lp
}

func (t *TxEntry) GetLockPointFromTxEntry() LockPoints {
	return t.lp
}

func (t *TxEntry) GetSpendsCoinbase() bool {
	return t.spendsCoinbase
}

func (t *TxEntry) GetTime() int64 {
	return t.time
}

// UpdateParent update the tx's parent transaction.
func (t *TxEntry) UpdateParent(parent *TxEntry, add bool) {
	if add {
		t.ParentTx[parent] = struct{}{}
		t.usageSize += int(unsafe.Sizeof(parent))
		return
	}
	delete(t.ParentTx, parent)
	t.usageSize -= int(unsafe.Sizeof(parent))
}

func (t *TxEntry) UpdateChild(child *TxEntry, add bool) {
	if add {
		t.ChildTx[child] = struct{}{}
		t.usageSize += int(unsafe.Sizeof(child))
		return
	}
	delete(t.ChildTx, child)
	t.usageSize -= int(unsafe.Sizeof(child))
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
		thash := t.Tx.GetHash()
		thhash := th.Tx.GetHash()
		return thash.Cmp(&thhash) > 0
	}
	return t.time < th.time
}

func NewTxentry(tx *tx.Tx, txFee int64, acceptTime int64, height int32, lp LockPoints, sigOpsCount int,
	spendCoinbase bool) *TxEntry {
	t := new(TxEntry)
	t.Tx = tx
	t.time = acceptTime
	t.TxSize = int(tx.SerializeSize())
	t.TxFee = txFee
	t.usageSize = t.TxSize + int(unsafe.Sizeof(*t))
	t.spendsCoinbase = spendCoinbase
	t.lp = lp
	t.TxHeight = height
	t.SigOpCount = sigOpsCount

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

func (t *TxEntry) GetFeeRate() *util.FeeRate {
	return util.NewFeeRateWithSize(t.TxFee, int64(t.TxSize))
}

func (t *TxEntry) GetInfo() *TxMempoolInfo {
	return &TxMempoolInfo{
		Tx:      t.Tx,
		Time:    t.time,
		FeeRate: *t.GetFeeRate(),
	}
}

func (t *TxEntry) CheckLockPointValidity(chain *chain.Chain) bool {
	if t.lp.MaxInputBlock != nil {
		if !chain.Contains(t.lp.MaxInputBlock) {
			return false
		}
	}
	return true
}

type EntryFeeSort TxEntry

func (e EntryFeeSort) Less(than btree.Item) bool {
	t := than.(EntryFeeSort)
	if e.SumFeeWithAncestors == t.SumFeeWithAncestors {
		thash := e.Tx.GetHash()
		thhash := t.Tx.GetHash()
		return thash.Cmp(&thhash) > 0
	}
	return e.SumFeeWithAncestors > than.(EntryFeeSort).SumFeeWithAncestors
}

type EntryAncestorFeeRateSort TxEntry

func (r EntryAncestorFeeRateSort) Less(than btree.Item) bool {
	t := than.(EntryAncestorFeeRateSort)
	b1 := util.NewFeeRateWithSize((r).SumFeeWithAncestors, r.SumSizeWitAncestors).SataoshisPerK
	b2 := util.NewFeeRateWithSize(t.SumFeeWithAncestors, t.SumSizeWitAncestors).SataoshisPerK
	if b1 == b2 {
		rhash := r.Tx.GetHash()
		thhash := t.Tx.GetHash()
		return rhash.Cmp(&thhash) > 0
	}
	return b1 > b2
}
