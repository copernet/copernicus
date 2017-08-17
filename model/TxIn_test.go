package model

import (
	"io"
	"os"
	"testing"

	"github.com/btcboost/copernicus/utils"
)

func TestNewTxIn(t *testing.T) {

	//1. create OutPoint object
	var buf utils.Hash
	for i := 0; i < utils.HashSize; i++ {
		buf[i] = byte(i * 20)
	}
	myOutPut := NewOutPoint(&buf, 10)

	//2. create Txin object
	myString := "hwd7yduncue0qe01ie8dhuscb3etde21gdahsbchqbw1y278"
	mySigscript := make([]byte, len(myString))
	copy(mySigscript, myString)
	myTxIn := NewTxIn(myOutPut, mySigscript)

	//3. create a file to store The News
	file, err := os.OpenFile("txIn.txt", os.O_RDWR|os.O_CREATE, 0666)
	checkErr(err)
	defer file.Close()

	//4. write The transaction input into file with seria News
	err = myTxIn.Serialize(file, 1)
	checkErr(err)

	//5. get The size for Transaction Input with serial News
	t.Log("Size : ", myTxIn.SerializeSize())

	//6. seek The fileIo in origin
	_, err = file.Seek(0, io.SeekStart)
	checkErr(err)

	//7. read The news from fileIO
	myTxInNew := &TxIn{PreviousOutPoint: myOutPut}
	myTxInNew.Deserialize(file, 1)
	t.Log(myTxInNew)
}
