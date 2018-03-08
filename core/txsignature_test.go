package core

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
		rawTx, _ := hex.DecodeString(test[0].(string))
		tx, err := DeserializeTx(bytes.NewReader(rawTx))
		if err != nil {
			t.Errorf("deserialize tx err (%s) , raw:%s", err.Error(), rawTx)
			continue
		}
		buf := new(bytes.Buffer)
		tx.Serialize(buf)
		scriptBytes, _ := hex.DecodeString(test[1].(string))
		preOutScript := NewScriptRaw(scriptBytes)
		inputIndex := int(int32(test[2].(float64)))

		hashType := uint32(int32(test[3].(float64)))

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

func TestCheckSigHash(t *testing.T) {
	//txHash := utils.HashFromString("d00838e883a7e7b4122ae645dbfb72de9d10df2dee058c802da170d28c4aeca3")
	//pub1, err := hex.DecodeString("03973c31b83d52eac7d1de67dcbfc564626ae2dca7198440f96d0ce6a1bcbab887")
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//pub2, err := hex.DecodeString("03b3639a9c9682c1dbd238cf4607e2bc507484b39666edd328215a379b1333defd")
	//pub3, err := hex.DecodeString("03bda29965ce46f8bc0644f4029570442fd71c6e8d2f2c7767d43e27d512a0ea4a")
	//sign1 := hexToBytes("3045022100bd14962655b13074cb3d2a9bf23569762f34de3e20461b9cfd2b14c5afb8b95f02204639a2e9f30b10c58327486f77ad9f8412f64410a0bdc412794786c9bbf0eaa341")
	//sign2 := hexToBytes("304402200ed7e308d126920dd739dcc191296d6f451117fc0b892d0022341ae5e79aeb2e02202271ca6e7799395a81a92ceaf4b15eedc56d4d18fcc32ea20a2c617b14aaf13441 ")
	//ret1, err := CheckSig(*txHash, sign1, pub1)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret1 {
	//	t.Error("pass:03973c31b83d52eac7d1de67dcbfc564626ae2dca7198440f96d0ce6a1bcbab887")
	//
	//}
	//ret2, err := CheckSig(*txHash, sign1, pub2)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret2 {
	//	t.Error("pass:03b3639a9c9682c1dbd238cf4607e2bc507484b39666edd328215a379b1333defd")
	//
	//}
	//ret3, err := CheckSig(*txHash, sign1, pub3)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret3 {
	//	t.Error("pass:03bda29965ce46f8bc0644f4029570442fd71c6e8d2f2c7767d43e27d512a0ea4a")
	//
	//}
	//
	//ret1, err = CheckSig(*txHash, sign2, pub1)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret1 {
	//	t.Error("pass:028bb6ee1127a620219c4f6fb22067536649d439929e177ebfe76386dff52a7084")
	//
	//}
	//ret2, err = CheckSig(*txHash, sign2, pub2)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret2 {
	//	t.Error("pass:02f9cd8728b12b6c8a17a15cb4a19de000641f78a449c1b619dc271b84643ce0e9")
	//
	//}
	//ret3, err = CheckSig(*txHash, sign2, pub3)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret3 {
	//	t.Error("pass:03d33aef1ae9ecfcfa0935a8e34bb4a285cfaad1be800fc38f9fc869043c1cbee2")
	//
	//}
}
