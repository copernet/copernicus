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

// GetScriptNum interprets the passed serialized bytes as an encoded integer
// and returns the result as a script number.
//
// Since the consensus rules dictate that serialized bytes interpreted as ints
// are only allowed to be in the range determined by a maximum number of bytes,
// on a per opcode basis, an error will be returned when the provided bytes
// would result in a number outside of that range.  In particular, the range for
// the vast majority of opcodes dealing with numeric values are limited to 4
// bytes and therefore will pass that value to this function resulting in an
// allowed range of [-2^31 + 1, 2^31 - 1].
//
// The requireMinimal flag causes an error to be returned if additional checks
// on the encoding determine it is not represented with the smallest possible
// number of bytes or is the negative 0 encoding, [0x80].  For example, consider
// the number 127.  It could be encoded as [0x7f], [0x7f 0x00],
// [0x7f 0x00 0x00 ...], etc.  All forms except [0x7f] will return an error with
// requireMinimal enabled.
//
// The scriptNumLen is the maximum number of bytes the encoded value can be
// before an ErrStackNumberTooBig is returned.  This effectively limits the
// range of allowed values.
// WARNING:  Great care should be taken if passing a value larger than
// defaultScriptNumLen, which could lead to addition and multiplication
// overflows.
func GetScriptNum(vch []byte, requireMinimal bool, maxNumSize int) (scriptNum *ScriptNum, err error) {
	vchLen := len(vch)

	if vchLen > maxNumSize {
		log.Debug("ScriptErrNumberOverflow")
		err = errcode.New(errcode.ScriptErrUnknownError)
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
				err = errcode.New(errcode.ScriptErrUnknownError)
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

func (n *ScriptNum) ToInt32() int32 {
	if n.Value > MaxInt32 {
		return MaxInt32
	}
	if n.Value < MinInt32 {
		return MinInt32
	}

	return int32(n.Value)

}
func (n *ScriptNum) Serialize() (bytes []byte) {
	if n.Value == 0 {
		return nil
	}

	negative := n.Value < 0
	absoluteValue := n.Value

	if negative {
		absoluteValue = -n.Value
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

func MinimallyEncode(data []byte) []byte {
	dataLen := len(data)
	if dataLen == 0 {
		return data
	}

	// If the last byte is not 0x00 or 0x80, we are minimally encoded.
	last := data[dataLen-1]
	if (last & 0x7f) != 0 {
		return data
	}

	// If the script is one byte long, then we have a zero, which encodes as an
	// empty array.
	if len(data) == 1 {
		data = data[:0]
		return data
	}

	// If the next byte has it sign bit set, then we are minimaly encoded.
	if (data[len(data)-2] & 0x80) != 0 {
		return data
	}

	// We are not minimally encoded, we need to figure out how much to trim.
	for i := len(data) - 1; i > 0; i-- {
		// We found a non zero byte, time to encode.
		if data[i-1] != 0 {
			if (data[i-1] & 0x80) != 0 {
				// We found a byte with it sign bit set so we need one more
				// byte.
				data[i] = last
				i++
			} else {
				// the sign bit is clear, we can use it.
				data[i-1] |= last
			}
			data = data[:i]
			return data
		}
	}

	// If we the whole thing is zeros, then we have a zero.
	data = data[:0]
	return data
}

func IsMinimallyEncoded(data []byte, nMaxNumSize int64) bool {
	dataLen := len(data)
	if int64(dataLen) > nMaxNumSize {
		return false
	}

	if dataLen > 0 {
		// Check that the number is encoded with the minimum possible number
		// of bytes.
		//
		// If the most-significant-byte - excluding the sign bit - is zero
		// then we're not minimal. Note how this test also rejects the
		// negative-zero encoding, 0x80.
		if (data[dataLen-1] & 0x7f) == 0 {
			// One exception: if there's more than one byte and the most
			// significant bit of the second-most-significant-byte is set it
			// would conflict with the sign bit. An example of this case is
			// +-255, which encode to 0xff00 and 0xff80 respectively.
			// (big-endian).
			if dataLen <= 1 || (data[dataLen-2]&0x80) == 0 {
				return false
			}
		}
	}

	return true
}

func NewScriptNum(v int64) *ScriptNum {
	return &ScriptNum{Value: v}
}
