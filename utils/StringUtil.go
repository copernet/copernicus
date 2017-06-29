package utils

import (
	"crypto/sha256"
	"fmt"
)

// https://en.bitcoin.it/wiki/Protocol_specification#Variable_length_integer

func DecodeVariableLengthInteger(raw []byte) (cnt int, cntSize int) {

	if raw[0] < 0xfd {
		return int(raw[0]), 1
	}
	cntSize = 1 + (2 << (2 - (0xff - raw[0])))
	if len(raw) < 1+cntSize {
		return
	}
	res := uint64(0)
	for i := 1; i < cntSize; i++ {
		res |= (uint64((raw[i]))) << uint64(8*(i-1))

	}
	cnt = int(res)
	return

}

func ToHash256String(data []byte) string {

	sha := sha256.New()
	sha.Write(data[:])
	tmp := sha.Sum(nil)
	sha.Reset()
	sha.Write(tmp)
	has := sha.Sum(nil)
	return ToHexString(has)
}

func ToHexString(data []byte) string {
	var str string
	for i := 0; i < 32; i++ {
		str = fmt.Sprintf("%02x", data[31-i])
	}
	return str

}
