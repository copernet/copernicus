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
	//myString := "asdqwhihnciwiqd827w7e6123cdsnvh43yt892ufimjf27rufian2yr8sacmejfgu3489utwej"
	//outScript := make([]byte, len(myString))
	//copy(outScript, myString)
	//
	//testTxout = NewTxOut(999, outScript[:])
	//t.Log(testTxout.Value, " : ", string(testTxout.Script))
}

func TestSerialize(t *testing.T) {
	//buf := bytes.NewBuffer(make([]byte, MaxMessagePayload))
	//err := testTxout.Serialize(buf, 1)
	//checkErr(err)
}

func TestDeserialize(t *testing.T) {
	//txOutRead := &TxOut{}
	//buf := bytes.NewBuffer(make([]byte, MaxMessagePayload))
	//err := txOutRead.Deserialize(buf, 1)
	//checkErr(err)
	//t.Log(txOutRead.Value, " : ", string(txOutRead.Script))
}

func TestSerializeSize(t *testing.T) {
	//t.Log("size : ", testTxout.SerializeSize())
}
