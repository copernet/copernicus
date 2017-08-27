package model

import (
	"testing"
)

func TestGetCScriptNum(t *testing.T) {
	buf := make([]byte, 4)
	copy(buf, "1234")
	script, err := GetCScriptNum(buf, true, 1024)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(script)
}
