package utils

import (
	"io"
	"encoding/binary"
	"github.com/pkg/errors"
	"fmt"
)

var errVarIntDesc = "non-rule varint %x - discriminant %x must encode a value greater than %x "

func ReadVarInt(r io.Reader, size uint32) (uint64, error) {
	discriminant, err := binarySerializer.Uint8(r)
	if err != nil {
		return 0, err
	}
	var result uint64
	switch discriminant {
	case 0xff:
		sv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return 0, err
		}
		result = sv
		min := uint64(0x100000000)
		if result < min {
			return 0, errors.New(fmt.Sprintf(errVarIntDesc, result, discriminant, min))
		}

	case 0xfe:
		sv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return 0, err
		}
		result = uint64(sv)
		min := uint64(0x10000)
		if result < min {
			return 0, errors.New(fmt.Sprintf(errVarIntDesc, result, discriminant, min))
		}
	case 0xfd:
		sv, err := binarySerializer.Uint16(r, binary.LittleEndian)
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
