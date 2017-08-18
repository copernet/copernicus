package model

import (
	"io"
	"os"
	"testing"

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

	file, err := os.OpenFile("txOut.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Log(err)
		return
	}
	defer file.Close()

	err = testOutPoint.WriteOutPoint(file, 10, 1)
	if err != nil {
		t.Log(err)
		return
	}
}

func TestOutPoint_ReadOutPoint(t *testing.T) {

	file, err := os.OpenFile("txOut.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Log(err)
		return
	}
	defer file.Close()

	txOutRead := &OutPoint{}
	txOutRead.Hash = new(utils.Hash)
	err = txOutRead.ReadOutPoint(file, 1)
	if err != nil {
		if err != io.EOF {
			t.Log(err)
			return
		}
	}
	t.Log(txOutRead)
}
