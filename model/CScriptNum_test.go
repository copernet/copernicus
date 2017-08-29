package model

import (
	"encoding/hex"
	"testing"
)

func TestGetCScriptNum(t *testing.T) {
	buf := make([]byte, 0)
	buf = append(buf, 4)
	scriptNum, err := GetCScriptNum(buf, true, 1024)
	if err != nil {
		t.Error(err)
		return
	}
	value := scriptNum.Int32()
	if value != 4 {
		t.Errorf(" value is %d not 4", value)
	}
	bytes := scriptNum.Serialize()
	str := hex.EncodeToString(bytes)
	if str != "04" {
		t.Errorf("value is %s not 4", str)
	}
	t.Log(scriptNum)
}
