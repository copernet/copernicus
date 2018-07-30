package blockindex

import (
	"testing"
	"bytes"
	"math/rand"
	"time"
	"reflect"
)

func TestSerialize(t *testing.T) {
	var bIndex1, bIndex2 BlockIndex
	buf := bytes.NewBuffer(nil);
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 100; i++ {
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
		if !reflect.DeepEqual(bIndex1, bIndex2) {
			t.Errorf("Unserialize after Serialize returns differently bIndex1=%#v, bIndex2=%#v",
				bIndex1, bIndex2)
			return
		}
	}
}