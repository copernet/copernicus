package utils

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

func ReadVarString(r io.Reader) (string, error) {
	count, err := ReadVarInt(r)
	if err != nil {
		return "", err
	}
	if count > MaxSize {
		str := fmt.Sprintf("variable length sring is too long count %d ,max %d", count, MaxSize)
		return "", errors.New(str)

	}
	buf := make([]byte, count)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return "", err

	}
	return string(buf), nil

}
func WriteVarString(w io.Writer, str string) error {
	err := WriteVarInt(w, uint64(len(str)))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(str))
	return err
}
