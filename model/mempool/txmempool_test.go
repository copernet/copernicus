package mempool

import (
	"fmt"
	"math"
	"testing"

	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	txin2 "github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/google/btree"
	"github.com/magiconair/properties/assert"
)

type TestMemPoolEntry struct {
	Fee            amount.Amount
	Time           int64
	Priority       float64
	Height         int32
	SpendsCoinbase bool
	SigOpCost      int
	lp             *LockPoints
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

func (t *TestMemPoolEntry) SetFee(fee amount.Amount) *TestMemPoolEntry {
	t.Fee = fee
	return t
}

func (t *TestMemPoolEntry) SetTime(time int64) *TestMemPoolEntry {
	t.Time = time
	return t
}

func (t *TestMemPoolEntry) SetHeight(height int32) *TestMemPoolEntry {
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

func (t *TestMemPoolEntry) FromTxToEntry(tx *tx.Tx) *TxEntry {
	lp := LockPoints{}
	if t.lp != nil {
		lp = *(t.lp)
	}
	entry := NewTxentry(tx, int64(t.Fee), t.Time, t.Height, lp, int(t.SigOpCost), t.SpendsCoinbase)
	return entry
}

func TestTxMempooladdTx(t *testing.T) {
	testEntryHelp := NewTestMemPoolEntry()

	txParentPtr := tx.NewTx(0, tx.TxVersion)
	txin := txin2.NewTxIn(&outpoint.OutPoint{Hash: util.HashOne, Index: 0}, script.NewScriptRaw([]byte{opcodes.OP_11}), script.SequenceFinal)
	txParentPtr.AddTxIn(txin)
	for i := 0; i < 3; i++ {
		o := txout.NewTxOut(33000, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
		txParentPtr.AddTxOut(o)
	}
	_ = txParentPtr.GetHash()

	var txChild [3]tx.Tx
	for i := 0; i < 3; i++ {
		ins := txin2.NewTxIn(&outpoint.OutPoint{Hash: txParentPtr.GetHash(), Index: uint32(i)}, script.NewScriptRaw([]byte{opcodes.OP_11}), script.SequenceFinal)
		txChild[i].AddTxIn(ins)
		outs := txout.NewTxOut(11000, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
		txChild[i].AddTxOut(outs)
		_ = txChild[i].GetHash()
	}

	var txGrandChild [3]tx.Tx
	for i := 0; i < 3; i++ {
		ins := txin2.NewTxIn(&outpoint.OutPoint{Hash: txChild[i].GetHash(), Index: uint32(0)}, script.NewScriptRaw([]byte{opcodes.OP_11}), script.SequenceFinal)
		txGrandChild[i].AddTxIn(ins)
		outs := txout.NewTxOut(11000, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
		txGrandChild[i].AddTxOut(outs)
		_ = txGrandChild[i].GetHash()
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
	ancestors, _ := testPool.CalculateMemPoolAncestors(txParentPtr, noLimit, noLimit, noLimit, noLimit, true)
	if err := testPool.AddTx(testEntryHelp.FromTxToEntry(txParentPtr), ancestors); err != nil {
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
	ancestors, _ = testPool.CalculateMemPoolAncestors(txParentPtr, noLimit, noLimit, noLimit, noLimit, true)
	testPool.AddTx(testEntryHelp.FromTxToEntry(txParentPtr), ancestors)
	for i := 0; i < 3; i++ {
		ancestors, _ := testPool.CalculateMemPoolAncestors(&txChild[i], noLimit, noLimit, noLimit, noLimit, true)
		testPool.AddTx(testEntryHelp.FromTxToEntry(&txChild[i]), ancestors)
		ancestors, _ = testPool.CalculateMemPoolAncestors(&txGrandChild[i], noLimit, noLimit, noLimit, noLimit, true)
		testPool.AddTx(testEntryHelp.FromTxToEntry(&txGrandChild[i]), ancestors)
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
		ancestors, _ := testPool.CalculateMemPoolAncestors(&txChild[i], noLimit, noLimit, noLimit, noLimit, true)
		testPool.AddTx(testEntryHelp.FromTxToEntry(&txChild[i]), ancestors)
		ancestors, _ = testPool.CalculateMemPoolAncestors(&txGrandChild[i], noLimit, noLimit, noLimit, noLimit, true)
		testPool.AddTx(testEntryHelp.FromTxToEntry(&txGrandChild[i]), ancestors)
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
	tx1 := tx.NewTx(0, tx.TxVersion)
	//ins := txin2.NewTxIn(&outpoint.OutPoint{txParentPtr.Hash, uint32(i)}, script.NewScriptRaw([]byte{opcodes.OP_11}), tx.MaxTxInSequenceNum)
	//txChild[i].AddTxIn(ins)
	outs := txout.NewTxOut(amount.Amount(10*util.COIN), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	tx1.AddTxOut(outs)
	_ = tx1.GetHash()
	txentry1 := testEntryHelp.SetTime(10000).FromTxToEntry(tx1)

	tx2 := tx.NewTx(0, tx.TxVersion)
	out2 := txout.NewTxOut(amount.Amount(2*util.COIN), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	tx2.AddTxOut(out2)
	_ = tx2.GetHash()
	txentry2 := testEntryHelp.SetTime(20000).FromTxToEntry(tx2)

	tx3 := tx.NewTx(0, tx.TxVersion)
	ins := txin2.NewTxIn(&outpoint.OutPoint{Hash: tx1.GetHash(), Index: 0}, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}), script.SequenceFinal)
	tx3.AddTxIn(ins)
	out3 := txout.NewTxOut(amount.Amount(5*util.COIN), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	tx3.AddTxOut(out3)
	_ = tx3.GetHash()
	txentry3 := testEntryHelp.SetTime(15000).FromTxToEntry(tx3)

	tx4 := tx.NewTx(0, tx.TxVersion)
	out4 := txout.NewTxOut(amount.Amount(6*util.COIN), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	tx4.AddTxOut(out4)
	_ = tx4.GetHash()
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
		ancestors, _ := testPool.CalculateMemPoolAncestors(e.Tx, noLimit, noLimit, noLimit, noLimit, true)
		testPool.AddTx(e, ancestors)
	}

	sortedOrder := make([]util.Hash, 4)
	sortedOrder[0] = set[0].Tx.GetHash() //10000
	sortedOrder[1] = set[2].Tx.GetHash() //15000
	sortedOrder[2] = set[1].Tx.GetHash() //20000
	sortedOrder[3] = set[3].Tx.GetHash() //25300

	if len(testPool.poolData) != len(sortedOrder) {
		t.Error("the pool element number is error, expect 4, but actual is ", len(testPool.poolData))
	}
	index := 0
	testPool.timeSortData.Ascend(func(i btree.Item) bool {
		entry := i.(*TxEntry)
		if entry.Tx.GetHash() != sortedOrder[index] {
			t.Errorf("the sort is error, index : %d, expect hash : %s, actual hash is : %v\n",
				index, sortedOrder[index], entry.Tx.GetHash())
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
		ancestors, _ := testPool.CalculateMemPoolAncestors(e.Tx, noLimit, noLimit, noLimit, noLimit, true)
		testPool.AddTx(e, ancestors)
		fmt.Printf("entry size : %d, hash : %v, mempool size : %d \n", e.usageSize, e.Tx.GetHash(), testPool.usageSize)
	}
	fmt.Println("mempool usage size : ", testPool.usageSize)

	testPool.trimToSize(testPool.usageSize)
	if testPool.Size() != len(set) {
		t.Errorf("the pool element number is error, expect number is : %d, actual number is : %d", len(set), testPool.Size())
	}
	fmt.Printf("============= end ============\n")
	testPool.trimToSize(int64(set[0].usageSize + set[1].usageSize))

	testPool.trimToSize(1)
	if testPool.Size() != 0 {
		t.Errorf("the pool element number is error, expect number is : %d, actual number is : %d", 0, testPool.Size())
	}
	if testPool.usageSize != 0 {
		t.Errorf("current the mempool size should be 0 byte, actual pool size is %d\n", testPool.usageSize)
	}
	fmt.Printf("============= end ============\n")
}

func TestTxMempool_GetCheckFrequency(t *testing.T) {
	mp := NewTxMempool()
	mp.checkFrequency = 10.0
	res := mp.GetCheckFrequency()
	assert.Equal(t, res, 10.0)

	res = GetInstance().GetCheckFrequency()
	assert.Equal(t, res, 0.0)
}

func TestTxMempool_PoolData(t *testing.T) {
	set := createTx()
	hash1 := set[0].Tx.GetHash()
	hash2 := set[2].Tx.GetHash()
	hash3 := set[1].Tx.GetHash()
	hash4 := set[3].Tx.GetHash()

	mp := NewTxMempool()
	mp.poolData[hash1] = set[0]
	mp.poolData[hash2] = set[1]
	mp.poolData[hash3] = set[2]
	mp.poolData[hash4] = set[3]

	txentry1 := mp.FindTx(hash1)
	assert.Equal(t, txentry1, set[0])
	txentry2 := mp.FindTx(hash2)
	assert.Equal(t, txentry2, set[1])
	txentry3 := mp.FindTx(hash3)
	assert.Equal(t, txentry3, set[2])
	txentry4 := mp.FindTx(hash4)
	assert.Equal(t, txentry4, set[3])

	res := mp.GetAllTxEntryWithoutLock()
	assert.Equal(t, res, mp.poolData)

	res1 := mp.GetAllTxEntry()
	assert.Equal(t, res1, mp.poolData)

	res2 := mp.CalculateMemPoolAncestorsWithLock(&hash2)
	tmpRes2 := make(map[*TxEntry]struct{})
	assert.Equal(t, res2, tmpRes2)

	res3 := mp.CalculateDescendantsWithLock(&hash2)
	tmpRes3 := make(map[*TxEntry]struct{})
	tmpRes3[set[1]] = struct{}{}
	assert.Equal(t, res3, tmpRes3)
}

func TestTxMempool_RootTx(t *testing.T) {
	set := createTx()

	hash1 := set[0].Tx.GetHash()
	mp := NewTxMempool()
	mp.rootTx[hash1] = set[0]
	rootTx := mp.GetRootTx()
	assert.Equal(t, rootTx[hash1], *mp.rootTx[hash1])

	outpoint1 := outpoint.NewOutPoint(hash1, 0x01)
	mp.nextTx[*outpoint1] = set[0]
	res := mp.GetAllSpentOutWithoutLock()
	assert.Equal(t, res, mp.nextTx)

	ok := mp.HasSpentOut(outpoint1)
	assert.Equal(t, ok, true)

	txentry := mp.HasSPentOutWithoutLock(outpoint1)
	wantTxEntry := mp.nextTx[*outpoint1]
	assert.Equal(t, txentry, wantTxEntry)
}

func TestTxMempool_GetCoin(t *testing.T) {
	mp := NewTxMempool()
	set := createTx()
	hash1 := set[0].Tx.GetHash()

	//return nil
	outpoint1 := outpoint.NewOutPoint(hash1, 0x01)
	coin1 := mp.GetCoin(outpoint1)
	if coin1 != nil {
		t.Errorf("error coin1:%v", coin1)
	}

	//return nil
	mp.poolData[hash1] = set[0]
	coin2 := mp.GetCoin(outpoint1)
	if coin2 != nil {
		t.Errorf("error coin2:%v", coin2)
	}

	//return coin
	out := set[0].Tx.GetTxOut(0x00)
	outpoint2 := outpoint.NewOutPoint(hash1, 0x00)
	coin3 := mp.GetCoin(outpoint2)
	assert.Equal(t, out.GetValue(), coin3.GetAmount())
	assert.Equal(t, out.GetScriptPubKey(), coin3.GetScriptPubKey())
}
