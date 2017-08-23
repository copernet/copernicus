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

	myString := "hwd7yduncue0qe01ie8dhuscb3etde21gdahsbchqbw1y278"
	mySigscript := make([]byte, len(myString))
	copy(mySigscript, myString)
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
