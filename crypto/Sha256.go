package crypto

import "github.com/btcsuite/fastsha256"

func Sha256Bytes(b []byte) []byte {
	hash := fastsha256.Sum256(b)
	return hash[:]
}
func Sha256Hash(b []byte) Hash {
	return Hash(fastsha256.Sum256(b))
}

func DoubleSha256Bytes(b []byte) []byte {
	first := fastsha256.Sum256(b)
	second := fastsha256.Sum256(first[:])
	return second[:]
}
func DoubleSha256Hash(b []byte) Hash {
	first := fastsha256.Sum256(b)
	return Hash(fastsha256.Sum256(first[:]))
}
