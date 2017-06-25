package utils

import (
	"io"
	"encoding/binary"
	"github.com/pkg/errors"
	"fmt"
	"math"
)

var errVarIntDesc = "non-rule varint %x - discriminant %x must encode a value greater than %x "

func ReadVarInt(r io.Reader, size uint32) (uint64, error) {
	discriminant, err := BinarySerializer.Uint8(r)
	if err != nil {
		return 0, err
	}
	var result uint64
	switch discriminant {
	case 0xff:
		sv, err := BinarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return 0, err
		}
		result = sv
		min := uint64(0x100000000)
		if result < min {
			return 0, errors.New(fmt.Sprintf(errVarIntDesc, result, discriminant, min))
		}

	case 0xfe:
		sv, err := BinarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return 0, err
		}
		result = uint64(sv)
		min := uint64(0x10000)
		if result < min {
			return 0, errors.New(fmt.Sprintf(errVarIntDesc, result, discriminant, min))
		}
	case 0xfd:
		sv, err := BinarySerializer.Uint16(r, binary.LittleEndian)
		if err != nil {
			return 0, err
		}
		result = uint64(sv)
		min := uint64(0xfd)
		if result < min {
			return 0, errors.New(fmt.Sprintf(errVarIntDesc, result, discriminant, min))
		}
	default:
		result = uint64(discriminant)

	}
	return result, nil
}

func WriteVarInt(w io.Writer, size uint32, val uint64) error {
	if val < 0xfd {
		return BinarySerializer.PutUint8(w, uint8(val))
	}
	if val < math.MaxUint16 {
		err := BinarySerializer.PutUint8(w, 0xfd)
		if err != nil {
			return err
		}
		return BinarySerializer.PutUint16(w, binary.LittleEndian, uint16(val))

	}
	if val <= math.MaxUint32 {
		err := BinarySerializer.PutUint8(w, 0xfe)
		if err != nil {
			return err
		}
		return BinarySerializer.PutUint32(w, binary.LittleEndian, uint32(val))
	}
	err := BinarySerializer.PutUint8(w, 0xff)
	if err != nil {
		return err
	}
	return BinarySerializer.PutUint64(w, binary.LittleEndian, val)

}

func VarIntSerializeSize(val uint64) int {
	if val < 0xfd {
		return 1
	}
	if val <= math.MaxUint16 {
		return 3
	}
	if val <= math.MaxUint32 {
		return 5
	}
	return 9

}
