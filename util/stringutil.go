package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

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

func HexToBytes(str string) []byte {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return nil
	}
	return bytes
}

func SplitHex(str string, split string) (string, error) {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return "", err
	}
	result := ""
	for i := 0; i < len(bytes); i++ {
		if i == 0 {
			result = fmt.Sprintf("%s0x%02x", result, bytes[i])
		} else {
			result = fmt.Sprintf("%s%s0x%02x", result, split, bytes[i])
		}
	}
	return result, nil

}
