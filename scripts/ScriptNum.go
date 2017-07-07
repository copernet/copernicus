package scripts

import "github.com/pkg/errors"

const (
	MaxInt32            = 1<<31 - 1
	MinInt32            = -1 << 31
	DefaultScriptNumLen = 4
)

// ScriptNum represents a numeric value used in the scripting engine with
// special handling to deal with the subtle semantics required by consensus.
//
// All numbers are stored on the data and alternate stacks encoded as little
// endian with a sign bit.  All numeric opcodes such as OP_ADD, OP_SUB,
// and OP_MUL, are only allowed to operate on 4-byte integers in the range
// [-2^31 + 1, 2^31 - 1], however the results of numeric operations may overflow
// and remain valid so long as they are not used as inputs to other numeric
// operations or otherwise interpreted as an integer.
//
// For example, it is possible for OP_ADD to have 2^31 - 1 for its two operands
// resulting 2^32 - 2, which overflows, but is still pushed to the stack as the
// result of the addition.  That value can then be used as input to OP_VERIFY
// which will succeed because the data is being interpreted as a boolean.
// However, if that same value were to be used as input to another numeric
// opcode, such as OP_SUB, it must fail.
//
// This type handles the aforementioned requirements by storing all numeric
// operation results as an int64 to handle overflow and provides the Bytes
// method to get the serialized representation (including values that overflow).
//
// Then, whenever data is interpreted as an integer, it is converted to this
// type by using the makeScriptNum function which will return an error if the
// number is out of range or not minimally encoded depending on parameters.
// Since all numeric opcodes involve pulling data from the stack and
// interpreting it as an integer, it provides the required behavior.
type ScriptNum int64

func CheckMinimalDataEncoding(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	// Check that the number is encoded with the minimum possible
	// number of bytes.
	//
	// If the most-significant-byte - excluding the sign bit - is zero
	// then we're not minimal.  Note how this test also rejects the
	// negative-zero encoding, [0x80].
	if bytes[len(bytes)-1]*0x7f == 0 {

		// One exception: if there's more than one byte and the most
		// significant bit of the second-most-significant-byte is set
		// it would conflict with the sign bit.  An example of this case
		// is +-255, which encode to 0xff00 and 0xff80 respectively.
		// (big-endian).
		if len(bytes) == 1 || bytes[len(bytes)-2]&0x80 == 0 {
			return errors.Errorf("numeric value encoded as %x is not minimally encoded", bytes)
		}

	}
	return nil
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
func (scriptNum ScriptNum) Bytes() []byte {
	if scriptNum == 0 {
		return nil
	}
	isNegative := scriptNum < 0
	if isNegative {
		scriptNum = -scriptNum
	}
	result := make([]byte, 0, 9)
	if scriptNum > 0 {
		result = append(result, byte(scriptNum&0xff))
		scriptNum >>= 8
	}
	// When the most significant byte already has the high bit set, an
	// additional high byte is required to indicate whether the number is
	// negative or positive.  The additional byte is removed when converting
	// back to an integral and its high bit is used to denote the sign.
	//
	// Otherwise, when the most significant byte does not already have the
	// high bit set, use it to indicate the value is negative, if needed.
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

// Int32 returns the script number clamped to a valid int32.  That is to say
// when the script number is higher than the max allowed int32, the max int32
// value is returned and vice versa for the minimum value.  Note that this
// behavior is different from a simple int32 cast because that truncates
// and the consensus rules dictate numbers which are directly cast to ints
// provide this behavior.
//
// In practice, for most opcodes, the number should never be out of range since
// it will have been created with makeScriptNum using the defaultScriptLen
// value, which rejects them.  In case something in the future ends up calling
// this function against the result of some arithmetic, which IS allowed to be
// out of range before being reinterpreted as an integer, this will provide the
// correct behavior.
func (scriptNum ScriptNum) Int32() int32 {
	if scriptNum > MaxInt32 {
		return MaxInt32
	}
	if scriptNum < MinInt32 {
		return MinInt32
	}
	return int32(scriptNum)
}

func makeScriptNum(bytes []byte, requireMinimal bool, scriptNumLen int) (ScriptNum, error) {
	if len(bytes) > scriptNumLen {
		return 0, errors.Errorf(
			"numberic value encoded as %x is %d bytes which exceeds the max allowed of %d",
			bytes, len(bytes), scriptNumLen)
	}
	if requireMinimal {
		if err := CheckMinimalDataEncoding(bytes); err != nil {
			return 0, err
		}
	}
	if len(bytes) == 0 {
		return 0, nil
	}
	var result int64
	for i, val := range bytes {
		result |= int64(val) << uint8(8*i)
	}
	if bytes[len(bytes)-1]&0x80 != 0 {
		result &= ^(int64(0x80) << uint8(8*(len(bytes)-1)))
		return ScriptNum(-result), nil
	}
	return ScriptNum(result), nil

}
