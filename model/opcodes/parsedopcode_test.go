package opcodes

import (
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

func TestBytes(t *testing.T) {
	var testParseOpCode ParsedOpCode

	testParseOpCode.OpValue = OP_0
	testParseOpCode.Length = 1
	testParseOpCode.Data = nil
	testBytes, err := testParseOpCode.bytes()
	if len(testBytes) != 1 || err != nil {
		t.Error("The bytes should have only OpCode, err : ", err)
	}

	testParseOpCode.Length = -1
	testParseOpCode.Data = append(testParseOpCode.Data, 1, 2, 3, 4, 5, 6, 7, 8)
	testParseOpCode.OpValue = byte(len(testParseOpCode.Data))
	testBytes, err = testParseOpCode.bytes()
	if len(testBytes) != 10 || err != nil {
		t.Error("The bytes should have 10 byte: OpCode(1), lenth(1), data(8), err : ", err)
	}
}

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
	data := []byte {'h', 'e', 'l', 'l', 'o'}
	dataLenAsOpCode := byte(len(data))

	assert.True(t, NewParsedOpCode(dataLenAsOpCode, unused, data).CheckMinimalDataPush())
}

func TestCheckMinimalDataPush___corner_case___op_1__to__op_16(t *testing.T) {
	assert.True(t, NewParsedOpCode(OP_0, 1, nil).CheckMinimalDataPush())

	assert.True(t, NewParsedOpCode(OP_1, 1, []byte{OP_1}).CheckMinimalDataPush())
	assert.False(t, NewParsedOpCode(OP_1, 1, nil).CheckMinimalDataPush(), "invalid object: without Data")
}

func TestCheckMinimalDataPush___corner_case__negative_1(t *testing.T) {
	assert.True(t, NewParsedOpCode(OP_1NEGATE, 1, []byte{0x81}).CheckMinimalDataPush())
}

func TestCheckMinimalDataPush___should_return_false_when_opcode_is_not_for_data_push(t *testing.T) {
	assert.False(t, NewParsedOpCode(OP_1NEGATE, unused, make([]byte, 65536)).CheckMinimalDataPush())
}