package script

import (
	"bytes"
	"testing"
)

var p2SHScript = [23]byte{
	OP_HASH160,
	0x14, //length
	0x89, 0xAB, 0xCD, 0xEF, 0xAB,
	0xBA, 0xAB, 0xBA, 0xAB, 0xBA,
	0xAB, 0xBA, 0xAB, 0xBA, 0xAB,
	0xBA, 0xAB, 0xBA, 0xAB, 0xBA, //script txHash
	OP_EQUAL,
}

var p2PKHScript = [...]byte{
	OP_DUP,
	OP_HASH160,
	0x14,
	0x41, 0xc5, 0xda, 0x42, 0x2d,
	0x1d, 0x3e, 0x6c, 0x06, 0xaf,
	0xb1, 0x9c, 0xa6, 0x2d, 0x83,
	0xb1, 0x57, 0xfc, 0x93, 0x55,
	OP_EQUALVERIFY,
	OP_CHECKSIG,
}

func TestScriptParseScript(t *testing.T) {
	p2shScript := NewScriptRaw(p2SHScript[:])
	if !p2shScript.IsPayToScriptHash() {
		t.Errorf("the script is P2SH should be true instead of false")
	}

	stk, err := p2shScript.ParseScript()
	if len(stk) != 3 || err != nil {
		t.Errorf("the P2SH script should have 3 ParsedOpCode struct instead of %d"+
			" The err : %v", len(stk), err)
	}

	for i, parseCode := range stk {
		if i == 0 {
			if stk[i].opValue != OP_HASH160 || len(stk[i].data) != 0 {
				t.Errorf("parse index %d value should be 0xa9 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
			}
		} else if i == 1 {
			if stk[i].opValue != 0x14 || len(stk[i].data) != 0x14 {
				t.Errorf("parse index %d value should be 0x14 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
			}
		} else if i == 2 {
			if stk[i].opValue != OP_EQUAL || len(stk[i].data) != 0 {
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
			if stk[i].opValue != OP_DUP || len(stk[i].data) != 0 {
				t.Errorf("parse index %d value should be 0x76 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
			}
		} else if i == 1 {
			if stk[i].opValue != OP_HASH160 || len(stk[i].data) != 0 {
				t.Errorf("parse index %d value should be 0xa9 instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
			}
		} else if i == 2 {
			if stk[i].opValue != 0x14 || len(stk[i].data) != 0x14 {
				t.Errorf("parse index %d value should be 0x14 instead of 0x%x, dataLenth should be 20 instead of %d ", i, parseCode.opValue, len(stk[i].data))
			}
		} else if i == 3 {
			if stk[i].opValue != OP_EQUALVERIFY || len(stk[i].data) != 0 {
				t.Errorf("parse index %d value should be 0x88 instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
			}
		} else if i == 4 {
			if stk[i].opValue != OP_CHECKSIG || len(stk[i].data) != 0 {
				t.Errorf("parse index %d value should be 0xac instead of 0x%x, dataLenth should be 0 instead of %d ", i, parseCode.opValue, len(stk[i].data))
			}
		}
	}

	num, err = p2pkhScript.GetSigOpCount()
	if err != nil || num != 1 {
		t.Errorf("Error : P2PKH script have 1 OpCode instead of %d\n", num)
	}

}

func TestCScriptPushData(t *testing.T) {
	script := NewScriptRaw(make([]byte, 0))

	err := script.PushOpCode(OP_HASH160)
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
	err = script.PushOpCode(OP_EQUAL)
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
	if script.bytes[0] != OP_3 {
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
