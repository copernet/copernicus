package mempool

import (
	"fmt"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/google/btree"
	"math"
	"testing"
)

type TestMemPoolEntry struct {
	Fee            utils.Amount
	Time           int64
	Priority       float64
	Height         int
	SpendsCoinbase bool
	SigOpCost      int
	lp             *core.LockPoints
}

func NewTestMemPoolEntry() *TestMemPoolEntry {
	t := TestMemPoolEntry{}
	t.Fee = 0
	t.Time = 0
	t.Priority = 0.0
	t.Height = 1
	t.SpendsCoinbase = false
	t.SigOpCost = 4
	t.lp = nil
	return &t
}

func (t *TestMemPoolEntry) SetFee(fee utils.Amount) *TestMemPoolEntry {
	t.Fee = fee
	return t
}

func (t *TestMemPoolEntry) SetTime(time int64) *TestMemPoolEntry {
	t.Time = time
	return t
}

func (t *TestMemPoolEntry) SetHeight(height int) *TestMemPoolEntry {
	t.Height = height
	return t
}

func (t *TestMemPoolEntry) SetSpendCoinbase(flag bool) *TestMemPoolEntry {
	t.SpendsCoinbase = flag
	return t
}

func (t *TestMemPoolEntry) SetSigOpsCost(sigOpsCost int) *TestMemPoolEntry {
	t.SigOpCost = sigOpsCost
	return t
}

func (t *TestMemPoolEntry) FromTxToEntry(tx *core.Tx) *TxEntry {
	lp := core.LockPoints{}
	if t.lp != nil {
		lp = *(t.lp)
	}
	entry := NewTxentry(tx, int64(t.Fee), t.Time, int(t.Height), lp, int(t.SigOpCost), t.SpendsCoinbase)
	return entry
}

