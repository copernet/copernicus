package util

import (
	"bytes"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"io"
	"testing"
)

func TestWriteVarBytes(t *testing.T) {
	bytes256 := bytes.Repeat([]byte{0x01}, 256)

	tests := []struct {
		in  []byte
		buf []byte
	}{
		// Empty byte array
		{[]byte{}, []byte{0x00}},
		// Single byte varint + byte array
		{[]byte{0x01}, []byte{0x01, 0x01}},
		// 2-byte varint + byte array
		{bytes256, append([]byte{0xfd, 0x00, 0x01}, bytes256...)},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		var buf bytes.Buffer
		err := WriteVarBytes(&buf, test.in)
		if err != nil {
			t.Errorf("WriteVarBytes #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("WriteVarBytes #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		rbuf := bytes.NewReader(test.buf)
		val, err := ReadVarBytes(rbuf, (1),
			"test payload")
		if err != nil {
			str := fmt.Sprintf("%s is larger that the max allowed size count %d,max %d", "test payload", 256, 1)
			if err.Error() == errors.New(str).Error() {
				break
			} else {
				t.Errorf("ReadVarBytes #%d error %v", i, err)
				continue
			}
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("ReadVarBytes #%d\n got: %s want: %s", i,
				val, test.buf)
			continue
		}
	}
}

func TestVarBytesError(t *testing.T) {

	// bytes256 is a byte array that takes a 2-byte varint to encode.
	bytes256 := bytes.Repeat([]byte{0x01}, 256)

	tests := []struct {
		in       []byte
		buf      []byte
		max      int
		writeErr error
		readErr  error
	}{
		// Latest protocol version with intentional read/write errors.
		// Force errors on empty byte array.
		{[]byte{}, []byte{0x00}, 0, io.ErrShortWrite, io.EOF},
		// Force error on single byte varint + byte array.
		{[]byte{0x01, 0x02, 0x03}, []byte{0x04}, 2, io.ErrShortWrite, io.EOF},
		// Force errors on 2-byte varint + byte array.
		{bytes256, []byte{0xfd}, 2, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		var r bytes.Reader
		_, err := ReadVarBytes(&r, (1),
			"test payload")
		if err != test.readErr {
			t.Errorf("ReadVarBytes #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}
	}
}
