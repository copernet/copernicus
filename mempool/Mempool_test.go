package mempool

import (
	"bytes"
	"testing"

	//"github.com/btcboost/copernicus/algorithm"
	"sort"

	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
	"gopkg.in/fatih/set.v0"
)

func fromTxToEntry(tx *model.Tx, fee btcutil.Amount, time int64, priority float64, pool *Mempool) *TxMempoolEntry {
	var inChainValue btcutil.Amount
	if pool != nil && pool.HasNoInputsOf(tx) {
		inChainValue = btcutil.Amount(tx.GetValueOut())
	}
	entry := NewTxMempoolEntry(tx, fee, 0, priority, 1, inChainValue, false, 4, nil)
	return entry
}

func TestMempoolAddUnchecked(t *testing.T) {
	txParentPtr := model.NewTx()
	txParentPtr.Ins = make([]*model.TxIn, 1)
	txParentPtr.Ins[0] = model.NewTxIn(model.NewOutPoint(&utils.HashOne, 0), []byte{model.OP_11})
	txParentPtr.Outs = make([]*model.TxOut, 3)
	for i := 0; i < 3; i++ {
		txParentPtr.Outs[i] = model.NewTxOut(33000, []byte{model.OP_11, model.OP_EQUAL})
	}
	parentBuf := bytes.NewBuffer(nil)
	txParentPtr.Serialize(parentBuf)
	parentHash := core.DoubleSha256Hash(parentBuf.Bytes())
	txParentPtr.Hash = parentHash
	var txChild [3]model.Tx
	for i := 0; i < 3; i++ {
		txChild[i].Ins = make([]*model.TxIn, 1)
		txChild[i].Ins[0] = model.NewTxIn(model.NewOutPoint(&parentHash, uint32(i)), []byte{model.OP_11})
		txChild[i].Outs = make([]*model.TxOut, 1)
		txChild[i].Outs[0] = model.NewTxOut(11000, []byte{model.OP_11, model.OP_EQUAL})
	}

	var txGrandChild [3]model.Tx
	for i := 0; i < 3; i++ {
		buf := bytes.NewBuffer(nil)
		txChild[i].Serialize(buf)
		txChildID := core.DoubleSha256Hash(buf.Bytes())
		txChild[i].Hash = txChildID
		txGrandChild[i].Ins = make([]*model.TxIn, 1)
		txGrandChild[i].Ins[0] = model.NewTxIn(model.NewOutPoint(&txChildID, 0), []byte{model.OP_11})
		txGrandChild[i].Outs = make([]*model.TxOut, 1)
		txGrandChild[i].Outs[0] = model.NewTxOut(11000, []byte{model.OP_11, model.OP_EQUAL})
		buf.Reset()
		txGrandChild[i].Serialize(buf)
		txGrandID := core.DoubleSha256Hash(buf.Bytes())
		txGrandChild[i].Hash = txGrandID
	}

	testPool := NewMemPool(utils.FeeRate{0})
	poolSize := testPool.Size()

	//Nothing in pool, remove should do nothing:
	testPool.RemoveRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize {
		t.Errorf("current poolSize : %d, except the poolSize : %d\n",
			testPool.Size(), poolSize)
	}

	//Just add the parent:
	if !testPool.AddUnchecked(&txParentPtr.Hash, fromTxToEntry(txParentPtr, 0, 0, 0, nil), true) {
		t.Error("add Tx failure ...")
	}
	poolSize = testPool.Size()
	testPool.RemoveRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize-1 {
		t.Errorf("current poolSize : %d, except the poolSize : %d\n",
			testPool.Size(), poolSize-1)
	}

	// Parent, children, grandchildren:
	testPool.AddUnchecked(&txParentPtr.Hash, fromTxToEntry(txParentPtr, 0, 0, 0, nil), true)
	for i := 0; i < 3; i++ {
		testPool.AddUnchecked(&txChild[i].Hash, fromTxToEntry(&txChild[i], 0, 0, 0, nil), true)
		testPool.AddUnchecked(&txGrandChild[i].Hash, fromTxToEntry(&txGrandChild[i], 0, 0, 0, nil), true)
	}
	poolSize = testPool.Size()
	if poolSize != 7 {
		t.Errorf("current poolSize : %d, except the poolSize 7 ", poolSize)
	}

	// Remove Child[0], GrandChild[0] should be removed:
	testPool.RemoveRecursive(&txChild[0], UNKNOWN)
	if poolSize-2 != testPool.Size() {
		t.Errorf("current poolSize : %d, except the poolSize %d ", testPool.Size(), poolSize-2)
	}

	// ... make sure grandchild and child are gone:
	poolSize = testPool.Size()
	testPool.RemoveRecursive(&txGrandChild[0], UNKNOWN)
	if testPool.Size() != poolSize {
		t.Errorf("current poolSize : %d, except the poolSize %d ", testPool.Size(), poolSize)
	}
	poolSize = testPool.Size()
	testPool.RemoveRecursive(&txChild[0], UNKNOWN)
	if testPool.Size() != poolSize {
		t.Errorf("current poolSize : %d, except the poolSize %d ", testPool.Size(), poolSize)
	}

	// Remove parent, all children/grandchildren should go:
	poolSize = testPool.Size()
	testPool.RemoveRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize-5 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), poolSize-5)
	}

	// Add children and grandchildren, but NOT the parent (simulate the parent
	// being in a block)
	for i := 0; i < 3; i++ {
		testPool.AddUnchecked(&txChild[i].Hash, fromTxToEntry(&txChild[i], 0, 0, 0, nil), true)
		testPool.AddUnchecked(&txGrandChild[i].Hash, fromTxToEntry(&txGrandChild[i], 0, 0, 0, nil), true)
	}
	// Now remove the parent, as might happen if a block-re-org occurs but the
	// parent cannot be put into the mempool (maybe because it is non-standard):
	poolSize = testPool.Size()
	testPool.RemoveRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize-6 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), poolSize-6)
	}

}

