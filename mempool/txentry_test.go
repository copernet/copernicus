package mempool

import (
	"github.com/google/btree"
	"testing"
)

func TestGetTxFromMemPool(t *testing.T) {
	t1 := EntryFeeRateSort{}
	t1.txFee = 990
	t1.sigOpCount = 927

	bt1 := btree.New(32)
	bt2 := btree.New(32)
	bt1.ReplaceOrInsert(t1)
	bt2.ReplaceOrInsert(EntryFeeRateSort(t1))

	a := bt1.Min()
	if b := bt2.Get(a); b == nil {
		t.Errorf("error")
	}

}
