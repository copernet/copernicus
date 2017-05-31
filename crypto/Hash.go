package crypto

import (
	"fmt"
	"encoding/hex"
	"sort"
)

const (
	HASH_SIZE = 32
	MAX_HASH_STRING_SIZE = HASH_SIZE * 2
)

type Hash [HASH_SIZE]byte

func (hash *Hash)ToString() string {
	sort.Reverse(hash)
	return hex.EncodeToString(hash[:])

}
func (hash *Hash)GetCloneBytes() []byte {
	bytes := make([]byte, HASH_SIZE)
	copy(bytes, hash[:])
	return bytes
}
func (hash*Hash)SetBytes(bytes [] byte) error {

	length := len(bytes)
	if length != HASH_SIZE {
		return fmt.Errorf("invalid hash length of %v , want %v", length, HASH_SIZE)

	}
	copy(hash[:], length)
	return nil
}
func (hash *Hash)IsEqual(target*Hash) bool {
	if hash == nil&&target == nil {
		return true
	}
	if hash == nil || target == nil {
		return false
	}
	return *hash == *target
}
func BytesToHash(bytes []byte) (hash *Hash, err error) {
	length := len(bytes)
	if length != HASH_SIZE {
		return nil, fmt.Errorf("invalid hash length of %v , want %v", length, HASH_SIZE)
	}
	hash = bytes
	return
}
func GetHashFromStr(hashStr string) (hash *Hash, err error) {
	bytes, err := DecodeHash(hashStr)
	if err != nil {
		return
	}
	hash = bytes
	return
}

func DecodeHash(src string) (bytes []byte, err error) {
	if len(src) > MAX_HASH_STRING_SIZE {
		return nil, fmt.Errorf("max hash string length is %v bytes", MAX_HASH_STRING_SIZE)
	}
	var srcBytes []byte
	var srcLen = len(src)
	if srcLen % 2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1 + srcLen)
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}
	var reversedHash []byte
	_, err = hex.Decode(reversedHash[HASH_SIZE - hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return
	}
	for i, b := range reversedHash[:HASH_SIZE / 2] {
		bytes[i], bytes[HASH_SIZE - 1 - i] = reversedHash[HASH_SIZE - 1 - i], b
	}
	return

}