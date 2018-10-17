package util

import (
	"bytes"
	"github.com/davecgh/go-spew/spew"
	"io"
	"testing"
)

func TestVarInt(t *testing.T) {
	tests := []struct {
		in  uint64
		out uint64
		buf []byte
	}{
		{0, 0, []byte{0x00}},
		// Max single byte
		{0xfc, 0xfc, []byte{0xfc}},
		// Min 2-byte
		{0xfd, 0xfd, []byte{0xfd, 0x0fd, 0x00}},
		// Max 2-byte
		{0xffff, 0xffff, []byte{0xfd, 0xff, 0xff}},
		// Min 4-byte
		{0x10000, 0x10000, []byte{0xfe, 0x00, 0x00, 0x01, 0x00}},
		// Max 4-byte
		{0xffffffff, 0xffffffff, []byte{0xfe, 0xff, 0xff, 0xff, 0xff}},
		// Min 8-byte
		{
			0x100000000, 0x100000000,
			[]byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		// Max 8-byte
		{
			0xffffffffffffffff, 0xffffffffffffffff,
			[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		var buf bytes.Buffer
		err := WriteVarInt(&buf, test.in)
		if err != nil {
			t.Errorf("WriteVarInt #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("WriteVarInt #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode from wire format.
		rbuf := bytes.NewReader(test.buf)
		val, err := ReadVarInt(rbuf)
		if err != nil {
			t.Errorf("ReadVarInt #%d error %v", i, err)
			continue
		}
		if val != test.out {
			t.Errorf("ReadVarInt #%d\n got: %d want: %d", i,
				val, test.out)
			continue
		}
	}
}

func TestVarIntWireErrors(t *testing.T) {
	tests := []struct {
		in       uint64
		buf      []byte
		max      int
		writeErr error
		readErr  error
	}{
		{0, []byte{0x00}, 0, nil, io.EOF},
		{0xfd, []byte{0xfd}, 2, nil, io.EOF},
		{0x10000, []byte{0xfe}, 2, nil, io.EOF},
		{0x100000000, []byte{0xff}, 2, nil, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		var w bytes.Buffer
		err := WriteVarInt(&w, test.in)
		if err != test.writeErr {
			t.Errorf("WriteVarInt #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		var r bytes.Reader
		_, err = ReadVarInt(&r)
		if err != test.readErr {
			t.Errorf("ReadVarInt #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}

func TestVarIntSerializeSize(t *testing.T) {
	tests := []struct {
		val  uint64
		size uint32
	}{
		// Single byte
		{0, 1},
		// Max single byte
		{0xfc, 1},
		// Min 2-byte
		{0xfd, 3},
		// Max 2-byte
		{0xffff, 3},
		// Min 4-byte
		{0x10000, 5},
		// Max 4-byte
		{0xffffffff, 5},
		// Min 8-byte
		{0x100000000, 9},
		// Max 8-byte
		{0xffffffffffffffff, 9},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		serializedSize := VarIntSerializeSize(test.val)
		if serializedSize != test.size {
			t.Errorf("VarIntSerializeSize #%d got: %d, want: %d", i,
				serializedSize, test.size)
			continue
		}
	}
}
