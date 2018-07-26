package blockindex

import (
	"testing"
	"bytes"
	"math/rand"
)

func TestSerialize(t *testing.T) {
	var bIndex1, bIndex2 BlockIndex
	buf := bytes.NewBuffer(nil);
	r := rand.New(rand.NewSource(0));
	for i := 1; i <= 100; i++ {
		bIndex1.Height = r.Int31()
		bIndex1.Status = r.Uint32()
		bIndex1.TxCount = r.Int31()
		bIndex1.File = r.Int31()
		bIndex1.DataPos = r.Uint32()
		bIndex1.UndoPos = r.Uint32()
		err := bIndex1.Serialize(buf)
		if err != nil {
			t.Error(err)
		}
		err = bIndex2.Unserialize(buf)
		if err != nil {
			t.Error(err)
		}
		if bIndex1.Height != bIndex2.Height || bIndex1.Status != bIndex2.Status ||
			bIndex1.TxCount != bIndex2.TxCount || bIndex1.File != bIndex2.File ||
			bIndex1.DataPos != bIndex2.DataPos || bIndex1.UndoPos != bIndex2.UndoPos {
			t.Errorf("Unserialize after Serialize returns differently")
			return
		}
	}
}
