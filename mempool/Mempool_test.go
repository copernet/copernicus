package mempool

import (
	"bytes"
	"testing"

	"fmt"
	//"github.com/btcboost/copernicus/algorithm"

	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
	"gopkg.in/fatih/set.v0"
	//"math"
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
func checkSort(pool *Mempool, sortedOrder []utils.Hash, typeName int) error {
	if pool.Size() != len(sortedOrder) {
		return errors.Errorf("current pool Size() : %d, sortSlice SIze() : %d, the two size should be equal",
			pool.Size(), len(sortedOrder))
	}

	processFunc := func(keys []*TxMempoolEntry) error {
		for i, v := range keys {
			txEntry := v
			oriHash := txEntry.TxRef.Hash
			dstHash := sortedOrder[i]
			if !(&oriHash).IsEqual(&dstHash) {
				return errors.Errorf("pool store element : %v;\n except element : %v;\n sort index : %d, they should be equal",
					oriHash.ToString(), dstHash.ToString(), i)
			}
		}
		return nil
	}
	printSlice := func(keys []*TxMempoolEntry) {
		for i, v := range keys {
			fmt.Printf("******** pool txentry index : %d, element : %v\n", i, v.TxRef.Hash.ToString())
		}
	}
	printSort := func(sortSlice []utils.Hash) {
		for i, v := range sortSlice {
			fmt.Printf("-------- expect txentry index : %d, element : %v\n", i, v.ToString())
		}
	}
	printSort(sortedOrder)

	var err error
	switch typeName {
	case DESCENDANTSCORE:
		keys := pool.MapTx.GetByDescendantScoreSort()
		err = processFunc(keys)
		printSlice(keys)
	case ANCESTORSCORE:
		keys := pool.MapTx.GetbyAncestorFeeSort()
		err = processFunc(keys)
		printSlice(keys)
	case MININGSCORE:
		keys := pool.MapTx.GetbyScoreSort()
		err = processFunc(keys)
		printSlice(keys)
	}

	return err
}

func TestMempoolEstimatePriority(t *testing.T) {
	testPool := NewMemPool(utils.FeeRate{0})

	// 3rd highest fee
	tx1 := model.NewTx()
	tx1.Ins = make([]*model.TxIn, 0)
	tx1.Outs = make([]*model.TxOut, 1)
	tx1.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx1.Hash = tx1.TxHash()
	txentry1 := fromTxToEntry(tx1, 10000, 0, 10.0, nil)
	testPool.AddUnchecked(&tx1.Hash, txentry1, true)

	// highest fee
	tx2 := model.NewTx()
	tx2.Ins = make([]*model.TxIn, 0)
	tx2.Outs = make([]*model.TxOut, 1)
	tx2.Outs[0] = model.NewTxOut(2*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx2.Hash = tx2.TxHash()
	txentry2 := fromTxToEntry(tx2, 20000, 0, 9.0, nil)
	testPool.AddUnchecked(&tx2.Hash, txentry2, true)

	// lowest fee
	tx3 := model.NewTx()
	tx3.Ins = make([]*model.TxIn, 0)
	tx3.Outs = make([]*model.TxOut, 1)
	tx3.Outs[0] = model.NewTxOut(5*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx3.Hash = tx3.TxHash()
	txentry3 := fromTxToEntry(tx3, 0, 0, 100.0, nil)
	testPool.AddUnchecked(&tx3.Hash, txentry3, true)

	// 2nd highest fee
	tx4 := model.NewTx()
	tx4.Ins = make([]*model.TxIn, 0)
	tx4.Outs = make([]*model.TxOut, 1)
	tx4.Outs[0] = model.NewTxOut(6*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx4.Hash = tx4.TxHash()
	testPool.AddUnchecked(&tx4.Hash, fromTxToEntry(tx4, 15000, 0, 1.0, nil), true)

	// equal fee rate to tx1, but newer
	tx5 := model.NewTx()
	tx5.Ins = make([]*model.TxIn, 0)
	tx5.Outs = make([]*model.TxOut, 1)
	tx5.Outs[0] = model.NewTxOut(11*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx5.Hash = tx5.TxHash()
	txentry5 := fromTxToEntry(tx5, 10000, 1, 10.0, nil)
	testPool.AddUnchecked(&tx5.Hash, txentry5, true)
	if testPool.Size() != 5 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 5)
	}

	sortedOrder := make([]utils.Hash, 5)
	sortedOrder[0] = tx3.Hash //0
	sortedOrder[1] = tx5.Hash //10000
	sortedOrder[2] = tx1.Hash //10000
	sortedOrder[3] = tx4.Hash //15000
	sortedOrder[4] = tx2.Hash //20000
	err := checkSort(testPool, sortedOrder, DESCENDANTSCORE)
	if err != nil {
		t.Error(err)
		return
	}

	// low fee but with high fee child
	// tx6 -> tx7 -> tx8, tx9 -> tx10
	tx6 := model.NewTx()
	tx6.Ins = make([]*model.TxIn, 0)
	tx6.Outs = make([]*model.TxOut, 1)
	tx6.Outs[0] = model.NewTxOut(20*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx6.Hash = tx6.TxHash()
	txentry6 := fromTxToEntry(tx6, 0, 1, 10.0, nil)
	testPool.AddUnchecked(&tx6.Hash, txentry6, true)
	if testPool.Size() != 6 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 6)
	}

	// Check that at this point, tx6 is sorted low
	tmpSorted := make([]utils.Hash, 6)
	tmpSorted[0] = tx6.Hash
	copy(tmpSorted[1:], sortedOrder)
	sortedOrder = tmpSorted
	fmt.Println("===============  00 ------------------")
	err = checkSort(testPool, sortedOrder, DESCENDANTSCORE)
	if err != nil {
		t.Error(err)
		fmt.Println("===============  01 ------------------")
		return
	}

	setAncestors := set.New()
	setAncestors.Add(testPool.MapTx.GetEntryByHash(tx6.Hash))
	tx7 := model.NewTx()
	tx7.Ins = make([]*model.TxIn, 1)
	tx7.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx6.Hash, 0), []byte{model.OP_11})
	tx7.Outs = make([]*model.TxOut, 2)
	tx7.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx7.Outs[1] = model.NewTxOut(1*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx7.Hash = tx7.TxHash()

	setAncestorsCalculated := set.New()
	testPool.CalculateMemPoolAncestors(fromTxToEntry(tx7, 2000000, 1, 10.0, nil), setAncestorsCalculated,
		100, 1000000, 1000, 1000000, true)
	if !setAncestorsCalculated.IsEqual(setAncestors) {
		t.Errorf("setAncestorsCalculated.Size() : %d, setAncestors.Size() : %d, their should be equal"+
			"\n setAncestorsCalculated : %v,\n setAncestors : %v \n setAncestorsCalculated : %v, setAncestors : %v",
			setAncestorsCalculated.Size(), setAncestors.Size(), setAncestorsCalculated.List()[0], setAncestors.List()[0],
			setAncestorsCalculated.List(), setAncestors.List())
	}
	txentry7 := fromTxToEntry(tx7, 2000000, 1, 10.0, nil)
	testPool.AddUncheckedWithAncestors(&tx7.Hash, txentry7, setAncestors, true)
	if testPool.Size() != 7 {
		t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 7)
	}

	// Now tx6 should be sorted higher (high fee child): tx7, tx6, tx2, ...
	tmpSorted = make([]utils.Hash, 7)
	copy(tmpSorted, sortedOrder[1:])
	tmpSorted[5] = tx6.Hash
	tmpSorted[6] = tx7.Hash
	sortedOrder = tmpSorted
	fmt.Println("===============  02 ------------------")
	err = checkSort(testPool, sortedOrder, DESCENDANTSCORE)
	if err != nil {
		t.Error(err)
		fmt.Println("===============  03 ------------------")
		return
	}

	// low fee child of tx7
	tx8 := model.NewTx()
	tx8.Ins = make([]*model.TxIn, 1)
	tx8.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx7.Hash, 0), []byte{model.OP_11})
	tx8.Outs = make([]*model.TxOut, 1)
	tx8.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx8.Hash = tx8.TxHash()
	setAncestors.Add(testPool.MapTx.GetEntryByHash(tx7.Hash))
	fmt.Printf("tx8 Ancestors Number size : %v\n", setAncestors.Size())
	txentry8 := fromTxToEntry(tx8, 0, 2, 0, nil)
	//testPool.AddUncheckedWithAncestors(&tx8.Hash, txentry8, setAncestors, true)
	testPool.AddUnchecked(&tx8.Hash, txentry8, true)
	// Now tx8 should be sorted low, but tx6/tx both high
	tmpSorted = make([]utils.Hash, 8)
	tmpSorted[0] = tx8.Hash
	copy(tmpSorted[1:], sortedOrder)
	sortedOrder = tmpSorted
	fmt.Println("===============  04 ------------------")
	fmt.Printf("add child tx late txentry8.GetModifiedFee : %d, txentry8.SizeWithDescendants : %v, "+
		"txentry8.CountWithDescendants : %v,  txentry8.ModFeesWithDescendants() : %d, TxSize : %v, useDescendantScore() : %v;"+
		" txentry8 And txentry3 : %v, should be true\n",
		txentry8.GetModifiedFee(), txentry8.SizeWithDescendants, txentry8.CountWithDescendants,
		txentry8.ModFeesWithDescendants, txentry8.TxSize, useDescendantScore(txentry8),
		CompareTxMemPoolEntryByDescendantScore(txentry8, txentry3))
	err = checkSort(testPool, sortedOrder, DESCENDANTSCORE)
	err = nil
	if err != nil {
		t.Error(tx8.Hash.ToString())
		t.Error(err)
		fmt.Println("===============  05 ------------------")
		return
	}
	/*
		// low fee child of tx7
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
		tmpSorted = make([]utils.Hash, 9)
		tmpSorted[0] = tx9.Hash
		copy(tmpSorted[1:], sortedOrder)
		sortedOrder = tmpSorted
		err = checkSort(testPool, sortedOrder, DESCENDANTSCORE)
		if err != nil {
			t.Error(err)
			return
		}

		snapshotOrder := make([]utils.Hash, 9)
		copy(snapshotOrder, sortedOrder)
		setAncestors.Add(testPool.MapTx.GetEntryByHash(tx8.Hash))
		setAncestors.Add(testPool.MapTx.GetEntryByHash(tx9.Hash))

		// tx10 depends on tx8 and tx9 and has a high fee
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

		//*  tx8 and tx9 should both now be sorted higher
		//*  Final order after tx10 is added:
		//*
		//*  tx3 = 0 (1)
		//*  tx5 = 10000 (1)
		//*  tx1 = 10000 (1)
		//*  tx4 = 15000 (1)
		//*  tx2 = 20000 (1)
		//*  tx9 = 200k (2 txs)
		//*  tx8 = 200k (2 txs)
		//*  tx10 = 200k (1 tx)
		//*  tx6 = 2.2M (5 txs)
		//*  tx7 = 2.2M (4 txs)

		// take out tx9, tx8 from the beginning
		t.Log("sortedOrder.Size() : ", len(sortedOrder))
		tmpSorted = make([]utils.Hash, 10)
		tmpSorted[5] = tx9.Hash
		tmpSorted[6] = tx8.Hash
		copy(tmpSorted[:5], sortedOrder[2:7])
		copy(tmpSorted[8:], sortedOrder[7:])
		// tx10 is just before tx6
		tmpSorted[7] = tx10.Hash
		t.Log("tmpSorted.Size() : ", len(tmpSorted))
		err = checkSort(testPool, sortedOrder, DESCENDANTSCORE)
		if err != nil {
			t.Error(err)
			return
		}

		// there should be 10 transactions in the mempool
		if testPool.Size() != 10 {
			t.Errorf("current poolSize : %d, except the poolSize : %d", testPool.Size(), 10)
		}

		// Now try removing tx10 and verify the sort order returns to normal
		testPool.RemoveRecursive(testPool.MapTx.GetEntryByHash(tx10.Hash).TxRef, UNKNOWN)
		err = checkSort(testPool, snapshotOrder, DESCENDANTSCORE)
		if err != nil {
			t.Error(err)
			return
		}

		testPool.RemoveRecursive(testPool.MapTx.GetEntryByHash(tx9.Hash).TxRef, UNKNOWN)
		testPool.RemoveRecursive(testPool.MapTx.GetEntryByHash(tx8.Hash).TxRef, UNKNOWN)

		// * Now check the sort on the mining score index.
		// * Final order should be:
		// * tx7 (2M)
		// * tx2 (20k)
		// * tx4 (15000)
		// * tx1/tx5 (10000)
		// * tx3/6 (0)
		// * (Ties resolved by hash)
		// *
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
		err = checkSort(testPool, snapshotOrder, MININGSCORE)
		if err != nil {
			t.Error(err)
			return
		}
	*/
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

	sortedOrder := make([]utils.Hash, 5)
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
	err := checkSort(testPool, sortedOrder, ANCESTORSCORE)
	if err != nil {
		t.Error(err)
		return
	}

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
	tmpSorted := make([]utils.Hash, 6)
	copy(tmpSorted[:5], sortedOrder)
	sortedOrder = tmpSorted
	if tx3.Hash.ToBigInt().Cmp(tx6.Hash.ToBigInt()) < 0 {
		sortedOrder[5] = tx6.Hash
	} else {
		sortedOrder[4] = tx6.Hash
		sortedOrder[5] = tx3.Hash
	}
	err = checkSort(testPool, sortedOrder, ANCESTORSCORE)
	if err != nil {
		t.Error(err)
		return
	}

	tx7 := model.NewTx()
	tx7.Ins = make([]*model.TxIn, 1)
	tx7.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx6.Hash, 0), []byte{model.OP_11})
	tx7.Outs = make([]*model.TxOut, 1)
	tx7.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_11, model.OP_EQUAL})
	tx7.Hash = tx7.TxHash()
	tx7Size := tx7.SerializeSize()

	// set the fee to just below tx2's feerate when including ancestor
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
	err = checkSort(testPool, tmpSort, ANCESTORSCORE)
	if err != nil {
		t.Error(err)
		return
	}
	/*
		//after tx6 is mined, tx7 should move up in the sort
		vtx := algorithm.NewVector()
		vtx.PushBack(tx6)
		testPool.RemoveForBlock(vtx, 1)

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
		err = checkSort(testPool, tmpSort, ANCESTORSCORE)
		if err != nil {
			t.Error(err)
			return
		}
	*/
}

