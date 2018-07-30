package block

import (
	"bytes"
	"testing"
	"math/rand"
	"time"
	"reflect"
)

func TestDiskBlockPosSerialize(t *testing.T) {
	var dbp1, dbp2 DiskBlockPos
	buf := bytes.NewBuffer(nil)
	r := rand.New(rand.NewSource(time.Now().Unix()));
	for i := 0; i < 100; i++ {
		dbp1.File = r.Int31()
		dbp2.Pos = r.Uint32()
		buf.Reset()
		dbp1.Serialize(buf)
		dbp2.Unserialize(buf)
		if !dbp1.Equal(&dbp2) {
			t.Errorf("Unserialize after Serialize returns differently, dbp1=%#v, dbp2=%#v",
				dbp1, dbp2)
			return
		}
	}
}

func TestSetNull(t *testing.T) {
	var dbp DiskBlockPos
	dbp.SetNull()
	if !dbp.IsNull() {
		t.Errorf("DiskBlockPos SetNull is wrong")
	}
}

func TestDiskTxPosSerialize(t *testing.T) {
	var dtp2 DiskTxPos
	buf := bytes.NewBuffer(nil)
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 100; i++ {
		file := r.Int31()
		pos := r.Uint32()
		offsetIn := r.Uint32()
		dbp := NewDiskBlockPos(file, pos)
		dtp1 := NewDiskTxPos(dbp, offsetIn)
		if dbp.File != file || dbp.Pos != pos {
			t.Errorf("NewDiskBlockPos is wrong")
			return
		}
		if dtp1.TxOffsetIn != offsetIn {
			t.Errorf("NewDiskTxPos is wrong")
			return
		}
		buf.Reset()
		dtp1.Serialize(buf)
		dtp2.Unserialize(buf)
		if !reflect.DeepEqual(*dtp1, dtp2) {
			t.Errorf("Unserialize after Serialize returns differently, dtp1=%#v, dtp2=%#v",
				dtp1, dtp2)
			return
		}
	}
}