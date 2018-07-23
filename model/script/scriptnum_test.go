package script

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

func TestGetScriptNum(t *testing.T) {
	t.Parallel()

	errNumTooBig := errors.New("script number overflow")
	errMinimalData := errors.New("non-minimally encoded script number")

	tests := []struct {
		serialized      []byte
		num             ScriptNum
		numLen          int
		minimalEncoding bool
		err             error
	}{
		// Minimal encoding must reject negative 0.
		{hexToBytes("80"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},

		// Minimally encoded valid values with minimal encoding flag.
		// Should not error and return expected integral number.
		{nil, ScriptNum{0}, DefaultMaxNumSize, true, nil},
		{hexToBytes("01"), ScriptNum{1}, DefaultMaxNumSize, true, nil},
		{hexToBytes("81"), ScriptNum{-1}, DefaultMaxNumSize, true, nil},
		{hexToBytes("7f"), ScriptNum{127}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ff"), ScriptNum{-127}, DefaultMaxNumSize, true, nil},
		{hexToBytes("8000"), ScriptNum{128}, DefaultMaxNumSize, true, nil},
		{hexToBytes("8080"), ScriptNum{-128}, DefaultMaxNumSize, true, nil},
		{hexToBytes("8100"), ScriptNum{129}, DefaultMaxNumSize, true, nil},
		{hexToBytes("8180"), ScriptNum{-129}, DefaultMaxNumSize, true, nil},
		{hexToBytes("0001"), ScriptNum{256}, DefaultMaxNumSize, true, nil},
		{hexToBytes("0081"), ScriptNum{-256}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ff7f"), ScriptNum{32767}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffff"), ScriptNum{-32767}, DefaultMaxNumSize, true, nil},
		{hexToBytes("008000"), ScriptNum{32768}, DefaultMaxNumSize, true, nil},
		{hexToBytes("008080"), ScriptNum{-32768}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffff00"), ScriptNum{65535}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffff80"), ScriptNum{-65535}, DefaultMaxNumSize, true, nil},
		{hexToBytes("000008"), ScriptNum{524288}, DefaultMaxNumSize, true, nil},
		{hexToBytes("000088"), ScriptNum{-524288}, DefaultMaxNumSize, true, nil},
		{hexToBytes("000070"), ScriptNum{7340032}, DefaultMaxNumSize, true, nil},
		{hexToBytes("0000f0"), ScriptNum{-7340032}, DefaultMaxNumSize, true, nil},
		{hexToBytes("00008000"), ScriptNum{8388608}, DefaultMaxNumSize, true, nil},
		{hexToBytes("00008080"), ScriptNum{-8388608}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffffff7f"), ScriptNum{2147483647}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffffffff"), ScriptNum{-2147483647}, DefaultMaxNumSize, true, nil},
		{hexToBytes("ffffffff7f"), ScriptNum{549755813887}, 5, true, nil},
		{hexToBytes("ffffffffff"), ScriptNum{-549755813887}, 5, true, nil},
		{hexToBytes("ffffffffffffff7f"), ScriptNum{9223372036854775807}, 8, true, nil},
		{hexToBytes("ffffffffffffffff"), ScriptNum{-9223372036854775807}, 8, true, nil},
		{hexToBytes("ffffffffffffffff7f"), ScriptNum{-1}, 9, true, nil},
		{hexToBytes("ffffffffffffffffff"), ScriptNum{1}, 9, true, nil},
		{hexToBytes("ffffffffffffffffff7f"), ScriptNum{-1}, 10, true, nil},
		{hexToBytes("ffffffffffffffffffff"), ScriptNum{1}, 10, true, nil},

		// Minimally encoded values that are out of range for data that
		// is interpreted as script numbers with the minimal encoding
		// flag set.  Should error and return 0.
		{hexToBytes("0000008000"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000008080"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000009000"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000009080"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffff00"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffff80"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000000001"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("0000000081"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffff00"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffff80"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffffff00"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffffff80"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffffff7f"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},
		{hexToBytes("ffffffffffffffff"), ScriptNum{0}, DefaultMaxNumSize, true, errNumTooBig},

		// Non-minimally encoded, but otherwise valid values with
		// minimal encoding flag.  Should error and return 0.
		{hexToBytes("00"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},       // 0
		{hexToBytes("0100"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},     // 1
		{hexToBytes("7f00"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},     // 127
		{hexToBytes("800000"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},   // 128
		{hexToBytes("810000"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},   // 129
		{hexToBytes("000100"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},   // 256
		{hexToBytes("ff7f00"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData},   // 32767
		{hexToBytes("00800000"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData}, // 32768
		{hexToBytes("ffff0000"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData}, // 65535
		{hexToBytes("00000800"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData}, // 524288
		{hexToBytes("00007000"), ScriptNum{0}, DefaultMaxNumSize, true, errMinimalData}, // 7340032
		{hexToBytes("0009000100"), ScriptNum{0}, 5, true, errMinimalData},               // 16779520

		// Non-minimally encoded, but otherwise valid values without
		// minimal encoding flag.  Should not error and return expected
		// integral number.
		{hexToBytes("00"), ScriptNum{0}, DefaultMaxNumSize, false, nil},
		{hexToBytes("0100"), ScriptNum{1}, DefaultMaxNumSize, false, nil},
		{hexToBytes("7f00"), ScriptNum{127}, DefaultMaxNumSize, false, nil},
		{hexToBytes("800000"), ScriptNum{128}, DefaultMaxNumSize, false, nil},
		{hexToBytes("810000"), ScriptNum{129}, DefaultMaxNumSize, false, nil},
		{hexToBytes("000100"), ScriptNum{256}, DefaultMaxNumSize, false, nil},
		{hexToBytes("ff7f00"), ScriptNum{32767}, DefaultMaxNumSize, false, nil},
		{hexToBytes("00800000"), ScriptNum{32768}, DefaultMaxNumSize, false, nil},
		{hexToBytes("ffff0000"), ScriptNum{65535}, DefaultMaxNumSize, false, nil},
		{hexToBytes("00000800"), ScriptNum{524288}, DefaultMaxNumSize, false, nil},
		{hexToBytes("00007000"), ScriptNum{7340032}, DefaultMaxNumSize, false, nil},
		{hexToBytes("0009000100"), ScriptNum{16779520}, 5, false, nil},
	}
	for _, test := range tests {
		// Ensure the error code is of the expected type and the error
		// code matches the value specified in the test instance.
		gotNum, err := GetScriptNum(test.serialized, test.minimalEncoding,
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
		num        ScriptNum
		serialized []byte
	}{
		{ScriptNum{0}, nil},
		{ScriptNum{1}, hexToBytes("01")},
		{ScriptNum{-1}, hexToBytes("81")},
		{ScriptNum{127}, hexToBytes("7f")},
		{ScriptNum{-127}, hexToBytes("ff")},
		{ScriptNum{128}, hexToBytes("8000")},
		{ScriptNum{-128}, hexToBytes("8080")},
		{ScriptNum{129}, hexToBytes("8100")},
		{ScriptNum{-129}, hexToBytes("8180")},
		{ScriptNum{256}, hexToBytes("0001")},
		{ScriptNum{-256}, hexToBytes("0081")},
		{ScriptNum{32767}, hexToBytes("ff7f")},
		{ScriptNum{-32767}, hexToBytes("ffff")},
		{ScriptNum{32768}, hexToBytes("008000")},
		{ScriptNum{-32768}, hexToBytes("008080")},
		{ScriptNum{65535}, hexToBytes("ffff00")},
		{ScriptNum{-65535}, hexToBytes("ffff80")},
		{ScriptNum{524288}, hexToBytes("000008")},
		{ScriptNum{-524288}, hexToBytes("000088")},
		{ScriptNum{7340032}, hexToBytes("000070")},
		{ScriptNum{-7340032}, hexToBytes("0000f0")},
		{ScriptNum{8388608}, hexToBytes("00008000")},
		{ScriptNum{-8388608}, hexToBytes("00008080")},
		{ScriptNum{2147483647}, hexToBytes("ffffff7f")},
		{ScriptNum{-2147483647}, hexToBytes("ffffffff")},

		// Values that are out of range for data that is interpreted as
		// numbers, but are allowed as the result of numeric operations.
		{ScriptNum{2147483648}, hexToBytes("0000008000")},
		{ScriptNum{-2147483648}, hexToBytes("0000008080")},
		{ScriptNum{2415919104}, hexToBytes("0000009000")},
		{ScriptNum{-2415919104}, hexToBytes("0000009080")},
		{ScriptNum{4294967295}, hexToBytes("ffffffff00")},
		{ScriptNum{-4294967295}, hexToBytes("ffffffff80")},
		{ScriptNum{4294967296}, hexToBytes("0000000001")},
		{ScriptNum{-4294967296}, hexToBytes("0000000081")},
		{ScriptNum{281474976710655}, hexToBytes("ffffffffffff00")},
		{ScriptNum{-281474976710655}, hexToBytes("ffffffffffff80")},
		{ScriptNum{72057594037927935}, hexToBytes("ffffffffffffff00")},
		{ScriptNum{-72057594037927935}, hexToBytes("ffffffffffffff80")},
		{ScriptNum{9223372036854775807}, hexToBytes("ffffffffffffff7f")},
		{ScriptNum{-9223372036854775807}, hexToBytes("ffffffffffffffff")},
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
		in   ScriptNum
		want int32
	}{
		// Values inside the valid int32 range are just the values
		// themselves cast to an int32.
		{ScriptNum{0}, 0},
		{ScriptNum{1}, 1},
		{ScriptNum{-1}, -1},
		{ScriptNum{127}, 127},
		{ScriptNum{-127}, -127},
		{ScriptNum{128}, 128},
		{ScriptNum{-128}, -128},
		{ScriptNum{129}, 129},
		{ScriptNum{-129}, -129},
		{ScriptNum{256}, 256},
		{ScriptNum{-256}, -256},
		{ScriptNum{32767}, 32767},
		{ScriptNum{-32767}, -32767},
		{ScriptNum{32768}, 32768},
		{ScriptNum{-32768}, -32768},
		{ScriptNum{65535}, 65535},
		{ScriptNum{-65535}, -65535},
		{ScriptNum{524288}, 524288},
		{ScriptNum{-524288}, -524288},
		{ScriptNum{7340032}, 7340032},
		{ScriptNum{-7340032}, -7340032},
		{ScriptNum{8388608}, 8388608},
		{ScriptNum{-8388608}, -8388608},
		{ScriptNum{2147483647}, 2147483647},
		{ScriptNum{-2147483647}, -2147483647},
		{ScriptNum{-2147483648}, -2147483648},

		// Values outside of the valid int32 range are limited to int32.
		{ScriptNum{2147483648}, 2147483647},
		{ScriptNum{-2147483649}, -2147483648},
		{ScriptNum{1152921504606846975}, 2147483647},
		{ScriptNum{-1152921504606846975}, -2147483648},
		{ScriptNum{2305843009213693951}, 2147483647},
		{ScriptNum{-2305843009213693951}, -2147483648},
		{ScriptNum{4611686018427387903}, 2147483647},
		{ScriptNum{-4611686018427387903}, -2147483648},
		{ScriptNum{9223372036854775807}, 2147483647},
		{ScriptNum{-9223372036854775808}, -2147483648},
	}

	for _, test := range tests {
		got := test.in.ToInt32()
		if got != test.want {
			t.Errorf("Int32: did not get expected value for %d - "+
				"got %d, want %d", test.in.Value, got, test.want)
			continue
		}
	}
}
