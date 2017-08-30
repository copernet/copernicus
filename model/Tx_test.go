package model

import (
	"github.com/btcboost/copernicus/utils"

	"bytes"
	"os"
	"testing"
)

var testNewTx *Tx

func TestTxAddTxIn(t *testing.T) {
	var buf = utils.Hash{
		0x60, 0x46, 0x21, 0xeb, 0x7b, 0x9c, 0xcf, 0x44,
		0xb2, 0x12, 0xe6, 0x7b, 0x1c, 0x04, 0x80, 0x32,
		0x58, 0x6c, 0xe0, 0xcf, 0x5b, 0x36, 0xb8, 0x0b,
		0xe6, 0x01, 0x09, 0xae, 0x3b, 0xb6, 0xb5, 0x58,
	}

	testNewTx = NewTx()
	testNewTx.Hash = buf

	myOutPut := NewOutPoint(&buf, 1)
	myScriptSig := []byte{0x16, 0x00, 0x14, 0xc3, 0xe2, 0x27, 0x9d,
		0x2a, 0xc7, 0x30, 0xbd, 0x33, 0xc4, 0x61, 0x74,
		0x4d, 0x8e, 0xd8, 0xe8, 0x11, 0xf8, 0x05, 0xdb}

	myTxIn := NewTxIn(myOutPut, myScriptSig)

	testNewTx.AddTxIn(myTxIn)
}

func TestTxAddTxOut(t *testing.T) {
	script := [...]byte{0x14, 0x69, 0xe1, 0x2a, 0x40, 0xd4, 0xa2, 0x21, 0x8d, 0x33, 0xf2,
		0x08, 0xb9, 0xa0, 0x44, 0x78, 0x94, 0xdc, 0x9b, 0xea, 0x31}
	txOut := NewTxOut(9, script[:])
	testNewTx.AddTxOut(txOut)

}

func TestTxCopy(t *testing.T) {
	copyTx := testNewTx.Copy()
	if len(copyTx.Ins) != 1 {
		t.Error("should have 1 input")
	}
	if len(copyTx.Outs) != 1 {
		t.Error("should have 1 outPut")
	}

}

func TestTxSerialize(t *testing.T) {

	file, err := os.OpenFile("tmp.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	err = testNewTx.Serialize(file)
	if err != nil {
		t.Error(err)
	}

	file.Seek(0, 0)

	txDeseria := &Tx{}
	txDeseria.Deserialize(file)
	if len(txDeseria.Outs) != 1 {
		t.Errorf("the Tx should have 1 output instead of %d", len(txDeseria.Outs))
	}
	if !bytes.Equal(txDeseria.Ins[0].Script.bytes, testNewTx.Ins[0].Script.bytes) {
		t.Errorf("Deserialize() return the tx inputScript data %v"+
			"should be equal origin inputScript data %v", txDeseria.Ins[0].Script.bytes, testNewTx.Ins[0].Script.bytes)
	}
	if txDeseria.Ins[0].PreviousOutPoint.Index != testNewTx.Ins[0].PreviousOutPoint.Index {
		t.Errorf("Deserialize() return the tx preIndex data %d"+
			"should be equal origin preIndex data %d", txDeseria.Ins[0].PreviousOutPoint.Index, testNewTx.Ins[0].PreviousOutPoint.Index)
	}

	err = os.Remove("tmp.txt")
	if err != nil {
		t.Error(err)
	}
}
