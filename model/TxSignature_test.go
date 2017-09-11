package model

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"testing"
)

// The test data comes from https://github.com/bitcoin/bitcoin/blob/master/src/test/data/sighash.json
// Data structure is : "raw_transaction, script, input_index, hashType, signature_hash (result)"
func TestSignatureHash(t *testing.T) {
	file, err := ioutil.ReadFile("../test/data/sighash.json")
	if err != nil {
		t.Errorf("test signature hash open file error , %s", err.Error())
	}
	var testData [][]interface{}
	err = json.Unmarshal(file, &testData)
	if err != nil {
		t.Errorf("unmarshal json is wrong (%s)", err.Error())
	}
	for i, test := range testData {

		if i == 0 {
			continue
		}
		hashStr := test[4].(string)
		//if hashStr != "a7aff48f3b8aeb7a4bfe2e6017c80a84168487a69b69e46681e0d0d8e63a84b6" {
		//	continue
		//}
		rawTx, _ := hex.DecodeString(test[0].(string))
		//fmt.Println("raw string:" + test[0].(string))
		tx, err := DeserializeTx(bytes.NewReader(rawTx))
		if err != nil {
			t.Errorf("deserialize tx err (%s) , raw:%s", err.Error(), rawTx)
			continue
		}
		buf := new(bytes.Buffer)
		tx.Serialize(buf)
		scriptBytes, _ := hex.DecodeString(test[1].(string))
		preOutScript := NewScriptRaw(scriptBytes)
		//fmt.Println("preOutScript:" + hex.EncodeToString(preOutScript.bytes))
		inputIndex := int(int32(test[2].(float64)))

		hashType := uint32(int32(test[3].(float64)))

		//fmt.Println("tx string :" + tx.String())
		//fmt.Println("tx bytes :" + hex.EncodeToString(buf.Bytes()))
		buf.Reset()
		tx.Serialize(buf)
		hash, err := SignatureHash(tx, preOutScript, hashType, inputIndex)

		if err != nil {
			t.Errorf("signature hash err (%s)", err.Error())
		}
		if hashStr != hash.ToString() {
			t.Errorf("get signature hash is wrong  (%s) v (%s)", hashStr, hash.ToString())

		}

	}

}
