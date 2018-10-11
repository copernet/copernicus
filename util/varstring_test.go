package util

import (
	"bytes"
	"testing"
)

func TestVarString(t *testing.T) {
	bs := bytes.NewBuffer(nil)
	context := "hello copernicus"

	if err := WriteVarString(bs, context); err != nil {
		t.Errorf("WriteVarString : %v", err)
	}

	readStr, err := ReadVarString(bs)
	if err != nil {
		t.Error(err.Error())
	}

	if readStr != context {
		t.Errorf("the readStr: %s not equal context: %s", readStr, context)
	}
}
