package mempool

import (
	"bytes"
	"testing"

	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

func fromTx(tx *model.Tx, pool *Mempool) *TxMempoolEntry {
	var inChainValue btcutil.Amount
	if pool != nil && pool.HasNoInputsOf(tx) {
		inChainValue = btcutil.Amount(tx.GetValueOut())
	}
	entry := NewTxMempoolEntry(tx, 0, 0, 0, 1, inChainValue, false, 4, nil)
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
	if !testPool.AddUnchecked(&txParentPtr.Hash, fromTx(txParentPtr, nil), true) {
		t.Error("add Tx failure ...")
	}
	poolSize = testPool.Size()
	testPool.RemoveRecursive(txParentPtr, UNKNOWN)
	if testPool.Size() != poolSize-1 {
		t.Errorf("current poolSize : %d, except the poolSize : %d\n",
			testPool.Size(), poolSize-1)
	}

	// Parent, children, grandchildren:
	testPool.AddUnchecked(&txParentPtr.Hash, fromTx(txParentPtr, nil), true)
	for i := 0; i < 3; i++ {
		testPool.AddUnchecked(&txChild[i].Hash, fromTx(&txChild[i], nil), true)
		testPool.AddUnchecked(&txGrandChild[i].Hash, fromTx(&txGrandChild[i], nil), true)
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
		testPool.AddUnchecked(&txChild[i].Hash, fromTx(&txChild[i], nil), true)
		testPool.AddUnchecked(&txGrandChild[i].Hash, fromTx(&txGrandChild[i], nil), true)
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
	testPool.AddUnchecked(&txParentPtr.Hash, fromTx(txParentPtr, nil), true)
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

func TestMempoolEstimatePriority(t *testing.T) {

}
