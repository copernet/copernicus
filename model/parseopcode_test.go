package model

import (
	"testing"
)

func TestIsDisabled(t *testing.T) {

	tests := []byte{OP_CAT, OP_SUBSTR, OP_LEFT, OP_RIGHT, OP_INVERT,
		OP_AND, OP_OR, OP_2MUL, OP_2DIV, OP_MUL, OP_DIV, OP_MOD,
		OP_LSHIFT, OP_RSHIFT,
	}

	for _, opcodeVal := range tests {

		pop := ParsedOpCode{opValue: opcodeVal}
		if !pop.isDisabled() {
			t.Errorf("%s OpCode should be Disabled ", GetOpName(int(opcodeVal)))
		}
	}

}

func TestCheckMinimalPush(t *testing.T) {
	var testParseOpCode ParsedOpCode

	testParseOpCode.opValue = OP_0
	testParseOpCode.length = 1
	testParseOpCode.data = nil
	err := testParseOpCode.checkMinimalDataPush()
	if err != nil {
		t.Error(err)
	}

	testParseOpCode.opValue = OP_15
	err = testParseOpCode.checkMinimalDataPush()
	if err == nil {
		t.Error("should have error, because the datalenth is ", err)
	}

	testParseOpCode.data = append(testParseOpCode.data, 15)
	err = testParseOpCode.checkMinimalDataPush()
	if err != nil {
		t.Error(err)
	}

	testParseOpCode.data = append(testParseOpCode.data, 15, 1, 2, 3, 4, 5, 6)
	testParseOpCode.opValue = byte(len(testParseOpCode.data))
	err = testParseOpCode.checkMinimalDataPush()
	if err != nil {
		t.Error(err)
	}
}

func TestBytes(t *testing.T) {
	var testParseOpCode ParsedOpCode

	testParseOpCode.opValue = OP_0
	testParseOpCode.length = 1
	testParseOpCode.data = nil
	testBytes, err := testParseOpCode.bytes()
	if len(testBytes) != 1 || err != nil {
		t.Error("The bytes should have only OpCode, err : ", err)
	}

	testParseOpCode.length = -1
	testParseOpCode.data = append(testParseOpCode.data, 1, 2, 3, 4, 5, 6, 7, 8)
	testParseOpCode.opValue = byte(len(testParseOpCode.data))
	testBytes, err = testParseOpCode.bytes()
	if len(testBytes) != 10 || err != nil {
		t.Error("The bytes should have 10 byte: OpCode(1), lenth(1), data(8), err : ", err)
	}

}
