package scripts

import (
	"fmt"
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
	cScript := NewScriptWithRaw(p2SHScript[:])
	t.Logf("whether is P2SH script : %v\n", cScript.IsPayToScriptHash())

	num, err := cScript.GetSigOpCount()
	if err != nil {
		t.Error("Error : cScript.GetSigOpCount()")
		return
	}
	t.Logf("getOpCount : %d\n", num)
	stk, err := cScript.ParseScript()
	fmt.Println(stk, err)
}
