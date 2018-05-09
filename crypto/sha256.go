package crypto

import (
	"github.com/btcboost/copernicus/util"
	"github.com/btcsuite/fastsha256"
)

func Sha256Bytes(b []byte) []byte {
	hash := fastsha256.Sum256(b)
	return hash[:util.Hash256Size]
}
func Sha256Hash(b []byte) util.Hash {
	return util.Hash(fastsha256.Sum256(b))
}

func DoubleSha256Bytes(b []byte) []byte {
	first := fastsha256.Sum256(b)
	second := fastsha256.Sum256(first[:])
	return second[:]
}
func DoubleSha256Hash(b []byte) util.Hash {
	first := fastsha256.Sum256(b)
	return util.Hash(fastsha256.Sum256(first[:util.Hash256Size]))
}

func HexToHash(str string) util.Hash {
	bytes := util.HexToBytes(str)
	if bytes == nil {
		return util.Hash{}
	}
	var hashBytes [util.Hash256Size]byte
	copy(hashBytes[:], bytes[:util.Hash256Size])
	return util.Hash(hashBytes)

}
