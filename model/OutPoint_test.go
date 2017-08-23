package model

import (
	"io"
	"testing"

	"bytes"
	"github.com/btcboost/copernicus/utils"
)

var testOutPoint *OutPoint

func TestNewOutPoint(t *testing.T) {
	var buf utils.Hash
	for i := 0; i < utils.HashSize; i++ {
		buf[i] = byte(i + 49)
	}

	testOutPoint = NewOutPoint(&buf, 10)
	t.Log("index : ", testOutPoint.Index, " : ", testOutPoint.Hash)
	t.Log("String() : ", testOutPoint.String())
}

func TestOutPoint_WriteOutPoint(t *testing.T) {

	buf := bytes.NewBuffer(make([]byte, MaxMessagePayload))
	err := testOutPoint.WriteOutPoint(buf, 10, 1)
	if err != nil {
		t.Log(err)
		return
	}
}

func TestOutPoint_ReadOutPoint(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, MaxMessagePayload))

	txOutRead := &OutPoint{}
	txOutRead.Hash = new(utils.Hash)
	err := txOutRead.ReadOutPoint(buf, 1)
	if err != nil {
		if err != io.EOF {
			t.Log(err)
			return
		}
	}
	t.Log(txOutRead)
}
