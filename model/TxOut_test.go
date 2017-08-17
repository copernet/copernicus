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

//func NewTxOut(value int64, pkScript []byte) *TxOut {
func TestSerializeSize(t *testing.T) {
	myString := "asdqwhihnciwiqd827w7e6123cdsnvh43yt892ufimjf27rufian2yr8sacmejfgu3489utwej"
	outScript := make([]byte, len(myString))
	copy(outScript, myString)

	//1. create Transaction Out
	txOut := NewTxOut(999, outScript[:])
	fmt.Println(txOut.Value, " : ", string(txOut.Script))

	//2. write transaction Out in use fileIO
	file, err := os.OpenFile("txOut.txt", os.O_RDWR|os.O_CREATE, 0666)
	checkErr(err)
	defer file.Close()
	err = txOut.Serialize(file, 1)
	checkErr(err)

	//3. read transaction In in use fileIO
	txOutRead := &TxOut{}
	_, err = file.Seek(0, io.SeekStart)
	checkErr(err)
	err = txOutRead.Deserialize(file, 1)
	checkErr(err)

	fmt.Println((*txOutRead).Value, " : ", string(txOutRead.Script))

	//4. get The transaction Out news Serialize Size
	fmt.Println("size : ", txOutRead.SerializeSize())

}
