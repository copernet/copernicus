package blockchain

import (
	"math/big"
	"testing"
)

func TestBigToCompact(t *testing.T) {
	tests := []struct {
		in  int64
		out uint32
	}{
		{0, 0},
		{-1, 25231360},
	}

	for x, test := range tests {
		n := big.NewInt(test.in)
		r := BigToCompact(n)
		if r != test.out {
			t.Errorf("TestBigToCompact test #%d failed: got %d want %d\n",
				x, r, test.out)
			return
		}
	}
}

func TestCompactToBig(t *testing.T) {
	tests := []struct {
		in  string
		out uint32
	}{
		{"0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0x200fffff},
		{"00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 486604799},
	}

	bits := CompactToBig(0x200fffff)
	if BigToCompact(bits) != 0x200fffff {
		t.Error()
	}

	bigInt := new(big.Int)
	for x, test := range tests {
		bigInt.SetString(test.in, 16)
		if BigToCompact(bigInt) != test.out {
			t.Errorf("TestBigToCompact test #%d failed: got %d want %d\n",
				x, BigToCompact(bigInt), test.out)
			return
		}
	}
}

/*
func GetRand(nmax uint64) uint64 {
	if nmax == 0{
		return  0
	}

	// The range of the random source must be a multiple of the modulus to give
	// every possible output value an equal possibility
	nRange := (math.MaxUint64 / nmax) * nmax
	nRand := uint64(0)

	for nRand >= nRange{
		GetRandBytes(&nRand, unsafe.Sizeof(nRand))
	}

	return nRand % nmax
}

func TestGetBlockProofEquivalentTime(t *testing.T) {
	blocks := make([]*BlockIndex, 1000)

	for i := uint32(0); i < 1000; i++ {
		blocks[i].PPrev = nil
		if i > 0 {
			blocks[i].PPrev = blocks[i-1]
		}
		blocks[i].Height = int(i)
		blocks[i].Time = 1269211443 + i*uint32(msg.ActiveNetParams.TargetTimePerBlock)
		blocks[i].Bits = 0x207fffff
		blocks[i].ChainWork = *big.NewInt(0)
		if i > 0 {
			blocks[i].ChainWork = *(big.NewInt(0)).Add(&blocks[i-1].ChainWork, GetBlockProof(blocks[i]))
		}
	}

	for j := 0; j < 1000 ; j++{
		p1 := blocks[GetRand(10000)]
		p2 := blocks[GetRand(10000)]
		p3 := blocks[GetRand(10000)]
		tdiff := GetBlockProofEquivalentTime(p1, p2, p3, msg.ActiveNetParams)
		if tdiff != int64(p1.GetBlockTime() - p2.GetBlockTime()) {
			t.Errorf("the two value should should be equal, expect value : %d, actual value : %d ",
				tdiff, p1.GetBlockTime() - p2.GetBlockTime())
			return
		}
	}
}
*/
