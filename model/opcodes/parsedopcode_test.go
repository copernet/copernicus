package opcodes

import (
	"encoding/binary"
	"github.com/stretchr/testify/assert"
	"testing"
)

//func TestIsDisabled(t *testing.T) {
//
//	tests := []byte{OP_CAT, OP_SUBSTR, OP_LEFT, OP_RIGHT, OP_INVERT,
//		OP_AND, OP_OR, OP_2MUL, OP_2DIV, OP_MUL, OP_DIV, OP_MOD,
//		OP_LSHIFT, OP_RSHIFT,
//	}
//
//	for _, opcodeVal := range tests {
//
//		pop := ParsedOpCode{OpValue: opcodeVal}
//		if !pop.isDisabled() {
//			t.Errorf("%s OpCode should be Disabled ", GetOpName(int(opcodeVal)))
//		}
//	}
//
//}

const unused int = 0

func TestCheckCompactDataPush___happy_path(t *testing.T) {
	assert.True(t, NewParsedOpCode(OP_0, unused, nil).CheckCompactDataPush())
	assert.True(t, NewParsedOpCode(1, unused, make([]byte, 1)).CheckCompactDataPush())
	assert.True(t, NewParsedOpCode(75, unused, make([]byte, 75)).CheckCompactDataPush())

	assert.True(t, NewParsedOpCode(OP_PUSHDATA1, unused, make([]byte, 76)).CheckCompactDataPush())
	assert.True(t, NewParsedOpCode(OP_PUSHDATA1, unused, make([]byte, 255)).CheckCompactDataPush())

	assert.True(t, NewParsedOpCode(OP_PUSHDATA2, unused, make([]byte, 256)).CheckCompactDataPush())
	assert.True(t, NewParsedOpCode(OP_PUSHDATA2, unused, make([]byte, 65535)).CheckCompactDataPush())

	assert.True(t, NewParsedOpCode(OP_PUSHDATA4, unused, make([]byte, 65536)).CheckCompactDataPush())
	assert.True(t, NewParsedOpCode(OP_PUSHDATA4, unused, make([]byte, 65537)).CheckCompactDataPush())
}

func TestCheckCompactDataPush___push_data_not_in_compactest_way(t *testing.T) {
	var unused int

	assert.False(t, NewParsedOpCode(OP_PUSHDATA1, unused, make([]byte, 1)).CheckCompactDataPush())
	assert.False(t, NewParsedOpCode(OP_PUSHDATA2, unused, make([]byte, 1)).CheckCompactDataPush())
	assert.False(t, NewParsedOpCode(OP_PUSHDATA4, unused, make([]byte, 1)).CheckCompactDataPush())
}

func TestCheckCompactDataPush___opcode_is_not_for_data_push(t *testing.T) {
	var unused int

	assert.False(t, NewParsedOpCode(OP_1NEGATE, unused, make([]byte, 1)).CheckCompactDataPush())
	assert.False(t, NewParsedOpCode(OP_1NEGATE, unused, make([]byte, 75)).CheckCompactDataPush())
	assert.False(t, NewParsedOpCode(OP_1NEGATE, unused, make([]byte, 255)).CheckCompactDataPush())
	assert.False(t, NewParsedOpCode(OP_1NEGATE, unused, make([]byte, 65535)).CheckCompactDataPush())
	assert.False(t, NewParsedOpCode(OP_1NEGATE, unused, make([]byte, 65536)).CheckCompactDataPush())
}

func TestCheckMinimalDataPush___happy_path(t *testing.T) {
	data := []byte{'h', 'e', 'l', 'l', 'o'}
	dataLenAsOpCode := byte(len(data))

	assert.True(t, NewParsedOpCode(dataLenAsOpCode, unused, data).CheckMinimalDataPush())
}

func TestCheckMinimalDataPush___corner_case___op_0__to__op_16(t *testing.T) {
	assert.True(t, NewParsedOpCode(OP_0, unused, nil).CheckMinimalDataPush())

	assert.True(t, NewParsedOpCode(OP_1, unused, []byte{OP_1}).CheckMinimalDataPush())
	assert.False(t, NewParsedOpCode(OP_1, unused, nil).CheckMinimalDataPush(), "invalid object: without Data")
}

func TestCheckMinimalDataPush___corner_case__negative_1(t *testing.T) {
	assert.True(t, NewParsedOpCode(OP_1NEGATE, unused, []byte{0x81}).CheckMinimalDataPush())
}

func TestCheckMinimalDataPush___should_return_false_when_opcode_is_not_for_data_push(t *testing.T) {
	assert.False(t, NewParsedOpCode(OP_1NEGATE, unused, make([]byte, 65536)).CheckMinimalDataPush())
}

func TestBytes___case___op_0__to__op_16(t *testing.T) {
	bytes, err := NewParsedOpCode(OP_0, 1, nil).bytes()
	assert.Nil(t, err)
	assert.Equal(t, []byte{0}, bytes)

	bytes, err = NewParsedOpCode(OP_1, 1, nil).bytes()
	assert.Nil(t, err)
	assert.Equal(t, []byte{0x51}, bytes)

	bytes, err = NewParsedOpCode(OP_9, 1, nil).bytes()
	assert.Nil(t, err)
	assert.Equal(t, []byte{0x59}, bytes)
}

func TestBytes___case___encoding_op_pushdata1(t *testing.T) {
	const dataLen int = 200

	data := make([]byte, dataLen)
	bytes, err := NewParsedOpCode(OP_PUSHDATA1, -1, data).bytes()

	assert.Nil(t, err)
	assert.Equal(t, 1 /*OP_PUSHDATA1*/ +1 /*length*/ +len(data) /*data*/, len(bytes))
	assert.Equal(t, byte(dataLen), bytes[1], "the length byte should be equal to len(data), which is 200")
}

func TestBytes___case___encoding_op_pushdata2(t *testing.T) {
	const dataLen int = 65535

	data := make([]byte, dataLen)
	bytes, err := NewParsedOpCode(OP_PUSHDATA1, -2, data).bytes()

	assert.Nil(t, err)
	assert.Equal(t, 1 /*OP_PUSHDATA1*/ +2 /*2 byte length*/ +len(data) /*data*/, len(bytes))

	encodedLength := binary.LittleEndian.Uint16(bytes[1:])
	assert.Equal(t, uint16(dataLen), encodedLength, "should correctly encode the 2 bytes length in LittleEndian")
}
