package script

import (
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
)

const (
	DefaultMaxNumSize = 4

	MaxInt32 = 1<<31 - 1
	MinInt32 = -1 << 31
)

type ScriptNum struct {
	Value int64
}

// Bytes returns the number serialized as a little endian with a sign bit.
//
// Example encodings:
//       127 -> [0x7f]
//      -127 -> [0xff]
//       128 -> [0x80 0x00]
//      -128 -> [0x80 0x80]
//       129 -> [0x81 0x00]
//      -129 -> [0x81 0x80]
//       256 -> [0x00 0x01]
//      -256 -> [0x00 0x81]
//     32767 -> [0xff 0x7f]
//    -32767 -> [0xff 0xff]
//     32768 -> [0x00 0x80 0x00]
//    -32768 -> [0x00 0x80 0x80]
func (n ScriptNum) Bytes() []byte {
	// Zero encodes as an empty byte slice.
	if n.Value == 0 {
		return nil
	}

	// Take the absolute value and keep track of whether it was originally
	// negative.
	isNegative := n.Value < 0
	if isNegative {
		n.Value = -n.Value
	}

	// Encode to little endian.  The maximum number of encoded bytes is 9
	// (8 bytes for max int64 plus a potential byte for sign extension).
	result := make([]byte, 0, 9)
	for n.Value > 0 {
		result = append(result, byte(n.Value&0xff))
		n.Value >>= 8
	}

	// When the most significant byte already has the high bit set, an
	// additional high byte is required to indicate whether the number is
	// negative or positive.  The additional byte is removed when converting
	// back to an integral and its high bit is used to denote the sign.
	//
	// Otherwise, when the most significant byte does not already have the
	// high bit set, use it to indicate the value is negative, if needed.
	// get the value of the last byte of result and 0x80. If it is not equal to 0, the 8th bit has been set.
	if result[len(result)-1]&0x80 != 0 {
		extraByte := byte(0x00)
		if isNegative {
			extraByte = 0x80
		}
		result = append(result, extraByte)

	} else if isNegative {
		result[len(result)-1] |= 0x80
	}

	return result
}

func GetScriptNum(vch []byte, requireMinimal bool, maxNumSize int) (scriptNum *ScriptNum, err error) {
	vchLen := len(vch)

	if vchLen > maxNumSize {
		log.Debug("ScriptErrNumberOverflow")
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
		if vch[vchLen-1]&0x7f == 0 {

			// One exception: if there's more than one byte and the most
			// significant bit of the second-most-significant-byte is set
			// it would conflict with the sign bit. An example of this case
			// is +-255, which encode to 0xff00 and 0xff80 respectively.
			// (big-endian).
			if vchLen == 1 || (vch[vchLen-2]&0x80) == 0 {
				log.Debug("ScriptErrNonMinimalEncodedNumber")
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
		v |= int64(vch[i]) << uint8(8*i)
	}

	// If the input vector's most significant byte is 0x80, remove it from
	// the result and return a negative(set the sign bit of int64 to 1).
	if vch[vchLen-1]&0x80 != 0 {
		v &= ^(int64(0x80) << uint8(8*(vchLen-1)))
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
		bytes = append(bytes, byte(absoluteValue&0xff))
		absoluteValue >>= 8
	}
	// - If the most significant byte is >= 0x80 and the value is positive, push a
	// new zero-byte to make the significant byte < 0x80 again.

	// - If the most significant byte is >= 0x80 and the value is negative, push a
	// new 0x80 byte that will be popped off when converting to an integral.

	// - If the most significant byte is < 0x80 and the value is negative, add
	// 0x80 to it, since it will be subtracted and interpreted as a negative when
	// converting to an integral.

	if bytes[len(bytes)-1]&0x80 != 0 {
		extraByte := byte(0x00)
		if negative {
			extraByte = 0x80
		}
		bytes = append(bytes, extraByte)
	} else if negative {
		bytes[len(bytes)-1] |= 0x80
	}

	return
}
func NewScriptNum(v int64) *ScriptNum {
	return &ScriptNum{Value: v}
}
