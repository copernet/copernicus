package utils

import (
	"bytes"
	"io"
)

func WriteVarLenInt(w io.Writer, n uint64) error {
	buf := make([]byte, 10)
	len := 0
	mask := uint64(0)
	for {
		if len > 0 {
			mask = 0x80
		}
		buf[len] = byte((n & 0x7f) | mask)
		if n <= 0x7f {
			break
		}
		n = (n >> 7) - 1
		len++
	}
	var tmp bytes.Buffer
	for i := len; i >= 0; i-- {
		tmp.WriteByte(buf[i])
	}
	_, err := w.Write(tmp.Bytes())
	return err
}

func ReadVarLenInt(r io.Reader) (uint64, error) {
	ret := uint64(0)
	buf := make([]byte, 1)
	for {
		len, err := r.Read(buf)
		if len == 0 {
			return ret, err
		}
		ret = (ret << 7) | uint64(buf[0]&0x7f)
		if buf[0]&0x80 != 0 {
			ret++
		} else {
			return ret, nil
		}
	}
}

func VarLenIntSize(n uint64) uint {
	size := uint(0)
	for {
		size++
		if n <= 0x7f {
			break
		}
		n = (n >> 7) - 1
	}
	return size
}