func TestMempoolClear(t *testing.T) {
	txParentPtr := model.NewTx()
	txParentPtr.Ins = make([]*model.TxIn, 1)
	txParentPtr.Ins[0] = model.NewTxIn(model.NewOutPoint(&utils.HashOne, 0), []byte{model.OP_11})
	txParentPtr.Outs = make([]*model.TxOut, 3)
	for i := 0; i < 3; i++ {
		txParentPtr.Outs[i] = model.NewTxOut(33000, []byte{model.OP_11, model.OP_EQUAL})
	}
	testPool := NewMemPool(utils.FeeRate{0})

	// Nothing in pool, clear should do nothing:
	testPool.Clear()
	if testPool.Size() != 0 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 0)
	}

	// Add the transaction
	testPool.AddUnchecked(&txParentPtr.Hash, fromTxToEntry(txParentPtr, 0, 0, 0, nil), true)
	if testPool.Size() != 1 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 1)
	}
	if testPool.MapNextTx.Size() != 1 {
		t.Errorf("current testPool.MapNextTx : %d, except the poolSize : %d", testPool.MapNextTx.Size(), 1)
	}
	if len(testPool.vTxHashes) != 1 {
		t.Errorf("current testPool.vTxHashes : %d, except the poolSize : %d", len(testPool.vTxHashes), 1)
	}

	// CTxMemPool's members should be empty after a clear
	testPool.Clear()
	if testPool.Size() != 0 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 0)
	}
	if testPool.MapNextTx.Size() != 0 {
		t.Errorf("current testPool.MapNextTx : %d, except the poolSize : %d", testPool.MapNextTx.Size(), 0)
	}
	if len(testPool.vTxHashes) != 0 {
		t.Errorf("current testPool.vTxHashes : %d, except the poolSize : %d", len(testPool.vTxHashes), 0)
	}
}

//there to be compare mempool store tx sorted And manual sorted, their sort should be the same
//todo add a element in the function to compare different type Sort
func checkSort(pool *Mempool, sortedOrder []utils.Hash) error {
	if pool.Size() != len(sortedOrder) {
		return errors.Errorf("current pool Size() : %d, sortSlice SIze() : %d, the two size should be equal",
			pool.Size(), len(sortedOrder))
	}
	sort.SliceStable(sortedOrder, func(i, j int) bool {
		tx1 := pool.MapTx[sortedOrder[i]]
		tx2 := pool.MapTx[sortedOrder[j]]
		return CompareTxMemPoolEntryByDescendantScore(tx1, tx2)
	})

	return nil
}

