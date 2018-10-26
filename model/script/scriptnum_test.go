package script

import (
	"bytes"
	"encoding/hex"
	"github.com/copernet/copernicus/errcode"
	"github.com/stretchr/testify/assert"
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

	//errNumTooBig := errors.New("script number overflow")
	//errMinimalData := errors.New("non-minimally encoded script number")
	errNumOverflow := errcode.New(errcode.ScriptErrUnknownError)
	errNonMinimal := errcode.New(errcode.ScriptErrUnknownError)

	tests := []struct {
		serialized      []byte
		num             ScriptNum
		numLen          int
		minimalEncoding bool
		err             error
	}{
		// Minimal encoding must reject negative 0.
		{hexToBytes("80"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal},

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
		{hexToBytes("0000008000"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("0000008080"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("0000009000"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("0000009080"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("ffffffff00"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("ffffffff80"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("0000000001"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("0000000081"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("ffffffffffff00"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("ffffffffffff80"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("ffffffffffffff00"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("ffffffffffffff80"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("ffffffffffffff7f"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},
		{hexToBytes("ffffffffffffffff"), ScriptNum{0}, DefaultMaxNumSize, true, errNumOverflow},

		// Non-minimally encoded, but otherwise valid values with
		// minimal encoding flag.  Should error and return 0.
		{hexToBytes("00"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal},       // 0
		{hexToBytes("0100"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal},     // 1
		{hexToBytes("7f00"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal},     // 127
		{hexToBytes("800000"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal},   // 128
		{hexToBytes("810000"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal},   // 129
		{hexToBytes("000100"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal},   // 256
		{hexToBytes("ff7f00"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal},   // 32767
		{hexToBytes("00800000"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal}, // 32768
		{hexToBytes("ffff0000"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal}, // 65535
		{hexToBytes("00000800"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal}, // 524288
		{hexToBytes("00007000"), ScriptNum{0}, DefaultMaxNumSize, true, errNonMinimal}, // 7340032
		{hexToBytes("0009000100"), ScriptNum{0}, 5, true, errNonMinimal},               // 16779520

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
		value := test
		num, err := GetScriptNum(value.serialized, value.minimalEncoding, value.numLen)
		assert.Equal(t, value.err, err, hex.EncodeToString(value.serialized))
		assert.Equal(t, num, &value.num)
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

func TestScriptNum_Bytes(t *testing.T) {
	tests := []struct {
		in   ScriptNum
		want []byte
	}{
		{ScriptNum{0}, nil},
		{ScriptNum{127}, []byte{0x7f}},
		{ScriptNum{-127}, []byte{0xff}},
		{ScriptNum{128}, []byte{0x80, 0x00}},
		{ScriptNum{-128}, []byte{0x80, 0x80}},
		{ScriptNum{129}, []byte{0x81, 0x00}},
		{ScriptNum{-129}, []byte{0x81, 0x80}},
		{ScriptNum{256}, []byte{0x00, 0x01}},
		{ScriptNum{-256}, []byte{0x00, 0x81}},
		{ScriptNum{32767}, []byte{0xff, 0x7f}},
		{ScriptNum{-32767}, []byte{0xff, 0xff}},
		{ScriptNum{32768}, []byte{0x00, 0x80, 0x00}},
		{ScriptNum{-32768}, []byte{0x00, 0x80, 0x80}},
	}

	for _, v := range tests {
		value := v
		result := value.in.Bytes()
		assert.Equal(t, value.want, result)
	}
}

func TestMinimallyEncode(t *testing.T) {
	tests := []struct {
		in   []byte
		want []byte
	}{
		{hexToBytes("80"), []byte{}},

		{nil, nil},

		{hexToBytes("01"), []byte{0x01}},
		{hexToBytes("81"), []byte{0x81}},
		{hexToBytes("7f"), []byte{0x7f}},
		{hexToBytes("ff"), []byte{0xff}},
		{hexToBytes("8000"), []byte{0x80, 0x00}},
		{hexToBytes("8080"), []byte{0x80, 0x80}},
		{hexToBytes("8100"), []byte{0x81, 0x00}},
		{hexToBytes("8180"), []byte{0x81, 0x80}},
		{hexToBytes("0001"), []byte{0x00, 0x01}},
		{hexToBytes("0081"), []byte{0x00, 0x81}},
		{hexToBytes("ff7f"), []byte{0xff, 0x7f}},
		{hexToBytes("ffff"), []byte{0xff, 0xff}},
		{hexToBytes("008000"), []byte{0x00, 0x80, 0x00}},
		{hexToBytes("008080"), []byte{0x00, 0x80, 0x80}},
		{hexToBytes("ffff00"), []byte{0xff, 0xff, 0x00}},
		{hexToBytes("ffff80"), []byte{0xff, 0xff, 0x80}},
		{hexToBytes("000008"), []byte{0x00, 0x00, 0x08}},
		{hexToBytes("000088"), []byte{0x00, 0x00, 0x88}},
		{hexToBytes("000070"), []byte{0x00, 0x00, 0x70}},
		{hexToBytes("0000f0"), []byte{0x00, 0x00, 0xf0}},
		{hexToBytes("00008000"), []byte{0x00, 0x00, 0x80, 0x00}},
		{hexToBytes("00008080"), []byte{0x00, 0x00, 0x80, 0x80}},
		{hexToBytes("ffffff7f"), []byte{0xff, 0xff, 0xff, 0x7f}},
		{hexToBytes("ffffffff"), []byte{0xff, 0xff, 0xff, 0xff}},
		{hexToBytes("ffffffff7f"), []byte{0xff, 0xff, 0xff, 0xff, 0x7f}},
		{hexToBytes("ffffffffff"), []byte{0xff, 0xff, 0xff, 0xff, 0xff}},
		{hexToBytes("ffffffffffffff7f"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}},
		{hexToBytes("ffffffffffffffff"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{hexToBytes("ffffffffffffffff7f"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}},
		{hexToBytes("ffffffffffffffffff"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{hexToBytes("ffffffffffffffffff7f"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}},
		{hexToBytes("ffffffffffffffffffff"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},

		{hexToBytes("0000008000"), []byte{0x00, 0x00, 0x00, 0x80, 0x00}},
		{hexToBytes("0000008080"), []byte{0x00, 0x00, 0x00, 0x80, 0x80}},
		{hexToBytes("0000009000"), []byte{0x00, 0x00, 0x00, 0x90, 0x00}},
		{hexToBytes("0000009080"), []byte{0x00, 0x00, 0x00, 0x90, 0x80}},
		{hexToBytes("ffffffff00"), []byte{0xff, 0xff, 0xff, 0xff, 0x00}},
		{hexToBytes("ffffffff80"), []byte{0xff, 0xff, 0xff, 0xff, 0x80}},
		{hexToBytes("0000000001"), []byte{0x00, 0x00, 0x00, 0x00, 0x01}},
		{hexToBytes("0000000081"), []byte{0x00, 0x00, 0x00, 0x00, 0x81}},
		{hexToBytes("ffffffffffff00"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}},
		{hexToBytes("ffffffffffff80"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x80}},
		{hexToBytes("ffffffffffffff00"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}},
		{hexToBytes("ffffffffffffff80"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x80}},
		{hexToBytes("ffffffffffffff7f"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}},
		{hexToBytes("ffffffffffffffff"), []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},

		{hexToBytes("00"), []byte{}},                               // 0
		{hexToBytes("0100"), []byte{0x01}},                         // 1
		{hexToBytes("7f00"), []byte{0x7f}},                         // 127
		{hexToBytes("800000"), []byte{0x80, 0x00}},                 // 128
		{hexToBytes("810000"), []byte{0x81, 0x00}},                 // 129
		{hexToBytes("000100"), []byte{0x00, 0x01}},                 // 256
		{hexToBytes("ff7f00"), []byte{0xff, 0x7f}},                 // 32767
		{hexToBytes("00800000"), []byte{0x00, 0x80, 0x00}},         // 32768
		{hexToBytes("ffff0000"), []byte{0xff, 0xff, 0x00}},         // 65535
		{hexToBytes("00000800"), []byte{0x00, 0x00, 0x08}},         // 524288
		{hexToBytes("00007000"), []byte{0x00, 0x00, 0x70}},         // 7340032
		{hexToBytes("0009000100"), []byte{0x00, 0x09, 0x00, 0x01}}, // 16779520

		{hexToBytes("0000000000"), []byte{}},
	}

	for _, v := range tests {
		value := v
		result := MinimallyEncode(value.in)
		assert.Equal(t, value.want, result, hex.EncodeToString(value.in))
	}
}
