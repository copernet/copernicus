package model

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		t.Error("unmarshal json is wrong  ")
	}
	for i, test := range testData {
		if i == 0 {
			continue
		}
		rawTx, _ := hex.DecodeString(test[0].(string))
		fmt.Println(rawTx)

	}

}
