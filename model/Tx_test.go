package model

import (
	"io"
	"os"
	"testing"

	"github.com/btcboost/copernicus/utils"
)

func TestSerializeSizeTx(t *testing.T) {
	//1. create Transaction
	newTx := NewTx()
	t.Log(newTx)
	var buf utils.Hash
	copy(buf[:], "adbasg7wy7yswdwiuyc78sayxchwuniuhy")
	newTx.Hash = buf

	//2. create OutPoint object
	myOutPut := NewOutPoint(&buf, 10)

	//3. create Txin object
	myString := "hwd7yduncue0qe01ie8dhuscb3etde21gdahsbchqbw1y278"
	mySigscript := make([]byte, len(myString))
	copy(mySigscript, myString)
	myTxIn := NewTxIn(myOutPut, mySigscript)

	//4. add The txIn in Tx
	newTx.AddTxIn(myTxIn)

	//5. create a TransactionOut object
	myString = "asdqwhihnciwiqd827w7e6123cdsnvh43yt892ufimjf27rufian2yr8sacmejfgu3489utwej"
	outScript := make([]byte, len(myString))
	copy(outScript, myString)
	txOut := NewTxOut(999, outScript[:])

	//6. add The TxOut in Tx
	newTx.AddTxOut(txOut)

	//7. get The Size for Serialize Size with Tx news
	t.Log(newTx.SerializeSize())
	t.Log(newTx)

	//8. copy transaction
	copyTx := newTx.Copy()
	t.Log("copyTx : ", copyTx.SerializeSize())
	t.Log("copyTx : ", copyTx)

	//9. create a file to store The News
	file, err := os.OpenFile("tx.txt", os.O_RDWR|os.O_CREATE, 0666)
	checkErr(err)
	defer file.Close()

	//10. write The news with transaction into file
	err = newTx.Serialize(file)
	checkErr(err)

	//11. seek The fileIO
	_, err = file.Seek(0, io.SeekStart)
	checkErr(err)

	//12. return scriptBuffers
	newTx.returnScriptBuffers()

	//13. read The news With transaction from file
	newTx.Deserialize(file)
}
