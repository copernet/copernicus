package utils

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestVarLenInt(t *testing.T) {
	bs := bytes.NewBuffer(nil)
	size := uint(0)

	for i := 0; i < 100000; i++ {
		if err := WriteVarLenInt(bs, uint64(i)); err != nil {
			t.Errorf("WriteVarLenInt : %v", err)
		}
		size += VarLenIntSize(uint64(i))
		if int(size) != bs.Len() {
			t.Errorf("size not match")
		}
	}
	for i := uint64(0); i < 100000000000; i += 999999937 {
		if err := WriteVarLenInt(bs, i); err != nil {
			t.Errorf("WriteVarLenInt : %v", err)
		}
		size += VarLenIntSize(uint64(i))
		if int(size) != bs.Len() {
			t.Errorf("size not match")
		}
	}

	for i := 0; i < 100000; i++ {
		j, _ := ReadVarLenInt(bs)
		if uint64(i) != j {
			t.Errorf("got %d, expect %d", j, i)
		}
	}
	for i := uint64(0); i < 100000000000; i += 999999937 {
		j, _ := ReadVarLenInt(bs)
		if uint64(i) != j {
			t.Errorf("got %d, expect %d", j, i)
		}
	}
}

func testF(t *testing.T, bs *bytes.Buffer, n uint64, expect string) {
	tmp := make([]byte, 10)

	if err := WriteVarLenInt(bs, n); err != nil {
		t.Errorf("WriteVarLenInt : %v", err)
	}
	len, err := bs.Read(tmp)
	if err != nil {
		t.Errorf("failed read from buffer: %v", err)
	}
	if hex.EncodeToString(tmp[:len]) != expect {
		t.Errorf("expect %s", expect)
	}
}

func TestVarLenIntBitPattern(t *testing.T) {
	bs := bytes.NewBuffer(nil)
	testF(t, bs, 0, "00")
	testF(t, bs, 0x7f, "7f")
	testF(t, bs, 0x80, "8000")
	testF(t, bs, 0x1234, "a334")
	testF(t, bs, 0xffff, "82fe7f")
	testF(t, bs, 0x123456, "c7e756")
	testF(t, bs, 0x80123456, "86ffc7e756")
	testF(t, bs, 0xffffffff, "8efefefe7f")
	testF(t, bs, 0x7fffffffffffffff, "fefefefefefefefe7f")
	testF(t, bs, 0xffffffffffffffff, "80fefefefefefefefe7f")
}