func TestMempoolEstimateFee(t *testing.T) {
	testPool := NewMemPool(utils.FeeRate{1000})
	tx1 := model.NewTx()
	tx1.Ins = make([]*model.TxIn, 1)
	tx1.Ins[0] = model.NewTxIn(model.NewOutPoint(&utils.HashOne, 0), []byte{model.OP_1})
	tx1.Outs = make([]*model.TxOut, 1)
	tx1.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_1, model.OP_EQUAL})
	tx1.Hash = tx1.TxHash()
	testPool.AddUnchecked(&tx1.Hash, fromTxToEntry(tx1, 10000, 0, 10.0, testPool), true)

	tx2 := model.NewTx()
	tx2.Ins = make([]*model.TxIn, 1)
	tx2.Ins[0] = model.NewTxIn(model.NewOutPoint(&utils.HashOne, 0), []byte{model.OP_2})
	tx2.Outs = make([]*model.TxOut, 1)
	tx2.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_2, model.OP_EQUAL})
	tx2.Hash = tx2.TxHash()
	testPool.AddUnchecked(&tx2.Hash, fromTxToEntry(tx2, 5000, 0, 10.0, testPool), true)
	fmt.Println("---------- 0 -------------")
	// should do nothing
	testPool.TrimToSize(testPool.DynamicMemoryUsage(), nil)
	if !testPool.Exists(tx1.Hash) {
		t.Errorf("tx1 should be In Mempool ...")
		return
	}
	if !testPool.Exists(tx2.Hash) {
		t.Errorf("tx2 should be In Mempool ...")
		return
	}
	fmt.Println("---------- 1 -------------")
	/*
		// should remove the lower-feerate transaction
		testPool.TrimToSize(testPool.DynamicMemoryUsage()*3/4, nil)
		if !testPool.Exists(tx1.Hash) {
			t.Errorf("tx1 should be In Mempool ...")
			return
		}
		if testPool.Exists(tx2.Hash) {
			t.Errorf("tx2 should be Not In Mempool ...")
			return
		}
		testPool.AddUnchecked(&tx2.Hash, fromTxToEntry(tx2, 5000, 0, 10.0, testPool), true)

		tx3 := model.NewTx()
		tx3.Ins = make([]*model.TxIn, 1)
		tx3.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx2.Hash, 0), []byte{model.OP_2})
		tx3.Outs = make([]*model.TxOut, 1)
		tx3.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_3, model.OP_EQUAL})
		tx3.Hash = tx3.TxHash()
		testPool.AddUnchecked(&tx3.Hash, fromTxToEntry(tx3, 20000, 0, 10.0, testPool), true)

		// tx3 should pay for tx2 (CPFP)
		testPool.TrimToSize(testPool.DynamicMemoryUsage()*3/4, nil)
		if testPool.Exists(tx1.Hash) {
			t.Errorf("tx1 should be Not In Mempool ...")
			return
		}
		if !testPool.Exists(tx2.Hash) {
			t.Errorf("tx2 should be In Mempool ...")
			return
		}
		if !testPool.Exists(tx3.Hash) {
			t.Errorf("tx3 should be In Mempool ...")
			return
		}

		// mempool is limited to tx1's size in memory usage, so nothing fits
		testPool.TrimToSize(int64(tx1.SerializeSize()), nil)
		if testPool.Exists(tx1.Hash) {
			t.Errorf("tx1 should Not be Not In Mempool ...")
			return
		}
		if testPool.Exists(tx2.Hash) {
			t.Errorf("tx2 should Not be In Mempool ...")
			return
		}
		if testPool.Exists(tx3.Hash) {
			t.Errorf("tx3 should Not be In Mempool ...")
			return
		}

		maxFeeRateRemoved := utils.NewFeeRateWithSize(25000, tx3.SerializeSize()+tx2.SerializeSize())
		if testPool.GetMinFee(1).GetFeePerK() != maxFeeRateRemoved.GetFeePerK()+1000 {
			t.Errorf("current FeePerk : %d, except FeePerk : %d",
				testPool.GetMinFee(1).GetFeePerK(), maxFeeRateRemoved.GetFeePerK()+1000)
		}

		tx4 := model.NewTx()
		tx4.Ins = make([]*model.TxIn, 2)
		tx4.Ins[0] = model.NewTxIn(model.NewOutPoint(&utils.HashZero, math.MaxUint32), []byte{model.OP_4})
		tx4.Ins[1] = model.NewTxIn(model.NewOutPoint(&utils.HashZero, math.MaxUint32), []byte{model.OP_4})
		tx4.Outs = make([]*model.TxOut, 2)
		tx4.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_4, model.OP_EQUAL})
		tx4.Outs[1] = model.NewTxOut(10*utils.COIN, []byte{model.OP_4, model.OP_EQUAL})
		tx4.Hash = tx4.TxHash()

		tx5 := model.NewTx()
		tx5.Ins = make([]*model.TxIn, 2)
		tx5.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx4.Hash, 0), []byte{model.OP_4})
		tx5.Ins[1] = model.NewTxIn(model.NewOutPoint(&utils.HashZero, math.MaxUint32), []byte{model.OP_5})
		tx5.Outs = make([]*model.TxOut, 2)
		tx5.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_5, model.OP_EQUAL})
		tx5.Outs[1] = model.NewTxOut(10*utils.COIN, []byte{model.OP_5, model.OP_EQUAL})
		tx5.Hash = tx5.TxHash()

		tx6 := model.NewTx()
		tx6.Ins = make([]*model.TxIn, 2)
		tx6.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx4.Hash, 1), []byte{model.OP_4})
		tx6.Ins[1] = model.NewTxIn(model.NewOutPoint(&utils.HashZero, math.MaxUint32), []byte{model.OP_6})
		tx6.Outs = make([]*model.TxOut, 2)
		tx6.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_6, model.OP_EQUAL})
		tx6.Outs[1] = model.NewTxOut(10*utils.COIN, []byte{model.OP_6, model.OP_EQUAL})
		tx6.Hash = tx6.TxHash()

		tx7 := model.NewTx()
		tx7.Ins = make([]*model.TxIn, 2)
		tx7.Ins[0] = model.NewTxIn(model.NewOutPoint(&tx5.Hash, 0), []byte{model.OP_5})
		tx7.Ins[1] = model.NewTxIn(model.NewOutPoint(&tx6.Hash, 0), []byte{model.OP_6})
		tx7.Outs = make([]*model.TxOut, 2)
		tx7.Outs[0] = model.NewTxOut(10*utils.COIN, []byte{model.OP_7, model.OP_EQUAL})
		tx7.Outs[1] = model.NewTxOut(10*utils.COIN, []byte{model.OP_7, model.OP_EQUAL})
		tx7.Hash = tx7.TxHash()

		testPool.AddUnchecked(&tx4.Hash, fromTxToEntry(tx4, 7000, 0, 10.0, testPool), true)
		testPool.AddUnchecked(&tx5.Hash, fromTxToEntry(tx5, 1000, 0, 10.0, testPool), true)
		testPool.AddUnchecked(&tx6.Hash, fromTxToEntry(tx6, 1100, 0, 10.0, testPool), true)
		testPool.AddUnchecked(&tx7.Hash, fromTxToEntry(tx7, 9000, 0, 10.0, testPool), true)

		// we only require this remove, at max, 2 txn, because its not clear what
		// we're really optimizing for aside from that
		testPool.TrimToSize(testPool.DynamicMemoryUsage()-1, nil)
		if !testPool.Exists(tx4.Hash) {
			t.Errorf("tx4 should  be In Mempool ...")
		}
		if !testPool.Exists(tx6.Hash) {
			t.Errorf("tx6 should  be In Mempool ...")
		}
		if testPool.Exists(tx7.Hash) {
			t.Errorf("tx7 should  Not be In Mempool ...")
		}

		if testPool.Exists(tx5.Hash) {
			testPool.AddUnchecked(&tx5.Hash, fromTxToEntry(tx5, 1000, 0, 10.0, testPool), true)
		}
		testPool.AddUnchecked(&tx7.Hash, fromTxToEntry(tx7, 9000, 0, 10.0, testPool), true)

		// should maximize mempool size by only removing 5/7
		testPool.TrimToSize(testPool.DynamicMemoryUsage()/2, nil)
		if !testPool.Exists(tx4.Hash) {
			t.Errorf("tx4 should  be In Mempool ...")
		}
		if testPool.Exists(tx5.Hash) {
			t.Errorf("tx5 should Not be In Mempool ...")
		}
		if !testPool.Exists(tx6.Hash) {
			t.Errorf("tx6 should be In Mempool ...")
		}
		if testPool.Exists(tx7.Hash) {
			t.Errorf("tx7 should  Not be In Mempool ...")
		}

		testPool.AddUnchecked(&tx5.Hash, fromTxToEntry(tx5, 1000, 0, 10.0, testPool), true)
		testPool.AddUnchecked(&tx7.Hash, fromTxToEntry(tx7, 9000, 0, 10.0, testPool), true)

		vtx := algorithm.NewVector()
		utils.SetMockTime(42)
		utils.SetMockTime(42 + ROLLING_FEE_HALFLIFE)
		if testPool.GetMinFee(1).GetFeePerK() != maxFeeRateRemoved.GetFeePerK()+1000 {
			t.Errorf("current FeePerk : %d, except FeePerk : %d",
				testPool.GetMinFee(1).GetFeePerK(), maxFeeRateRemoved.GetFeePerK()+1000)
		}

		// ... we should keep the same min fee until we get a block
		testPool.RemoveForBlock(vtx, 1)
		utils.SetMockTime(42 + 2*ROLLING_FEE_HALFLIFE)
		if testPool.GetMinFee(1).GetFeePerK() != (maxFeeRateRemoved.GetFeePerK()+1000)/2 {
			t.Errorf("current FeePerk : %d, except FeePerk : %d",
				testPool.GetMinFee(1).GetFeePerK(), (maxFeeRateRemoved.GetFeePerK()+1000)/2)
		}
		// ... then feerate should drop 1/2 each halflife

		utils.SetMockTime(42 + 2*ROLLING_FEE_HALFLIFE + ROLLING_FEE_HALFLIFE/2)
		if testPool.GetMinFee(testPool.DynamicMemoryUsage()*5/2).GetFeePerK() !=
			(maxFeeRateRemoved.GetFeePerK()+1000)/4 {
			t.Errorf("current FeePerk : %d, except FeePerk : %d",
				testPool.GetMinFee(testPool.DynamicMemoryUsage()*5/2).GetFeePerK(),
				(maxFeeRateRemoved.GetFeePerK()+1000)/4)
		}
		// ... with a 1/2 halflife when mempool is < 1/2 its target size

		utils.SetMockTime(42 + 2*ROLLING_FEE_HALFLIFE + ROLLING_FEE_HALFLIFE/2 + ROLLING_FEE_HALFLIFE/4)
		if testPool.GetMinFee(testPool.DynamicMemoryUsage()*9/2).GetFeePerK() !=
			(maxFeeRateRemoved.GetFeePerK()+1000)/8 {
			t.Errorf("current FeePerk : %d, except FeePerk : %d",
				testPool.GetMinFee(testPool.DynamicMemoryUsage()*9/2).GetFeePerK(), (maxFeeRateRemoved.GetFeePerK()+1000)/8)
		}
		// ... with a 1/4 halflife when mempool is < 1/4 its target size

		utils.SetMockTime(42 + 7*ROLLING_FEE_HALFLIFE + ROLLING_FEE_HALFLIFE/2 + ROLLING_FEE_HALFLIFE/4)
		if testPool.GetMinFee(1).GetFeePerK() != 1000 {
			t.Errorf("current FeePerk : %d, except FeePerk : %d",
				testPool.GetMinFee(1).GetFeePerK(), 1000)
		}
		// ... but feerate should never drop below 1000

		utils.SetMockTime(42 + 8*ROLLING_FEE_HALFLIFE + ROLLING_FEE_HALFLIFE/2 + ROLLING_FEE_HALFLIFE/4)
		if testPool.GetMinFee(1).GetFeePerK() != 0 {
			t.Errorf("current FeePerk : %d, except FeePerk : %d",
				testPool.GetMinFee(1).GetFeePerK(), 0)
		}
		// ... unless it has gone all the way to 0 (after getting past 1000/2)

		utils.SetMockTime(0)
	*/
}