func TestTxMempoolAddTx(t *testing.T) {
	testEntryHelp := NewTestMemPoolEntry()

	txParentPtr := core.NewTx()
	txParentPtr.Ins = make([]*core.TxIn, 1)
	txParentPtr.Ins[0] = core.NewTxIn(core.NewOutPoint(utils.HashOne, 0), []byte{core.OP_11})
	txParentPtr.Outs = make([]*core.TxOut, 3)
	for i := 0; i < 3; i++ {
		txParentPtr.Outs[i] = core.NewTxOut(33000, []byte{core.OP_11, core.OP_EQUAL})
	}
	txParentPtr.Hash = txParentPtr.TxHash()

	var txChild [3]core.Tx
	for i := 0; i < 3; i++ {
		txChild[i].Ins = make([]*core.TxIn, 1)
		txChild[i].Ins[0] = core.NewTxIn(core.NewOutPoint(txParentPtr.Hash, uint32(i)), []byte{core.OP_11})
		txChild[i].Outs = make([]*core.TxOut, 1)
		txChild[i].Outs[0] = core.NewTxOut(11000, []byte{core.OP_11, core.OP_EQUAL})
		txChild[i].Hash = txChild[i].TxHash()
	}

	var txGrandChild [3]core.Tx
	for i := 0; i < 3; i++ {
		txGrandChild[i].Ins = make([]*core.TxIn, 1)
		txGrandChild[i].Ins[0] = core.NewTxIn(core.NewOutPoint(txChild[i].Hash, 0), []byte{core.OP_11})
		txGrandChild[i].Outs = make([]*core.TxOut, 1)
		txGrandChild[i].Outs[0] = core.NewTxOut(11000, []byte{core.OP_11, core.OP_EQUAL})
		txGrandChild[i].Hash = txGrandChild[i].TxHash()
	}

	testPool := NewTxMempool()
	poolSize := testPool.Size()
	noLimit := uint64(math.MaxUint64)

	// Nothing in pool, remove should do nothing:
	testPool.removeTxRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize {
		t.Errorf("current poolSize : %d, except the poolSize : %d\n",
			testPool.Size(), poolSize)
		return
	}

	// Just add the parent:

	if err := testPool.AddTx(testEntryHelp.FromTxToEntry(txParentPtr), noLimit, noLimit, noLimit, noLimit, true); err != nil {
		t.Error("add Tx failure : ", err)
		return
	}
	poolSize = testPool.Size()
	testPool.removeTxRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize-1 {
		t.Errorf("current poolSize : %d, except the poolSize : %d\n",
			testPool.Size(), poolSize-1)
		return
	}

	// Parent, children, grandchildren:
	testPool.AddTx(testEntryHelp.FromTxToEntry(txParentPtr), noLimit, noLimit, noLimit, noLimit, true)
	for i := 0; i < 3; i++ {
		testPool.AddTx(testEntryHelp.FromTxToEntry(&txChild[i]), noLimit, noLimit, noLimit, noLimit, true)
		testPool.AddTx(testEntryHelp.FromTxToEntry(&txGrandChild[i]), noLimit, noLimit, noLimit, noLimit, true)
	}
	poolSize = testPool.Size()
	if poolSize != 7 {
		t.Errorf("current poolSize : %d, except the poolSize 7 ", poolSize)
		return
	}

	// Remove Child[0], GrandChild[0] should be removed:
	testPool.removeTxRecursive(&txChild[0], UNKNOWN)
	if poolSize-2 != testPool.Size() {
		t.Errorf("current poolSize : %d, except the poolSize %d ", testPool.Size(), poolSize-2)
		return
	}

	// ... make sure grandchild and child are gone:
	poolSize = testPool.Size()
	testPool.removeTxRecursive(&txGrandChild[0], UNKNOWN)
	if testPool.Size() != poolSize {
		t.Errorf("current poolSize : %d, except the poolSize %d ", testPool.Size(), poolSize)
		return
	}
	poolSize = testPool.Size()
	testPool.removeTxRecursive(&txChild[0], UNKNOWN)
	if testPool.Size() != poolSize {
		t.Errorf("current poolSize : %d, except the poolSize %d ", testPool.Size(), poolSize)
		return
	}

	// Remove parent, all children/grandchildren should go:
	poolSize = testPool.Size()
	testPool.removeTxRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize-5 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), poolSize-5)
		return
	}

	// Add children and grandchildren, but NOT the parent (simulate the parent
	// being in a block)
	for i := 0; i < 3; i++ {
		testPool.AddTx(testEntryHelp.FromTxToEntry(&txChild[i]), noLimit, noLimit, noLimit, noLimit, true)
		testPool.AddTx(testEntryHelp.FromTxToEntry(&txGrandChild[i]), noLimit, noLimit, noLimit, noLimit, true)
	}
	// Now remove the parent, as might happen if a block-re-org occurs but the
	// parent cannot be put into the mempool (maybe because it is non-standard):
	poolSize = testPool.Size()
	testPool.removeTxRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize-6 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), poolSize-6)
		return
	}
}

