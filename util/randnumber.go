package util

import (
	"encoding/binary"
	"encoding/hex"
	"crypto/rand"
	"math"
)

// new a insecure rand creator from crypto/rand seed
func newInsecureRand(n int) []byte {
	randByte := make([]byte, n)
	_, err := rand.Read(randByte)
	if err != nil {
		panic("init rand number creator failed...")
	}
	return randByte
}

// InsecureRand32 create a random number in [0 math.MaxUint32]
func InsecureRand32() uint32 {
	r := newInsecureRand(32)
	return binary.LittleEndian.Uint32(r)
}

func InsecureRand64() uint64 {
	r := newInsecureRand(64)
	return binary.LittleEndian.Uint64(r)
}

func GetRandHash() *Hash {
	hash := hex.EncodeToString(newInsecureRand(32))
	return HashFromString(hash)
}

func GetRand(nMax uint64) uint64 {
	if nMax == 0 {
		return 0
	}

	nRange := (math.MaxUint64 / nMax) * nMax
	nRand := InsecureRand64()
	for nRand >= nRange {
		nRand = InsecureRand64()
	}

	return nRand % nMax
}

func GetRandInt(nMax int) int {
	return int(GetRand(uint64(nMax)))
}
