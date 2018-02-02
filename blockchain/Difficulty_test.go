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
