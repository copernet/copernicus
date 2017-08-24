package scripts

import (
	"testing"
)

func TestGetCScriptNum(t *testing.T) {
	buf := make([]byte, 5)
	copy(buf, "12345")
	script, err := GetCScriptNum(buf, true, 1024)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(script)
}
