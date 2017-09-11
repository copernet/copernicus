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
	//	sha256Hash := core.DoubleSha256Hash(rawTx)
	//	fmt.Println("raw tx hash:", sha256Hash.ToString())
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
	//	inputIndex := int(int32(test[2].(float64)))
	//
	//	hashType := uint32(int32(test[3].(float64)))
	//
	//	hashStr := test[4].(string)
	//
	//	hash, err := SignatureHash(tx, preOutScript, hashType, inputIndex)
	//	if err != nil {
	//		t.Errorf("signature hash err (%s)", err.Error())
	//	}
	//	//hashTest := chainhash.DoubleHashB(hexToBytes("907c2bc503ade11cc3b04eb2918b6f547b0630ab569273824748c87ea14b0696526c66ba740200000000fd1f9bdd4ef073c7afc4ae00da8a66f429c917a0081ad1e1dabce28d373eab81d8628de80200000000ad042b5f25efb33beec9f3364e8a9139e8439d9d7e26529c3c30b6c3fd89f8684cfd68ea0200000000599ac2fe02a526ed040000000008535300516352515164370e010000000003006300ab2ec2291fe51c6f"))
	//	//fmt.Printf("(%s) v (%s)", hashStr, hex.EncodeToString(hashTest))
	//	if hashStr != hash.ToString() {
	//		t.Errorf("get signature hash is wrong  (%s) v (%s)", hashStr, hash.ToString())
	//
	//	}
	//
	//	if i == 1 {
	//		return
	//	}
	//
	//}

}
