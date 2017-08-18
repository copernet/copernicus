package model

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func checkErr(err error) {
	if err != nil {
		if err != io.EOF {
			fmt.Println("Errors : ", err)
			os.Exit(1)
		}
	}
}

var testTxout *TxOut

func TestNewTxOut(t *testing.T) {
	myString := "asdqwhihnciwiqd827w7e6123cdsnvh43yt892ufimjf27rufian2yr8sacmejfgu3489utwej"
	outScript := make([]byte, len(myString))
	copy(outScript, myString)

	testTxout = NewTxOut(999, outScript[:])
	t.Log(testTxout.Value, " : ", string(testTxout.Script))
}

func TestSerialize(t *testing.T) {

	testFile, err := os.OpenFile("txOut.txt", os.O_RDWR|os.O_CREATE, 0666)
	checkErr(err)
	defer testFile.Close()
	err = testTxout.Serialize(testFile, 1)
	checkErr(err)
}

func TestDeserialize(t *testing.T) {
	txOutRead := &TxOut{}
	testFile, err := os.OpenFile("txOut.txt", os.O_RDWR|os.O_CREATE, 0666)
	checkErr(err)
	defer testFile.Close()
	err = txOutRead.Deserialize(testFile, 1)
	checkErr(err)
	t.Log(txOutRead.Value, " : ", string(txOutRead.Script))
}

func TestSerializeSize(t *testing.T) {
	t.Log("size : ", testTxout.SerializeSize())
}
