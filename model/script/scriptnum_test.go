package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	return b
}

func TestGetCScriptNum(t *testing.T) {
	t.Parallel()

	errNumTooBig := errors.New("script number overflow")
	errMinimalData := errors.New("non-minimally encoded script number")

	tests := []struct {
		serialized      []byte
		num             CScriptNum
		numLen          int
		minimalEncoding bool
		err             error
	}{
		// Minimal encoding must reject negative 0.
		{hexToBytes("80"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},

		// Minimally encoded valid values with minimal encoding flag.
		// Should not error and return expected integral number.
		{nil, CScriptNum{0}, DefaultMaxNumSize, true, nil},
		{hexToBytes("01"), CScriptNum{1}, DefaultMaxNumSize, true, nil},
		{hexToBytes("81"), CScriptNum{-1}, DefaultMaxNumSize, true, nil},
		{hexToBytes("7f"), CScriptNum{127}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ff"), CScriptNum{-127}, DefaultMaxNumSize, true, nil},
		{hexToBytes("8000"), CScriptNum{128}, DefaultMaxNumSize, true, nil},
		{hexToBytes("8080"), CScriptNum{-128}, DefaultMaxNumSize, true, nil},
		{hexToBytes("8100"), CScriptNum{129}, DefaultMaxNumSize, true, nil},
		{hexToBytes("8180"), CScriptNum{-129}, DefaultMaxNumSize, true, nil},
		{hexToBytes("0001"), CScriptNum{256}, DefaultMaxNumSize, true, nil},
		{hexToBytes("0081"), CScriptNum{-256}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ff7f"), CScriptNum{32767}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffff"), CScriptNum{-32767}, DefaultMaxNumSize, true, nil},
		{hexToBytes("008000"), CScriptNum{32768}, DefaultMaxNumSize, true, nil},
		{hexToBytes("008080"), CScriptNum{-32768}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffff00"), CScriptNum{65535}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffff80"), CScriptNum{-65535}, DefaultMaxNumSize, true, nil},
		{hexToBytes("000008"), CScriptNum{524288}, DefaultMaxNumSize, true, nil},
		{hexToBytes("000088"), CScriptNum{-524288}, DefaultMaxNumSize, true, nil},
		{hexToBytes("000070"), CScriptNum{7340032}, DefaultMaxNumSize, true, nil},
		{hexToBytes("0000f0"), CScriptNum{-7340032}, DefaultMaxNumSize, true, nil},
		{hexToBytes("00008000"), CScriptNum{8388608}, DefaultMaxNumSize, true, nil},
		{hexToBytes("00008080"), CScriptNum{-8388608}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffffff7f"), CScriptNum{2147483647}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffffffff"), CScriptNum{-2147483647}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffffffff7f"), CScriptNum{549755813887}, 5, true, nil},
		{hexToBytes("ffffffffff"), CScriptNum{-549755813887}, 5, true, nil},
		{hexToBytes("ffffffffffffff7f"), CScriptNum{9223372036854775807}, 8, true, nil},
		{hexToBytes("ffffffffffffffff"), CScriptNum{-9223372036854775807}, 8, true, nil},
		{hexToBytes("ffffffffffffffff7f"), CScriptNum{-1}, 9, true, nil},
		{hexToBytes("ffffffffffffffffff"), CScriptNum{1}, 9, true, nil},
		{hexToBytes("ffffffffffffffffff7f"), CScriptNum{-1}, 10, true, nil},
		{hexToBytes("ffffffffffffffffffff"), CScriptNum{1}, 10, true, nil},

		// Minimally encoded values that are out of range for data that
		// is interpreted as script numbers with the minimal encoding
		// flag set.  Should error and return 0.
		{hexToBytes("0000008000"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000008080"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000009000"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000009080"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffff00"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffff80"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000000001"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000000081"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffff00"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffff80"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffffff00"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffffff80"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffffff7f"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffffffff"), CScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},

		// Non-minimally encoded, but otherwise valid values with
		// minimal encoding flag.  Should error and return 0.
		{hexToBytes("00"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},       // 0
		{hexToBytes("0100"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},     // 1
		{hexToBytes("7f00"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},     // 127
		{hexToBytes("800000"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},   // 128
		{hexToBytes("810000"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},   // 129
		{hexToBytes("000100"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},   // 256
		{hexToBytes("ff7f00"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},   // 32767
		{hexToBytes("00800000"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData}, // 32768
		{hexToBytes("ffff0000"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData}, // 65535
		{hexToBytes("00000800"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData}, // 524288
		{hexToBytes("00007000"), CScriptNum{0}, DefaultMaxNumSize, true, errMinimalData}, // 7340032
		{hexToBytes("0009000100"), CScriptNum{0}, 5, true, errMinimalData},               // 16779520

		// Non-minimally encoded, but otherwise valid values without
		// minimal encoding flag.  Should not error and return expected
		// integral number.
		{hexToBytes("00"), CScriptNum{0}, DefaultMaxNumSize, false, nil},
		{hexToBytes("0100"), CScriptNum{1}, DefaultMaxNumSize, false, nil},
		{hexToBytes("7f00"), CScriptNum{127}, DefaultMaxNumSize, false, nil},
		{hexToBytes("800000"), CScriptNum{128}, DefaultMaxNumSize, false, nil},
		{hexToBytes("810000"), CScriptNum{129}, DefaultMaxNumSize, false, nil},
		{hexToBytes("000100"), CScriptNum{256}, DefaultMaxNumSize, false, nil},
		{hexToBytes("ff7f00"), CScriptNum{32767}, DefaultMaxNumSize, false, nil},
		{hexToBytes("00800000"), CScriptNum{32768}, DefaultMaxNumSize, false, nil},
		{hexToBytes("ffff0000"), CScriptNum{65535}, DefaultMaxNumSize, false, nil},
		{hexToBytes("00000800"), CScriptNum{524288}, DefaultMaxNumSize, false, nil},
		{hexToBytes("00007000"), CScriptNum{7340032}, DefaultMaxNumSize, false, nil},
		{hexToBytes("0009000100"), CScriptNum{16779520}, 5, false, nil},
	}
	for _, test := range tests {
		// Ensure the error code is of the expected type and the error
		// code matches the value specified in the test instance.
		gotNum, err := GetCScriptNum(test.serialized, test.minimalEncoding,
			test.numLen)
		if err != nil {
			if err.Error() != test.err.Error() {
				t.Errorf("makeScriptNum(%#x): got error %v, but expect %v",
					test.serialized, err, test.err)
			}
			continue
		}
		if gotNum.Value != test.num.Value {
			t.Errorf("makeScriptNum(%#x): did not get expected "+
				"number - got %d, want %d", test.serialized,
				gotNum.Value, test.num.Value)
			continue
		}
	}
}

func TestScriptNumSerialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		num        CScriptNum
		serialized []byte
	}{
		{CScriptNum{0}, nil},
		{CScriptNum{1}, hexToBytes("01")},
		{CScriptNum{-1}, hexToBytes("81")},
		{CScriptNum{127}, hexToBytes("7f")},
		{CScriptNum{-127}, hexToBytes("ff")},
		{CScriptNum{128}, hexToBytes("8000")},
		{CScriptNum{-128}, hexToBytes("8080")},
		{CScriptNum{129}, hexToBytes("8100")},
		{CScriptNum{-129}, hexToBytes("8180")},
		{CScriptNum{256}, hexToBytes("0001")},
		{CScriptNum{-256}, hexToBytes("0081")},
		{CScriptNum{32767}, hexToBytes("ff7f")},
		{CScriptNum{-32767}, hexToBytes("ffff")},
		{CScriptNum{32768}, hexToBytes("008000")},
		{CScriptNum{-32768}, hexToBytes("008080")},
		{CScriptNum{65535}, hexToBytes("ffff00")},
		{CScriptNum{-65535}, hexToBytes("ffff80")},
		{CScriptNum{524288}, hexToBytes("000008")},
		{CScriptNum{-524288}, hexToBytes("000088")},
		{CScriptNum{7340032}, hexToBytes("000070")},
		{CScriptNum{-7340032}, hexToBytes("0000f0")},
		{CScriptNum{8388608}, hexToBytes("00008000")},
		{CScriptNum{-8388608}, hexToBytes("00008080")},
		{CScriptNum{2147483647}, hexToBytes("ffffff7f")},
		{CScriptNum{-2147483647}, hexToBytes("ffffffff")},

		// Values that are out of range for data that is interpreted as
		// numbers, but are allowed as the result of numeric operations.
		{CScriptNum{2147483648}, hexToBytes("0000008000")},
		{CScriptNum{-2147483648}, hexToBytes("0000008080")},
		{CScriptNum{2415919104}, hexToBytes("0000009000")},
		{CScriptNum{-2415919104}, hexToBytes("0000009080")},
		{CScriptNum{4294967295}, hexToBytes("ffffffff00")},
		{CScriptNum{-4294967295}, hexToBytes("ffffffff80")},
		{CScriptNum{4294967296}, hexToBytes("0000000001")},
		{CScriptNum{-4294967296}, hexToBytes("0000000081")},
		{CScriptNum{281474976710655}, hexToBytes("ffffffffffff00")},
		{CScriptNum{-281474976710655}, hexToBytes("ffffffffffff80")},
		{CScriptNum{72057594037927935}, hexToBytes("ffffffffffffff00")},
		{CScriptNum{-72057594037927935}, hexToBytes("ffffffffffffff80")},
		{CScriptNum{9223372036854775807}, hexToBytes("ffffffffffffff7f")},
		{CScriptNum{-9223372036854775807}, hexToBytes("ffffffffffffffff")},
	}

	for _, test := range tests {
		gotBytes := test.num.Serialize()
		if !bytes.Equal(gotBytes, test.serialized) {
			t.Errorf("Bytes: did not get expected bytes for %d - "+
				"got %x, want %x", test.num.Value, gotBytes,
				test.serialized)
			continue
		}
	}
}

func TestScriptNumInt32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   CScriptNum
		want int32
	}{
		// Values inside the valid int32 range are just the values
		// themselves cast to an int32.
		{CScriptNum{0}, 0},
		{CScriptNum{1}, 1},
		{CScriptNum{-1}, -1},
		{CScriptNum{127}, 127},
		{CScriptNum{-127}, -127},
		{CScriptNum{128}, 128},
		{CScriptNum{-128}, -128},
		{CScriptNum{129}, 129},
		{CScriptNum{-129}, -129},
		{CScriptNum{256}, 256},
		{CScriptNum{-256}, -256},
		{CScriptNum{32767}, 32767},
		{CScriptNum{-32767}, -32767},
		{CScriptNum{32768}, 32768},
		{CScriptNum{-32768}, -32768},
		{CScriptNum{65535}, 65535},
		{CScriptNum{-65535}, -65535},
		{CScriptNum{524288}, 524288},
		{CScriptNum{-524288}, -524288},
		{CScriptNum{7340032}, 7340032},
		{CScriptNum{-7340032}, -7340032},
		{CScriptNum{8388608}, 8388608},
		{CScriptNum{-8388608}, -8388608},
		{CScriptNum{2147483647}, 2147483647},
		{CScriptNum{-2147483647}, -2147483647},
		{CScriptNum{-2147483648}, -2147483648},

		// Values outside of the valid int32 range are limited to int32.
		{CScriptNum{2147483648}, 2147483647},
		{CScriptNum{-2147483649}, -2147483648},
		{CScriptNum{1152921504606846975}, 2147483647},
		{CScriptNum{-1152921504606846975}, -2147483648},
		{CScriptNum{2305843009213693951}, 2147483647},
		{CScriptNum{-2305843009213693951}, -2147483648},
		{CScriptNum{4611686018427387903}, 2147483647},
		{CScriptNum{-4611686018427387903}, -2147483648},
		{CScriptNum{9223372036854775807}, 2147483647},
		{CScriptNum{-9223372036854775808}, -2147483648},
	}

	for _, test := range tests {
		got := test.in.Int32()
		if got != test.want {
			t.Errorf("Int32: did not get expected value for %d - "+
				"got %d, want %d", test.in.Value, got, test.want)
			continue
		}
	}
}
