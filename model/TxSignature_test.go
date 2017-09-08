package model

import (
	"testing"
)

// The test data comes from https://github.com/bitcoin/bitcoin/blob/master/src/test/data/sighash.json
// Data structure is : "raw_transaction, script, input_index, hashType, signature_hash (result)"
func TestSignatureHash(t *testing.T) {
	//file, err := ioutil.ReadFile("../test/data/sighash.json")
	//if err != nil {
	//	t.Errorf("test signature hash open file error , %s", err.Error())
	//}
	//var testData [][]interface{}
	//err = json.Unmarshal(file, &testData)
	//if err != nil {
	//	t.Errorf("unmarshal json is wrong (%s)", err.Error())
	//}
	//for i, test := range testData {
	//	if i == 0 {
	//		continue
	//	}
	//	rawTx, _ := hex.DecodeString(test[0].(string))
	//	tx, err := DeserializeTx(bytes.NewReader(rawTx))
	//	if err != nil {
	//		t.Errorf("deserialize tx err (%s)", err.Error())
	//		continue
	//	}
	//	buf := new(bytes.Buffer)
	//	tx.Serialize(buf)
	//
	//	scriptBytes, _ := hex.DecodeString(test[1].(string))
	//	preOutScript := NewScriptRaw(scriptBytes)
	//
	//	inputIndex := int(test[2].(float64))
	//
	//	hashType := uint32(test[3].(float64))
	//
	//	hashStr := test[4].(string)
	//
	//	hash, err := SignatureHash(tx, preOutScript, hashType, inputIndex)
	//	if err != nil {
	//		t.Errorf("signature hash err (%s)", err.Error())
	//	}
	//	if hashStr != hash.ToString() {
	//		t.Errorf("get signature hash is wrong  (%s) v (%s)", hashStr, hash.ToString())
	//	}
	//
	//}

}