func TestMempoolEstimatePriority(t *testing.T) {
	testPool := NewMemPool(utils.FeeRate{0})

	/* 3rd highest fee */
	tx1 := model.NewTx()
	tx1.Ins = make([]*model.TxIn, 0)
	tx1.Outs = make([]*model.TxOut, 1)
	tx1.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx1.Hash = tx1.TxHash()
	testPool.AddUnchecked(&tx1.Hash, fromTxToEntry(tx1, 10000, 0, 10.0, nil), true)

	/* highest fee */
	tx2 := model.NewTx()
	tx2.Ins = make([]*model.TxIn, 0)
	tx2.Outs = make([]*model.TxOut, 1)
	tx2.Outs[0] = model.NewTxOut(2*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx2.Hash = tx2.TxHash()
	testPool.AddUnchecked(&tx2.Hash, fromTxToEntry(tx2, 20000, 0, 9.0, nil), true)

	/* lowest fee */
	tx3 := model.NewTx()
	tx3.Ins = make([]*model.TxIn, 0)
	tx3.Outs = make([]*model.TxOut, 1)
	tx3.Outs[0] = model.NewTxOut(5*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx3.Hash = tx3.TxHash()
	testPool.AddUnchecked(&tx3.Hash, fromTxToEntry(tx3, 0, 0, 100.0, nil), true)

	/* 2nd highest fee */
	tx4 := model.NewTx()
	tx4.Ins = make([]*model.TxIn, 0)
	tx4.Outs = make([]*model.TxOut, 1)
	tx4.Outs[0] = model.NewTxOut(6*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx4.Hash = tx4.TxHash()
	testPool.AddUnchecked(&tx4.Hash, fromTxToEntry(tx4, 15000, 0, 1.0, nil), true)

	/* equal fee rate to tx1, but newer */
	tx5 := model.NewTx()
	tx5.Ins = make([]*model.TxIn, 0)
	tx5.Outs = make([]*model.TxOut, 1)
	tx5.Outs[0] = model.NewTxOut(11*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx5.Hash = tx5.TxHash()
	testPool.AddUnchecked(&tx5.Hash, fromTxToEntry(tx5, 10000, 1, 10.0, nil), true)
	if testPool.Size() != 5 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 5)
	}

	sortedOrder := make([]utils.Hash, 0, 5)
	sortedOrder = append(sortedOrder, tx3.Hash) //0
	sortedOrder = append(sortedOrder, tx5.Hash) //10000
	sortedOrder = append(sortedOrder, tx1.Hash) //10000
	sortedOrder = append(sortedOrder, tx4.Hash) //15000
	sortedOrder = append(sortedOrder, tx2.Hash) //20000
	checkSort(testPool, sortedOrder)

	/* low fee but with high fee child */
	/* tx6 -> tx7 -> tx8, tx9 -> tx10 */
	tx6 := model.NewTx()
	tx6.Ins = make([]*model.TxIn, 0)
	tx6.Outs = make([]*model.TxOut, 1)
	tx6.Outs[0] = model.NewTxOut(20*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx6.Hash = tx6.TxHash()
	testPool.AddUnchecked(&tx6.Hash, fromTxToEntry(tx6, 0, 0, 0, nil), true)
	if testPool.Size() != 6 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 6)
	}
	// Check that at this point, tx6 is sorted low
	tmpSorted := make([]utils.Hash, 0, 6)
	tmpSorted = append(tmpSorted, tx6.Hash)
	tmpSorted = append(tmpSorted, sortedOrder...)
	sortedOrder = tmpSorted
	checkSort(testPool, sortedOrder)

	setAncestors := set.New()
	setAncestors.Add(testPool.MapTx[tx6.Hash])
	tx7 := model.NewTx()
	tx7.Ins = make([]*model.TxIn, 1)
	tx7.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx6.Hash, 0), []byte{model.OP_11})
	tx7.Outs = make([]*model.TxOut, 2)
	tx7.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx7.Outs[1] = model.NewTxOut(1*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx7.Hash = tx7.TxHash()

	setAncestorsCalculated := set.New()
	testPool.CalculateMemPoolAncestors(fromTxToEntry(tx7, 0, 0, 0, nil), setAncestorsCalculated,
		100, 1000000, 1000, 1000000, true)
	if !setAncestorsCalculated.IsEqual(setAncestors) {
		t.Errorf("setAncestorsCalculated.Size() : %d, setAncestors.Size() : %d, their should be equal"+
			"\n setAncestorsCalculated : %v,\n setAncestors : %v \n setAncestorsCalculated : %v, setAncestors : %v",
			setAncestorsCalculated.Size(), setAncestors.Size(), setAncestorsCalculated.List()[0], setAncestors.List()[0],
			setAncestorsCalculated.List(), setAncestors.List())
	}

	testPool.AddUnchecked(&tx7.Hash, fromTxToEntry(tx7, 0, 0, 0, nil), true)
	if testPool.Size() != 7 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 7)
	}

	// Now tx6 should be sorted higher (high fee child): tx7, tx6, tx2, ...
	tmpSorted = make([]utils.Hash, 0, 7)
	tmpSorted = append(tmpSorted, sortedOrder[1:]...)
	tmpSorted = append(tmpSorted, sortedOrder[0], tx6.Hash, tx7.Hash)
	sortedOrder = tmpSorted
	checkSort(testPool, sortedOrder)

	/* low fee child of tx7 */
	tx8 := model.NewTx()
	tx8.Ins = make([]*model.TxIn, 1)
	tx8.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx7.Hash, 0), []byte{model.OP_11})
	tx8.Outs = make([]*model.TxOut, 1)
	tx8.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx8.Hash = tx8.TxHash()
	setAncestors.Add(testPool.MapTx[tx7.Hash])
	testPool.AddUncheckedWithAncestors(&tx8.Hash, fromTxToEntry(tx8, 0, 2, 0, nil), setAncestors, true)

	// Now tx8 should be sorted low, but tx6/tx both high
	tmpSorted = make([]utils.Hash, 0, 8)
	tmpSorted = append(tmpSorted, tx8.Hash)
	tmpSorted = append(tmpSorted, sortedOrder...)
	sortedOrder = tmpSorted
	checkSort(testPool, sortedOrder)

	/* low fee child of tx7 */
	tx9 := model.NewTx()
	tx9.Ins = make([]*model.TxIn, 1)
	tx9.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx7.Hash, 1), []byte{model.OP_11})
	tx9.Outs = make([]*model.TxOut, 1)
	tx9.Outs[0] = model.NewTxOut(1*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx9.Hash = tx9.TxHash()
	testPool.AddUncheckedWithAncestors(&tx9.Hash, fromTxToEntry(tx9, 0, 3, 0, nil), setAncestors, true)

	// tx9 should be sorted low
	if testPool.Size() != 9 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 9)
	}
	tmpSorted = make([]utils.Hash, 0, 9)
	tmpSorted = append(tmpSorted, tx9.Hash)
	tmpSorted = append(tmpSorted, sortedOrder...)
	sortedOrder = tmpSorted
	checkSort(testPool, sortedOrder)

	snapshotOrder := make([]utils.Hash, 9)
	copy(snapshotOrder, sortedOrder)
	setAncestors.Add(testPool.MapTx[tx8.Hash])
	setAncestors.Add(testPool.MapTx[tx9.Hash])

	/* tx10 depends on tx8 and tx9 and has a high fee*/
	tx10 := model.NewTx()
	tx10.Ins = make([]*model.TxIn, 2)
	tx10.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx8.Hash, 0), []byte{model.OP_11})
	tx10.Ins[1] = model.NewTxIn(model.NewOutPoint(&tx9.Hash, 0), []byte{model.OP_11})
	tx10.Outs = make([]*model.TxOut, 1)
	tx10.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx10.Hash = tx10.TxHash()
	setAncestorsCalculated.Clear()
	testPool.CalculateMemPoolAncestors(fromTxToEntry(tx10, 200000, 4, 0, nil), setAncestorsCalculated,
		100, 1000000, 1000, 1000000, true)
	if !setAncestorsCalculated.IsEqual(setAncestors) {
		t.Errorf("setAncestorsCalculated.Size() : %d, setAncestors.Size() : %d, their should be equal"+
			"\n setAncestorsCalculated : %v,\n setAncestors : %v \n setAncestorsCalculated : %v, setAncestors : %v",
			setAncestorsCalculated.Size(), setAncestors.Size(), setAncestorsCalculated.List()[0], setAncestors.List()[0],
			setAncestorsCalculated.List(), setAncestors.List())
	}
	testPool.AddUncheckedWithAncestors(&tx10.Hash, fromTxToEntry(tx10, 0, 0, 0, nil), setAncestors, true)

	/**
	*  tx8 and tx9 should both now be sorted higher
	*  Final order after tx10 is added:
	*
	*  tx3 = 0 (1)
	*  tx5 = 10000 (1)
	*  tx1 = 10000 (1)
	*  tx4 = 15000 (1)
	*  tx2 = 20000 (1)
	*  tx9 = 200k (2 txs)
	*  tx8 = 200k (2 txs)
	*  tx10 = 200k (1 tx)
	*  tx6 = 2.2M (5 txs)
	*  tx7 = 2.2M (4 txs)
	 */
	// take out tx9, tx8 from the beginning
	t.Log("sortedOrder.Size() : ", len(sortedOrder))
	tmpSorted = make([]utils.Hash, 0, 10)
	copy(tmpSorted, sortedOrder[2:5]) //3
	tmpSorted = append(tmpSorted, tx9.Hash)
	tmpSorted = append(tmpSorted, tx8.Hash) //5
	// tx10 is just before tx6
	tmpSorted = append(tmpSorted, tx10.Hash)
	tmpSorted = append(tmpSorted, sortedOrder[5:]...)
	t.Log("tmpSorted.Size() : ", len(tmpSorted))

	// there should be 10 transactions in the mempool
	if testPool.Size() != 10 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 10)
	}

	// Now try removing tx10 and verify the sort order returns to normal
	testPool.RemoveRecursive(testPool.MapTx[tx10.Hash].TxRef, UNKNOWN)
	checkSort(testPool, snapshotOrder)

	testPool.RemoveRecursive(testPool.MapTx[tx9.Hash].TxRef, UNKNOWN)
	testPool.RemoveRecursive(testPool.MapTx[tx8.Hash].TxRef, UNKNOWN)
	/* Now check the sort on the mining score index.
	 * Final order should be:
	 * tx7 (2M)
	 * tx2 (20k)
	 * tx4 (15000)
	 * tx1/tx5 (10000)
	 * tx3/6 (0)
	 * (Ties resolved by hash)
	 */
	sortedOrder = make([]utils.Hash, 0)
	sortedOrder = append(sortedOrder, tx7.Hash)
	sortedOrder = append(sortedOrder, tx2.Hash)
	sortedOrder = append(sortedOrder, tx4.Hash)
	if tx1.Hash.ToBigInt().Cmp(tx5.Hash.ToBigInt()) < 0 {
		sortedOrder = append(sortedOrder, tx5.Hash)
		sortedOrder = append(sortedOrder, tx1.Hash)
	} else {
		sortedOrder = append(sortedOrder, tx1.Hash)
		sortedOrder = append(sortedOrder, tx5.Hash)
	}
	if tx3.Hash.ToBigInt().Cmp(tx6.Hash.ToBigInt()) < 0 {
		sortedOrder = append(sortedOrder, tx6.Hash)
		sortedOrder = append(sortedOrder, tx3.Hash)
	} else {
		sortedOrder = append(sortedOrder, tx3.Hash)
		sortedOrder = append(sortedOrder, tx6.Hash)
	}
	//todo sort for mempool element

}

