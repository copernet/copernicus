package util

import (
	"bytes"
	"testing"
	"github.com/davecgh/go-spew/spew"
	"strings"
	"fmt"
	"errors"
)

func TestVarString(t *testing.T) {
	// str256 is a string that takes a 2-byte varint to encode.
	str256 := strings.Repeat("test", 64)

	tests := []struct {
		in  string // String to encode
		out string // String to decoded value
		buf []byte // Wire encoding
	}{
		// Latest protocol version.
		// Empty string
		{"", "", []byte{0x00},},
		// Single byte varint + string
		{"Test", "Test", append([]byte{0x04}, []byte("Test")...),},
		// 2-byte varint + string
		{str256, str256, append([]byte{0xfd, 0x00, 0x01}, []byte(str256)...),},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		var buf bytes.Buffer
		err := WriteVarString(&buf, test.in)
		if err != nil {
			t.Errorf("WriteVarString #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("WriteVarString #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode from wire format.
		rbuf := bytes.NewReader(test.buf)
		val, err := ReadVarString(rbuf)
		if err != nil {
			t.Errorf("ReadVarString #%d error %v", i, err)
			continue
		}
		if val != test.out {
			t.Errorf("ReadVarString #%d\n got: %s want: %s", i,
				val, test.out)
			continue
		}
	}
}

// TestVarStringWireErrors performs negative tests against wire encode and
// decode of variable length strings to confirm error paths work correctly.
func TestVarStringWireErrors(t *testing.T) {
	// str256 is a string that takes a 2-byte varint to encode.
	str256 := strings.Repeat("test", 33554433)

	tests := []struct {
		in       string // Value to encode
		buf      []byte // Wire encoding
		max      int    // Max size of fixed buffer to induce errors
		writeErr error  // Expected write error
		readErr  error  // Expected read error
	}{
		// Latest protocol version with intentional read/write errors.
		// Force errors on empty string.
		{"", []byte{0x00}, 0, nil, nil},
		// Force error on single byte varint + string.
		{"Test", []byte{0x04}, 2, nil, nil},
		// Force errors on 2-byte varint + string.
		{str256, []byte{0xfd}, 2, nil, nil},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		buf := bytes.NewBuffer(nil)
		err := WriteVarString(buf, test.in)
		if err != test.writeErr {
			t.Errorf("WriteVarString #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// Decode from wire format.
		_, err = ReadVarString(buf)
		if err != test.readErr {
			str := fmt.Sprintf("variable length sring is too long count %d ,max %d", 134217732, MaxSize)
			if err.Error() == errors.New(str).Error() {
				continue
			} else {
				t.Errorf("ReadVarString #%d wrong error got: %v, want: %v",
					i, err, test.readErr)
				continue
			}
		}
	}
}
