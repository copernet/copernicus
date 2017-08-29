package model

import (
	"github.com/btcboost/copernicus/utils"
	"testing"

	"bytes"
)

var testTxIn *TxIn

func TestNewTxIn(t *testing.T) {

	var buf utils.Hash
	for i := 0; i < utils.HashSize; i++ {
		buf[i] = byte(i * 20)
	}
	testOutPut := NewOutPoint(&buf, 10)

	sigScript := []byte{0x04, 0x31, 0xdc, 0x00, 0x1b, 0x01, 0x62}
	mySigscript := make([]byte, len(sigScript))
	copy(mySigscript, sigScript)
	testTxIn = NewTxIn(testOutPut, mySigscript)
}

func TestTxIn_Serialize(t *testing.T) {

	buf := bytes.NewBuffer(make([]byte, MaxMessagePayload))
	err := testTxIn.Serialize(buf, 1)
	checkErr(err)
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
