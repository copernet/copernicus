package model

import (
	"os"
	"testing"

	"github.com/btcboost/copernicus/utils"
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
	file, err := os.OpenFile("txIn.txt", os.O_RDWR|os.O_CREATE, 0666)
	checkErr(err)
	defer file.Close()
	err = testTxIn.Serialize(file, 1)
	checkErr(err)
}

func TestTxIn_Deserialize(t *testing.T) {

	file, err := os.OpenFile("txIn.txt", os.O_RDWR|os.O_CREATE, 0666)
	checkErr(err)
	defer file.Close()

	testOutPut := &OutPoint{}
	testOutPut.Hash = new(utils.Hash)
	myTxInNew := &TxIn{PreviousOutPoint: testOutPut}
	myTxInNew.Deserialize(file, 1)
	t.Log(myTxInNew)
}

func TestTxIn_SerializeSize(t *testing.T) {
	t.Log("Size : ", testTxIn.SerializeSize())

}
