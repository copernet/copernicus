package mempool

import (
	"testing"

	"gopkg.in/fatih/set.v0"
)

func TestDynamicUsage(t *testing.T) {
	entry := TxMempoolEntry{}
	size := DynamicUsage(entry)
	if size != 168 {
		t.Errorf("current TxMempoolEntry type Size : %d except Size : 168", size)
	}
	entryPtr := &TxMempoolEntry{}
	size = DynamicUsage(entryPtr)
	if size != 8 {
		t.Errorf("current *TxMempoolEntry type Size : %d except Size : 8", size)
	}

	setInt := set.New()
	setInt.Add(1)
	setInt.Add(2)
	setInt.Add(3)
	size = DynamicUsage(setInt)
	if size != 24 {
		t.Errorf("current IntSet type Size : %d except Size : 24", size)
	}

	setEntry := set.New(TxMempoolEntry{Fee: 10})
	setEntry.Add(TxMempoolEntry{FeeDelta: 100})
	setEntry.Add(TxMempoolEntry{Fee: 90})
	setEntry.Add(TxMempoolEntry{Fee: 98})
	size = DynamicUsage(setEntry)
	if size != 168*4 {
		t.Errorf("current  TxMempoolEntrySet type Size : %d except Size : %d, element Number : %d",
			size, 168*4, setEntry.Size())
	}

	sizeTwo := MallocUsage(10)
	if sizeTwo != 32 {
		t.Errorf("current Size : %d, except Size : %d ", sizeTwo, 32)
	}
}
