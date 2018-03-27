package blockchain

import (
	"math/big"
	"testing"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/utils"
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

func TestGetBlockProofEquivalentTime(t *testing.T) {
	blocks := make([]*core.BlockIndex, 10000)
	for i := uint32(0); i < 10000; i++ {
		blocks[i] = &core.BlockIndex{}
		if i > 0 {
			blocks[i].Prev = blocks[i-1]
		}
		blocks[i].Height = int(i)
		blocks[i].Header.Time = 1269211443 + i*uint32(msg.ActiveNetParams.TargetTimePerBlock)
		blocks[i].Header.Bits = 0x207fffff
		blocks[i].ChainWork = *big.NewInt(0)
		if i > 0 {
			blocks[i].ChainWork = *(big.NewInt(0)).Add(&blocks[i-1].ChainWork, GetBlockProof(blocks[i]))
		}
	}

	for j := 0; j < 1000; j++ {
		p1 := blocks[utils.GetRand(10000)]
		p2 := blocks[utils.GetRand(10000)]
		p3 := blocks[utils.GetRand(10000)]
		tdiff := GetBlockProofEquivalentTime(p1, p2, p3, &msg.MainNetParams)
		if tdiff != int64(p1.GetBlockTime())-int64(p2.GetBlockTime()) {
			t.Errorf("the two value should should be equal, expect value : %d, actual value : %d ",
				tdiff, int64(p1.GetBlockTime())-int64(p2.GetBlockTime()))
			return
		}

	}
}
