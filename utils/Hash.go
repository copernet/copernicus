package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
)

const (
	HashSize          = 32
	MaxHashStringSize = HashSize * 2
	Hash160           = 20
)

type Hash [HashSize]byte

var HashZero = Hash{}
var HashOne = Hash{0x0000000000000000000000000000000000000000000000000000000000000001}

func (hash *Hash) ToString() string {
	bytes := hash.GetCloneBytes()
	for i := 0; i < HashSize/2; i++ {
		bytes[i], bytes[HashSize-1-i] = bytes[HashSize-1-i], bytes[i]
	}
	return hex.EncodeToString(bytes[:])
}

func (hash *Hash) GetCloneBytes() []byte {
	bytes := make([]byte, HashSize)
	copy(bytes, hash[:])
	return bytes
}
func (hash *Hash) ToBigInt() *big.Int {
	return new(big.Int).SetBytes(hash.GetCloneBytes())
}

func (hash *Hash) Cmp(other *Hash) int {

	if hash == nil || other == nil {
		return 0
	} else if hash == nil {
		return -1
	} else if other == nil {
		return 1
	}
	return hash.ToBigInt().Cmp(other.ToBigInt())
}
func (hash *Hash) SetBytes(bytes []byte) error {
	length := len(bytes)
	if length != HashSize {
		return fmt.Errorf("invalid hash length of %v , want %v", length, HashSize)
	}
	copy(hash[:], bytes)
	return nil
}

func (hash *Hash) IsEqual(target *Hash) bool {
	if hash == nil && target == nil {
		return true
	}
	if hash == nil || target == nil {
		return false
	}
	return *hash == *target
}

func BytesToHash(bytes []byte) (hash *Hash, err error) {
	length := len(bytes)
	if length != HashSize {
		return nil, fmt.Errorf("invalid hash length of %v , want %v", length, HashSize)
	}
	hash.SetBytes(bytes)
	return
}

func GetHashFromStr(hashStr string) (hash *Hash, err error) {
	hash = new(Hash)
	bytes, err := DecodeHash(hashStr)
	if err != nil {
		return
	}
	hash.SetBytes(bytes)
	return
}

func DecodeHash(src string) (bytes []byte, err error) {
	if len(src) > MaxHashStringSize {
		return nil, fmt.Errorf("max hash string length is %v bytes", MaxHashStringSize)
	}
	var srcBytes []byte
	var srcLen = len(src)
	if srcLen%2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1+srcLen)
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}
	var reversedHash = make([]byte, HashSize)
	_, err = hex.Decode(reversedHash[HashSize-hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return
	}
	bytes = make([]byte, HashSize)
	for i, b := range reversedHash[:HashSize/2] {
		bytes[i], bytes[HashSize-1-i] = reversedHash[HashSize-1-i], b
	}
	return
}

func CompareByHash(a, b interface{}) bool {
	comA := a.(Hash)
	comB := b.(Hash)
	ret := comA.Cmp(&comB)
	return ret > 0
}

func HashFromString(hexString string) *Hash {
	hash, err := GetHashFromStr(hexString)
	if err != nil {
		panic(err)
	}
	return hash
}
