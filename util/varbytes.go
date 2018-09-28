package util

import (
	"errors"
	"fmt"
	"io"
)

func ReadVarBytes(r io.Reader, maxAllowed uint64, fieldName string) ([]byte, error) {
	count, err := ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	if count > maxAllowed {
		str := fmt.Sprintf("%s is larger that the max allowed size count %d,max %d", fieldName, count, maxAllowed)
		return nil, errors.New(str)
	}
	b := make([]byte, count)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func WriteVarBytes(w io.Writer, bytes []byte) error {
	slen := uint64(len(bytes))
	err := WriteVarInt(w, slen)
	if err != nil {
		return err
	}
	_, err = w.Write(bytes)
	return err
}
