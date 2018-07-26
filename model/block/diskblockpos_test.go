package block

import (
	"bytes"
	"testing"
	"math/rand"
)

func TestDiskBlockPos_Serialize(t *testing.T) {
	var dbp1, dbp2 DiskBlockPos
	buf := bytes.NewBuffer(nil)
	r := rand.New(rand.NewSource(0));
	for i := 1; i <= 100; i++ {
		dbp1.File = r.Int31()
		dbp2.Pos = r.Uint32()
		buf.Reset()
		dbp1.Serialize(buf)
		dbp2.Unserialize(buf)
		if !dbp1.Equal(&dbp2) {
			t.Errorf("Unserialize after Serialize returns differently")
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

func TestDiskTxPos_Serialize(t *testing.T) {
	var dtp2 DiskTxPos
	buf := bytes.NewBuffer(nil)
	r := rand.New(rand.NewSource(0));
	for i := 1; i <= 100; i++ {
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
		dtp1.TxOffsetIn = r.Uint32()
		buf.Reset()
		dtp1.Serialize(buf)
		dtp2.Unserialize(buf)
		if dtp1.TxOffsetIn != dtp2.TxOffsetIn {
			t.Errorf("Unserialize after Serialize returns differently")
			return
		}
	}
}