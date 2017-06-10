package utils

import (
	"io"
	"fmt"
	"github.com/pkg/errors"
)

func ReadVarString(r io.Reader, size uint32) (string, error) {
	count, err := ReadVarInt(r, size)
	if err != nil {
		return "", err
	}
	if count > MAX_SIZE {
		str := fmt.Sprintf("variable length sring is too long count %d ,max %d", count, MAX_SIZE)
		return "", errors.New(str)

	}
	buf := make([]byte, count)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return "", err

	}
	return string(buf), nil

}
func WriteVarString(w io.Writer, size uint32, str string) error {
	err := WriteVarInt(w, size, uint64(len(str)))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(str))
	if err != nil {
		return err
	}
	return nil
}
