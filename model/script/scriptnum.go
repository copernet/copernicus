package script

import (
	"github.com/copernet/copernicus/errcode"
)

const (
	DefaultMaxNumSize = 4

	MaxInt32          = 1<<31 - 1
	MinInt32          = -1 << 31
)

type ScriptNum struct {
	Value int64
}

func GetScriptNum(vch []byte, requireMinimal bool, maxNumSize int) (scriptNum *ScriptNum, err error) {
	vchLen := len(vch)

	if vchLen > maxNumSize {
		err = errcode.New(errcode.ScriptErrNumberOverflow)
		scriptNum = NewScriptNum(0)
		return
	}
	// one byte should > 0
	// two bytes should > 255 or < -255
	if requireMinimal && vchLen > 0 {
		// Check that the number is encoded with the minimum possible
		// number of bytes.
		//
		// If the most-significant-byte - excluding the sign bit - is zero
		// then we're not minimal. Note how this test also rejects the
		// negative-zero encoding, 0x80.
		if vch[vchLen - 1] & 0x7f == 0 {

			// One exception: if there's more than one byte and the most
			// significant bit of the second-most-significant-byte is set
			// it would conflict with the sign bit. An example of this case
			// is +-255, which encode to 0xff00 and 0xff80 respectively.
			// (big-endian).
			if vchLen == 1 || (vch[vchLen - 2] & 0x80) == 0 {
				err = errcode.New(errcode.ScriptErrNonMinimalEncodedNumber)
				scriptNum = NewScriptNum(0)
				return
			}
		}
	}

	if vchLen == 0 {
		scriptNum = NewScriptNum(0)
		return
	}

	var v int64
	for i := 0; i < vchLen; i++ {
		v |= int64(vch[i]) << uint8(8 * i)
	}

	// If the input vector's most significant byte is 0x80, remove it from
	// the result and return a negative(set the sign bit of int64 to 1).
	if vch[vchLen -  1] & 0x80 != 0 {
		v &= ^(int64(0x80) << uint8(8 * (vchLen - 1)))
		scriptNum = NewScriptNum(-v)
		return
	}

	scriptNum = NewScriptNum(v)

	return
}

func (scriptNum *ScriptNum) ToInt32() int32 {
	if scriptNum.Value > MaxInt32 {
		return MaxInt32
	}
	if scriptNum.Value < MinInt32 {
		return MinInt32
	}

	return int32(scriptNum.Value)

}
func (scriptNum *ScriptNum) Serialize() (bytes []byte) {
	if scriptNum.Value == 0 {
		return nil
	}

	negative := scriptNum.Value < 0
	absoluteValue := scriptNum.Value

	if negative {
		absoluteValue = -scriptNum.Value
	}
	bytes = make([]byte, 0, 9)
	for absoluteValue > 0 {
		bytes = append(bytes, byte(absoluteValue & 0xff))
		absoluteValue >>= 8
	}
	// - If the most significant byte is >= 0x80 and the value is positive, push a
	// new zero-byte to make the significant byte < 0x80 again.

	// - If the most significant byte is >= 0x80 and the value is negative, push a
	// new 0x80 byte that will be popped off when converting to an integral.

	// - If the most significant byte is < 0x80 and the value is negative, add
	// 0x80 to it, since it will be subtracted and interpreted as a negative when
	// converting to an integral.

	if bytes[len(bytes) - 1] & 0x80 != 0 {
		extraByte := byte(0x00)
		if negative {
			extraByte = 0x80
		}
		bytes = append(bytes, extraByte)
	} else if negative {
		bytes[len(bytes) - 1] |= 0x80
	}

	return
}
func NewScriptNum(v int64) *ScriptNum {
	return &ScriptNum{Value: v}
}
