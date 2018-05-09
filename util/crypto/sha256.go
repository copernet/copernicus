package crypto

import (
	"github.com/btcboost/copernicus/utils"
	"github.com/btcsuite/fastsha256"
)

func Sha256Bytes(b []byte) []byte {
	hash := fastsha256.Sum256(b)
	return hash[:utils.Hash256Size]
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
	return utils.Hash(fastsha256.Sum256(first[:utils.Hash256Size]))
}

func HexToHash(str string) utils.Hash {
	bytes := utils.HexToBytes(str)
	if bytes == nil {
		return utils.Hash{}
	}
	var hashBytes [utils.Hash256Size]byte
	copy(hashBytes[:], bytes[:utils.Hash256Size])
	return utils.Hash(hashBytes)

}
