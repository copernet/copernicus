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

func TestCheckMinimalPush(t *testing.T) {
	var testParseOpCode ParsedOpCode

	testParseOpCode.OpValue = OP_0
	testParseOpCode.Length = 1
	testParseOpCode.Data = nil
	assert.True(t, testParseOpCode.CheckMinimalDataPush())

	testParseOpCode.OpValue = OP_15
	assert.False(t, testParseOpCode.CheckMinimalDataPush(), "check should failed, because the datalenth is 0")

	testParseOpCode.Data = append(testParseOpCode.Data, 15)
	assert.True(t, testParseOpCode.CheckMinimalDataPush())

	testParseOpCode.Data = append(testParseOpCode.Data, 15, 1, 2, 3, 4, 5, 6)
	testParseOpCode.OpValue = byte(len(testParseOpCode.Data))
	assert.True(t, testParseOpCode.CheckMinimalDataPush())
}

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
