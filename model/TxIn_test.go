package model

import (
	"github.com/btcboost/copernicus/utils"
	"testing"

	"bytes"
	"os"
)

var testTxIn *TxIn

func TestNewTxIn(t *testing.T) {

	var preHash = utils.Hash{
		0xc1, 0x60, 0x7e, 0x00, 0x31, 0xbc, 0xb1, 0x57,
		0xa3, 0xb2, 0xfd, 0x73, 0x0e, 0xcf, 0xac, 0xd1,
		0x6e, 0xda, 0x9d, 0x95, 0x7c, 0x5e, 0x03, 0xfa,
		0x34, 0x4e, 0x50, 0x21, 0xbb, 0x07, 0xcc, 0xbe,
	}
	outPut := NewOutPoint(&preHash, 1)

	myScriptSig := []byte{0x16, 0x00, 0x14, 0xc3, 0xe2, 0x27, 0x9d,
		0x2a, 0xc7, 0x30, 0xbd, 0x33, 0xc4, 0x61, 0x74,
		0x4d, 0x8e, 0xd8, 0xe8, 0x11, 0xf8, 0x05, 0xdb}
	sigScript := make([]byte, len(myScriptSig))
	copy(sigScript, myScriptSig)

	testTxIn = NewTxIn(outPut, sigScript)
	if !bytes.Equal(testTxIn.Script.bytes, myScriptSig) {
		t.Error("the two slice should be equal")
	}
	if testTxIn.PreviousOutPoint.Index != 1 {
		t.Error("The preOut index should be 1 instead of ", testTxIn.PreviousOutPoint.Index)
	}
	if !bytes.Equal(testTxIn.PreviousOutPoint.Hash[:], preHash[:]) {
		t.Error("The two slice should be equal")
	}

}

func TestTxIn_Serialize(t *testing.T) {

	file, err := os.OpenFile("tmp1.txt", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		t.Error(err)
	}

	err = testTxIn.Serialize(file, 1)
	if err != nil {
		t.Error(err)
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		t.Error(err)
	}
	txInRead := &TxIn{}
	txInRead.PreviousOutPoint = new(OutPoint)
	txInRead.PreviousOutPoint.Hash = new(utils.Hash)
	txInRead.Script = new(Script)

	err = txInRead.Deserialize(file, 1)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(txInRead.Script.bytes, testTxIn.Script.bytes) {
		t.Error("The two slice should be equal")
	}
	if txInRead.PreviousOutPoint.Index != testTxIn.PreviousOutPoint.Index {
		t.Error("The two index should be equal")
	}
	if !bytes.Equal(txInRead.PreviousOutPoint.Hash[:], testTxIn.PreviousOutPoint.Hash[:]) {
		t.Error("The two slice should be equal")
	}

	err = os.Remove("tmp1.txt")
	if err != nil {
		t.Error(err)
	}

}

func TestTxIn_Deserialize(t *testing.T) {

	buf := bytes.NewBuffer(make([]byte, MaxMessagePayload))
	testOutPut := &OutPoint{}
	testOutPut.Hash = new(utils.Hash)
	myTxInNew := &TxIn{PreviousOutPoint: testOutPut}
	myTxInNew.Deserialize(buf, 1)
	t.Log(myTxInNew)
}

func TestTxIn_SerializeSize(t *testing.T) {
	t.Log("Size : ", testTxIn.SerializeSize())
}
