package utils

import (
	"encoding/hex"
	"fmt"
)

const (
	HashSize          = 32
	MaxHashStringSize = HashSize * 2
)

type Hash [HashSize]byte

func (hash *Hash) ToString() string {
	for i := 0; i < HashSize/2; i++ {
		hash[i], hash[HashSize-1-i] = hash[HashSize-1-i], hash[i]
	}
	return hex.EncodeToString(hash[:])
}

func (hash *Hash) GetCloneBytes() []byte {
	bytes := make([]byte, HashSize)
	copy(bytes, hash[:])
	return bytes
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

func HashFromString(hexString string) *Hash {
	hash, err := GetHashFromStr(hexString)
	if err != nil {
		panic(err)
	}
	return hash
}
