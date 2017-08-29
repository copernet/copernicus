package model

import (
	"bytes"
	"os"
	"testing"
)

var testTxout *TxOut

func TestNewTxOut(t *testing.T) {
	script := [...]byte{0x69, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	testTxout = NewTxOut(9, script[:])
	if testTxout.Value != 9 {
		t.Error("The value should be 9 instead of ", testTxout.Value)
	}
	if !bytes.Equal(testTxout.Script, script[:]) {
		t.Error("this data should be equal")
	}
}

func TestSerialize(t *testing.T) {

	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	err = testTxout.Serialize(file, 1)
	if err != nil {
		t.Error(err)
	}

	txOutRead := &TxOut{}
	file.Seek(0, 0)
	err = txOutRead.Deserialize(file, 1)
	if err != nil {
		t.Error(err)
	}

	if txOutRead.Value != testTxout.Value {
		t.Error("The value should be equal", txOutRead.Value, " : ", testTxout.Value)
	}

	if !bytes.Equal(txOutRead.Script, testTxout.Script) {
		t.Error("The two []byte data should be equal ", txOutRead.Script, " : ", testTxout.Script)
	}

	if testTxout.SerializeSize() != 29 {
		t.Error("the serialSize should be 29 instead of ", testTxout.SerializeSize())
	}

	err = os.Remove("tmp.txt")
	if err != nil {
		t.Error(err)
	}
}
