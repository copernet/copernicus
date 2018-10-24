// Copyright (c) 2014-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pow

import (
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/blockindex"
	"math/big"
	"testing"
)

// TestBigToCompact ensures BigToCompact converts big integers to the expected
// compact representation.
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

// TestCompactToBig ensures CompactToBig converts numbers using the compact
// representation to the expected big intergers.
func TestCompactToBig(t *testing.T) {
	tests := []struct {
		in  uint32
		out int64
	}{
		{10000000, 0},
	}

	for x, test := range tests {
		n := CompactToBig(test.in)
		want := big.NewInt(test.out)
		if n.Cmp(want) != 0 {
			t.Errorf("TestCompactToBig test #%d failed: got %d want %d\n",
				x, n.Int64(), want.Int64())
			return
		}
	}
}

// TestCalcWork ensures CalcWork calculates the expected work value from values
// in compact representation.
func TestGetBlockProof(t *testing.T) {
	tests := []struct {
		in  uint32
		out int64
	}{
		{10000000, 0},
	}

	for x, test := range tests {
		bits := uint32(test.in)
		bi := new(blockindex.BlockIndex)
		bi.SetNull()
		bi.Header.Bits = bits

		r := GetBlockProof(bi)
		if r.Int64() != test.out {
			t.Errorf("TestCalcWork test #%d failed: got %v want %d\n",
				x, r.Int64(), test.out)
			return
		}
	}
}

func TestGetBlockProofEquivalentTime(t *testing.T) {
	blocks := make([]*blockindex.BlockIndex, 115)
	currentPow := big.NewInt(0).Rsh(model.ActiveNetParams.PowLimit, 1)
	initialBits := BigToCompact(currentPow)

	// Genesis block.
	blocks[0] = new(blockindex.BlockIndex)
	blocks[0].SetNull()
	blocks[0].Height = 0
	blocks[0].Header.Time = 1269211443
	blocks[0].Header.Bits = initialBits
	blocks[0].ChainWork = *GetBlockProof(blocks[0])

	for i := 1; i < 100; i++ {
		blocks[i] = getBlockIndex(blocks[i-1], int64(model.ActiveNetParams.TargetTimePerBlock), initialBits)
	}

	actual := GetBlockProofEquivalentTime(blocks[10], blocks[20], blocks[99], model.ActiveNetParams)
	exp := int64(-6000)
	if actual != exp {
		t.Errorf("GetBlockProofEquivalentTime1 Error, exp = %d, actual = %d", exp, actual)
	}
	actual = GetBlockProofEquivalentTime(blocks[20], blocks[10], blocks[99], model.ActiveNetParams)
	exp = int64(6000)
	if actual != exp {
		t.Errorf("GetBlockProofEquivalentTime2 Error, exp = %d, actual = %d", exp, actual)
	}
}
