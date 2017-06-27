package crypto

import (
	"github.com/btcsuite/fastsha256"
	"github.com/btccom/copernicus/utils"
)

func Sha256Bytes(b []byte) []byte {
	hash := fastsha256.Sum256(b)
	return hash[:]
}
func Sha256Hash(b []byte) utils.Hash {
	return utils.Hash(fastsha256.Sum256(b))
}

func DoubleSha256Bytes(b []byte) []byte {
	first := fastsha256.Sum256(b)
	second := fastsha256.Sum256(first[:])
	return second[:]
}
func DoubleSha256Hash(b []byte) utils.Hash {
	first := fastsha256.Sum256(b)
	return utils.Hash(fastsha256.Sum256(first[:]))
}
