package mempool

import (
	"github.com/btcboost/copernicus/core"
	"github.com/google/btree"
	"testing"
)

func TestGetTxFromMemPool(t *testing.T) {
	t1 := EntryAncestorFeeRateSort{}
	t1.TxFee = 990
	t1.SigOpCount = 927
	t1.SumFeeWithAncestors = 182
	t1.SumSizeWitAncestors = 93
	tx := core.NewTx()
	t1.Tx = tx
	bt1 := btree.New(32)
	bt2 := btree.New(32)
	bt1.ReplaceOrInsert(t1)
	bt2.ReplaceOrInsert(t1)

	a := bt1.Min()
	if b := bt2.Get(a); b == nil {
		t.Errorf("error")
	}

}
