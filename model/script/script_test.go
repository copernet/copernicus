package script

import (
	"github.com/copernet/copernicus/model/opcodes"
	"testing"
)

var p2SHScript = [23]byte{
	opcodes.OP_HASH160,
	0x14, //length
	0x89, 0xAB, 0xCD, 0xEF, 0xAB,
	0xBA, 0xAB, 0xBA, 0xAB, 0xBA,
	0xAB, 0xBA, 0xAB, 0xBA, 0xAB,
	0xBA, 0xAB, 0xBA, 0xAB, 0xBA, //script GetHash
	opcodes.OP_EQUAL,
}

var p2PKHScript = [...]byte{
	opcodes.OP_DUP,
	opcodes.OP_HASH160,
	0x14,
	0x41, 0xc5, 0xda, 0x42, 0x2d,
	0x1d, 0x3e, 0x6c, 0x06, 0xaf,
	0xb1, 0x9c, 0xa6, 0x2d, 0x83,
	0xb1, 0x57, 0xfc, 0x93, 0x55,
	opcodes.OP_EQUALVERIFY,
	opcodes.OP_CHECKSIG,
}

func TestScriptEncodeOPN(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		// for i in $(seq 0 16); do echo "{$i, opcodes.OP_$i},"; done
		{0, opcodes.OP_0},
		{1, opcodes.OP_1},
		{2, opcodes.OP_2},
		{3, opcodes.OP_3},
		{4, opcodes.OP_4},
		{5, opcodes.OP_5},
		{6, opcodes.OP_6},
		{7, opcodes.OP_7},
		{8, opcodes.OP_8},
		{9, opcodes.OP_9},
		{10, opcodes.OP_10},
		{11, opcodes.OP_11},
		{12, opcodes.OP_12},
		{13, opcodes.OP_13},
		{14, opcodes.OP_14},
		{15, opcodes.OP_15},
		{16, opcodes.OP_16},
	}

	for _, test := range tests {
		rv, err := EncodeOPN(test.input)
		if err != nil {
			t.Error(err)
		}
		if rv != test.expected {
			t.Errorf("EncodeOPN: expect %d got %d", test.expected, rv)
		}
	}

	_, err := EncodeOPN(opcodes.OP_16 + 1)
	if err == nil {
		t.Error("EncodeOPN(OP_16+1) expect error")
	}
}

func TestScriptDecodeOPN(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		// for i in $(seq 0 16); do echo "{opcodes.OP_$i, $i},"; done
		{opcodes.OP_0, 0},
		{opcodes.OP_1, 1},
		{opcodes.OP_2, 2},
		{opcodes.OP_3, 3},
		{opcodes.OP_4, 4},
		{opcodes.OP_5, 5},
		{opcodes.OP_6, 6},
		{opcodes.OP_7, 7},
		{opcodes.OP_8, 8},
		{opcodes.OP_9, 9},
		{opcodes.OP_10, 10},
		{opcodes.OP_11, 11},
		{opcodes.OP_12, 12},
		{opcodes.OP_13, 13},
		{opcodes.OP_14, 14},
		{opcodes.OP_15, 15},
		{opcodes.OP_16, 16},
	}

	for _, test := range tests {
		rv := DecodeOPN(byte(test.input))
		if rv != test.expected {
			t.Errorf("EncodeOPN: expect %d got %d", test.expected, rv)
		}
	}
}

func TestScriptParseScript(t *testing.T) {
	p2shScript := NewScriptRaw(p2SHScript[:])
	if !p2shScript.IsPayToScriptHash() {
		t.Errorf("the script is P2SH should be true instead of false")
	}
	/*
		stk, err := p2shScript.ParseScript()
		if len(stk) != 3 || err != nil {
			t.Errorf("the P2SH script should have 3 ParsedOpCode struct instead of %d"+
				" The err : %v", len(stk), err)
		}

		for i, parseCode := range stk {
			if i == 0 {
				if stk[i].opValue != opcodes.OP_HASH160 || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0xa9 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 1 {
				if stk[i].opValue != 0x14 || len(stk[i].data) != 0x14 {
					t.Errorf("parse index %d value should be 0x14 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 2 {
				if stk[i].opValue != opcodes.OP_EQUAL || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0x87 instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			}
		}

		num, err := p2shScript.GetSigOpCount()
		if err != nil || num != 0 {
			t.Errorf("Error : P2SH script have 0 OpCode instead of %d\n", num)
		}

		p2pkhScript := NewScriptRaw(p2PKHScript[:])
		if p2pkhScript.IsPayToScriptHash() {
			t.Error("script is P2PKH should be false instead of true")
		}

		stk, err = p2pkhScript.ParseScript()
		if len(stk) != 5 || err != nil {
			t.Errorf("the P2PKH script should have 5 ParsedOpCode struct instead of %d"+
				" The err : %v", len(stk), err)
		}

		for i, parseCode := range stk {
			if i == 0 {
				if stk[i].opValue != opcodes.OP_DUP || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0x76 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 1 {
				if stk[i].opValue != opcodes.OP_HASH160 || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0xa9 instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 2 {
				if stk[i].opValue != 0x14 || len(stk[i].data) != 0x14 {
					t.Errorf("parse index %d value should be 0x14 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 3 {
				if stk[i].opValue != opcodes.OP_EQUALVERIFY || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0x88 instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			} else if i == 4 {
				if stk[i].opValue != opcodes.OP_CHECKSIG || len(stk[i].data) != 0 {
					t.Errorf("parse index %d value should be 0xac instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
				}
			}
		}

		num, err = p2pkhScript.GetSigOpCount()
		if err != nil || num != 1 {
			t.Errorf("Error : P2PKH script have 1 OpCode instead of %d\n", num)
		}
	*/

}

/*
func TestCScriptPushData(t *testing.T) {
	script := NewScriptRaw(make([]byte, 0))

	err := script.PushOpCode(opcodes.OP_HASH160)
	if err != nil {
		t.Error(err)
	}

	data := [...]byte{
		0x89, 0xAB, 0xCD, 0xEF, 0xAB,
		0xBA, 0xAB, 0xBA, 0xAB, 0xBA,
		0xAB, 0xBA, 0xAB, 0xBA, 0xAB,
		0xBA, 0xAB, 0xBA, 0xAB, 0xBA,
	}

	script.PushData(data[:])
	err = script.PushOpCode(opcodes.OP_EQUAL)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(script.bytes, p2SHScript[:]) {
		t.Errorf("push data and OpCode composition script %v "+
			"should be equal origin script data %v", script.bytes, p2SHScript)
	}
}

func TestScriptPushInt64(t *testing.T) {
	var script Script
	script.PushInt64(3)
	if len(script.bytes) != 1 {
		t.Error("func PushInt64() error: should have one element")
	}
	if script.bytes[0] != opcodes.OP_3 {
		t.Error("func PushInt64() error: the element should be 83 instead of : ", script.bytes[0])
	}

	script.bytes = make([]byte, 0)
	script.PushInt64(35)
	if len(script.bytes) != 1 {
		t.Error("func PushInt64() error: should have one element")
	}
	if script.bytes[0] != 35 {
		t.Error("func PushInt64() error: the element should be 35 instead of : ", script.bytes[0])
	}

	script.bytes = make([]byte, 0)
	script.PushInt64(235)
	if len(script.bytes) != 2 {
		t.Errorf("func PushInt64() error: should have two element instead of %d element", len(script.bytes))
	}
	if script.bytes[0] != 235 && script.bytes[1] != 0 {
		t.Errorf("func PushInt64() error: the element should be 235 instead of : %d", script.bytes[0])
	}
}
*/