func TestMempoolApplyDeltas(t *testing.T) {
	testPool := NewMemPool(utils.FeeRate{0})

	//3rd highest fee
	tx1 := model.NewTx()
	tx1.Outs = make([]*model.TxOut, 1)
	tx1.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx1.Hash = tx1.TxHash()
	testPool.AddUnchecked(&tx1.Hash, fromTxToEntry(tx1, 10000, 0, 10.0, nil), true)

	// highest fee
	tx2 := model.NewTx()
	tx2.Outs = make([]*model.TxOut, 1)
	tx2.Outs[0] = model.NewTxOut(2*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx2.Hash = tx2.TxHash()
	testPool.AddUnchecked(&tx2.Hash, fromTxToEntry(tx2, 20000, 0, 9.0, nil), true)
	tx2Size := tx2.SerializeSize()

	// lowest fee
	tx3 := model.NewTx()
	tx3.Outs = make([]*model.TxOut, 1)
	tx3.Outs[0] = model.NewTxOut(5*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx3.Hash = tx3.TxHash()
	testPool.AddUnchecked(&tx3.Hash, fromTxToEntry(tx3, 0, 0, 100.0, nil), true)

	// 2nd highest fee
	tx4 := model.NewTx()
	tx4.Outs = make([]*model.TxOut, 1)
	tx4.Outs[0] = model.NewTxOut(6*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx4.Hash = tx4.TxHash()
	testPool.AddUnchecked(&tx4.Hash, fromTxToEntry(tx4, 15000, 0, 1.0, nil), true)

	// equal fee rate to tx1, but newer
	tx5 := model.NewTx()
	tx5.Outs = make([]*model.TxOut, 1)
	tx5.Outs[0] = model.NewTxOut(11*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx5.Hash = tx5.TxHash()
	testPool.AddUnchecked(&tx5.Hash, fromTxToEntry(tx5, 10000, 0, 1.0, nil), true)

	sortedOrder := make([]utils.Hash, 7)
	sortedOrder[0] = tx2.Hash
	sortedOrder[1] = tx4.Hash
	// tx1 and tx5 are both 10000
	// Ties are broken by hash, not timestamp, so determine which hash comes
	// first.
	if tx1.Hash.ToBigInt().Cmp(tx5.Hash.ToBigInt()) < 0 {
		sortedOrder[2] = tx1.Hash
		sortedOrder[3] = tx5.Hash
	} else {
		sortedOrder[2] = tx5.Hash
		sortedOrder[3] = tx1.Hash
	}
	sortedOrder[4] = tx3.Hash
	checkSort(testPool, sortedOrder)

	// low fee parent with high fee child
	// tx6 (0) -> tx7 (high)
	tx6 := model.NewTx()
	tx6.Outs = make([]*model.TxOut, 1)
	tx6.Outs[0] = model.NewTxOut(20*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx6.Hash = tx6.TxHash()
	testPool.AddUnchecked(&tx6.Hash, fromTxToEntry(tx6, 0, 0, 1.0, nil), true)
	tx6Size := tx6.SerializeSize()
	if testPool.Size() != 6 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 6)
	}
	// Ties are broken by hash
	if tx3.Hash.ToBigInt().Cmp(tx6.Hash.ToBigInt()) < 0 {
		sortedOrder[5] = tx6.Hash
	} else {
		sortedOrder[4] = tx6.Hash
		sortedOrder[5] = tx3.Hash
	}
	checkSort(testPool, sortedOrder)

	tx7 := model.NewTx()
	tx7.Ins = make([]*model.TxIn, 1)
	tx7.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx6.Hash, 0), []byte{model.OP_11})
	tx7.Outs = make([]*model.TxOut, 1)
	tx7.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx7.Hash = tx7.TxHash()
	tx7Size := tx7.SerializeSize()

	/* set the fee to just below tx2's feerate when including ancestor */
	fee := btcutil.Amount((20000/tx2Size)*(tx7Size+tx6Size) - 1)

	// CTxMemPoolEntry entry7(tx7, fee, 2, 10.0, 1, true);
	testPool.AddUnchecked(&tx7.Hash, fromTxToEntry(tx7, fee, 0, 1.0, nil), true)
	if testPool.Size() != 7 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 7)
	}
	tmpSort := make([]utils.Hash, 7)
	tmpSort[0] = sortedOrder[0]
	tmpSort[1] = tx7.Hash
	copy(tmpSort[2:], sortedOrder[1:])
	checkSort(testPool, tmpSort)
	/*
		//after tx6 is mined, tx7 should move up in the sort
		vtx := algorithm.NewVector()
		vtx.PushBack(tx6)
		testPool.RemoveForBlock(vtx, 1)
	*/

	sortedOrder = sortedOrder[:len(sortedOrder)-1]
	// Ties are broken by hash
	if tx3.Hash.ToBigInt().Cmp(tx6.Hash.ToBigInt()) < 0 {
		sortedOrder = sortedOrder[:len(sortedOrder)-1]
	} else {
		tmp := make([]utils.Hash, 0)
		tmp = append(tmp, sortedOrder[:len(sortedOrder)-2]...)
		tmp = append(tmp, sortedOrder[len(sortedOrder)-1])
		sortedOrder = tmp
	}
	tmpSort = make([]utils.Hash, 1)
	tmpSort[0] = tx7.Hash
	tmpSort = append(tmpSort, sortedOrder[:]...)
	checkSort(testPool, tmpSort)
}