func createTx() []*TxEntry {

	testEntryHelp := NewTestMemPoolEntry()
	tx1 := core.NewTx()
	tx1.Ins = make([]*core.TxIn, 0)
	tx1.Outs = make([]*core.TxOut, 1)
	tx1.Outs[0] = core.NewTxOut(10*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	tx1.Hash = tx1.TxHash()
	txentry1 := testEntryHelp.SetTime(10000).FromTxToEntry(tx1)

	tx2 := core.NewTx()
	tx2.Ins = make([]*core.TxIn, 0)
	tx2.Outs = make([]*core.TxOut, 1)
	tx2.Outs[0] = core.NewTxOut(2*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	tx2.Hash = tx2.TxHash()
	txentry2 := testEntryHelp.SetTime(20000).FromTxToEntry(tx2)

	tx3 := core.NewTx()
	tx3.Ins = make([]*core.TxIn, 1)
	tx3.Outs = make([]*core.TxOut, 1)
	tx3.Outs[0] = core.NewTxOut(5*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	tx3.Ins[0] = core.NewTxIn(core.NewOutPoint(tx1.Hash, 0), []byte{core.OP_11, core.OP_EQUAL})
	tx3.Hash = tx3.TxHash()
	txentry3 := testEntryHelp.SetTime(15000).FromTxToEntry(tx3)

	tx4 := core.NewTx()
	tx4.Ins = make([]*core.TxIn, 0)
	tx4.Outs = make([]*core.TxOut, 1)
	tx4.Outs[0] = core.NewTxOut(6*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	tx4.Hash = tx4.TxHash()
	txentry4 := testEntryHelp.SetTime(25300).FromTxToEntry(tx4)
	t := make([]*TxEntry, 4)

	t[0] = txentry1
	t[1] = txentry2
	t[2] = txentry3
	t[3] = txentry4
	return t
}

func TestMempoolSortTime(t *testing.T) {
	testPool := NewTxMempool()
	noLimit := uint64(math.MaxUint64)

	set := createTx()
	for _, e := range set {
		testPool.AddTx(e, noLimit, noLimit, noLimit, noLimit, true)
	}

	sortedOrder := make([]utils.Hash, 4)
	sortedOrder[0] = set[0].Tx.Hash //10000
	sortedOrder[1] = set[2].Tx.Hash //15000
	sortedOrder[2] = set[1].Tx.Hash //20000
	sortedOrder[3] = set[3].Tx.Hash //25300

	if len(testPool.poolData) != len(sortedOrder) {
		t.Error("the pool element number is error, expect 4, but actual is ", len(testPool.poolData))
	}
	index := 0
	testPool.timeSortData.Ascend(func(i btree.Item) bool {
		entry := i.(*TxEntry)
		if entry.Tx.Hash != sortedOrder[index] {
			t.Errorf("the sort is error, index : %d, expect hash : %s, actual hash is : %s\n",
				index, sortedOrder[index].ToString(), entry.Tx.Hash.ToString())
			return true
		}
		index++
		return true
	})

	testPool.expire(5000)
	if testPool.Size() != 4 {
		t.Error("after the expire time, the pool should have 4 element, but actual number is : ", testPool.Size())
	}

	testPool.expire(11000)
	if testPool.Size() != 2 {
		t.Error("after the expire time, the pool should have 2 element, but actual number is : ", testPool.Size())
	}

	testPool.expire(300000)
	if testPool.Size() != 0 {
		t.Error("after the expire time, the pool should have 0 element, but actual number is : ", testPool.Size())
	}
}

func TestTxMempoolTrimToSize(t *testing.T) {
	testPool := NewTxMempool()
	noLimit := uint64(math.MaxUint64)

	set := createTx()
	fmt.Println("tx number : ", len(set))
	for _, e := range set {
		testPool.AddTx(e, noLimit, noLimit, noLimit, noLimit, true)
		fmt.Printf("entry size : %d, hash : %s, mempool size : %d \n", e.usageSize, e.Tx.Hash.ToString(), testPool.cacheInnerUsage)
	}
	fmt.Println("mempool usage size : ", testPool.cacheInnerUsage)

	testPool.trimToSize(testPool.cacheInnerUsage)
	if testPool.Size() != len(set) {
		t.Errorf("the pool element number is error, expect number is : %d, actual number is : %d", len(set), testPool.Size())
	}
	fmt.Printf("============= end ============\n")
	testPool.trimToSize(int64(set[0].usageSize + set[1].usageSize))

	testPool.trimToSize(1)
	if testPool.Size() != 0 {
		t.Errorf("the pool element number is error, expect number is : %d, actual number is : %d", 0, testPool.Size())
	}
	if testPool.cacheInnerUsage != 0 {
		t.Errorf("current the mempool size should be 0 byte, actual pool size is %d\n", testPool.cacheInnerUsage)
	}
	fmt.Printf("============= end ============\n")
}
