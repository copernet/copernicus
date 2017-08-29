package model

import (
	"testing"
)

var p2SHScript = [23]byte{
	OP_HASH160,
	0x14, //lenth
	0x89, 0xAB, 0xCD, 0xEF, 0xAB,
	0xBA, 0xAB, 0xBA, 0xAB, 0xBA,
	0xAB, 0xBA, 0xAB, 0xBA, 0xAB,
	0xBA, 0xAB, 0xBA, 0xAB, 0xBA, //script hash
	OP_EQUAL,
}

func TestNewScriptWithRaw(t *testing.T) {
	script := NewScriptWithRaw(p2SHScript[:])
	if !script.IsPayToScriptHash() {
		t.Errorf("whether is P2SH script : %v\n", script.IsPayToScriptHash())
	}
	num, err := script.GetSigOpCount()
	if err != nil {
		t.Error("Error : CScript.GetSigOpCount failed")
	}
	if num != 0 {
		t.Errorf("Error: num is %d not 0 ", num)
	}
	//stk, err := script.ParseScript()
	//if len(stk) != 2 {
	//	t.Errorf("Error: ParseScript is %d not 2 ", len(stk))
	//}
}
