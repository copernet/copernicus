package utils

import (
	"io"
	"fmt"
	"github.com/pkg/errors"
)

func ReadVarBytes(r io.Reader, size uint32, maxAllowed uint32, fieldName string) ([]byte, error) {
	count, err := ReadVarInt(r, size)
	if err != nil {
		return nil, err
	}

	if count > uint64(maxAllowed) {
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

func WriteVarBytes(w io.Writer, size uint32, bytes [] byte) error {
	slen := uint64(len(bytes))
	err := WriteVarInt(w, size, slen)
	if err != nil {
		return err
	}
	_, err = w.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}
