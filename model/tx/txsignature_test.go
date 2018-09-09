package tx

import (
	"testing"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
)

// The test data comes from https://github.com/bitcoin/bitcoin/blob/master/src/test/data/sighash.json
// Data structure is : "raw_transaction, script, input_index, hashType, signature_hash (result)"
func TestSignatureHash(t *testing.T) {
	/*
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
			tx, err := Unserialize(bytes.NewReader(rawTx))
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

		}*/

}

func TestCheckSigHash(t *testing.T) {
	//GetHash := utils.HashFromString("d00838e883a7e7b4122ae645dbfb72de9d10df2dee058c802da170d28c4aeca3")
	//pub1, err := hex.DecodeString("03973c31b83d52eac7d1de67dcbfc564626ae2dca7198440f96d0ce6a1bcbab887")
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//pub2, err := hex.DecodeString("03b3639a9c9682c1dbd238cf4607e2bc507484b39666edd328215a379b1333defd")
	//pub3, err := hex.DecodeString("03bda29965ce46f8bc0644f4029570442fd71c6e8d2f2c7767d43e27d512a0ea4a")
	//sign1 := hexToBytes("3045022100bd14962655b13074cb3d2a9bf23569762f34de3e20461b9cfd2b14c5afb8b95f02204639a2e9f30b10c58327486f77ad9f8412f64410a0bdc412794786c9bbf0eaa341")
	//sign2 := hexToBytes("304402200ed7e308d126920dd739dcc191296d6f451117fc0b892d0022341ae5e79aeb2e02202271ca6e7799395a81a92ceaf4b15eedc56d4d18fcc32ea20a2c617b14aaf13441 ")
	//ret1, err := CheckSig(*GetHash, sign1, pub1)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret1 {
	//	t.Error("pass:03973c31b83d52eac7d1de67dcbfc564626ae2dca7198440f96d0ce6a1bcbab887")
	//
	//}
	//ret2, err := CheckSig(*GetHash, sign1, pub2)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret2 {
	//	t.Error("pass:03b3639a9c9682c1dbd238cf4607e2bc507484b39666edd328215a379b1333defd")
	//
	//}
	//ret3, err := CheckSig(*GetHash, sign1, pub3)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret3 {
	//	t.Error("pass:03bda29965ce46f8bc0644f4029570442fd71c6e8d2f2c7767d43e27d512a0ea4a")
	//
	//}
	//
	//ret1, err = CheckSig(*GetHash, sign2, pub1)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret1 {
	//	t.Error("pass:028bb6ee1127a620219c4f6fb22067536649d439929e177ebfe76386dff52a7084")
	//
	//}
	//ret2, err = CheckSig(*GetHash, sign2, pub2)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret2 {
	//	t.Error("pass:02f9cd8728b12b6c8a17a15cb4a19de000641f78a449c1b619dc271b84643ce0e9")
	//
	//}
	//ret3, err = CheckSig(*GetHash, sign2, pub3)
	//if err != nil {
	//	t.Error(err.Error())
	//}
	//if ret3 {
	//	t.Error("pass:03d33aef1ae9ecfcfa0935a8e34bb4a285cfaad1be800fc38f9fc869043c1cbee2")
	//
	//}
}

//address : 1F3sAm6ZtwLAUnj7d38pGFxtP3RVEvtsbV PrivateKey : L4rK1yDtCWekvXuE6oXD9jCYfFNV2cWRpVuPLBcCU2z8TrisoyY1
func TestSignHash(t *testing.T) {
	crypto.InitSecp256()
	var testTxs = []struct {
		txHash string
		txRaw  string
		tx     Tx
	}{
		{
			txHash: "2a6c6f76470540a3daf9c262544d40364d4aaf11275d588e54f906abc70eb4cd",
			txRaw:  "02000000010fc377638c9e677f496ab085744a859299d62f619c208928443ecffc1f1ee557010000006a473044022007e1480d5f1ddc7f9f925b98640b4813d79f94256c7b3e7999faea94b2402883022016b27df66be73546b8aba6857d1c4ac0e8bb52857ee4cb8ca69a7902c8225684012102b2481b781e230109610b7fee34738f9e051b6d4c4adbd0bfe6c0f23ad2aabb06feffffff0278a43300000000001976a91403efb6c9d387b97b598a2664225fe7b79a0ae05588acc11e0e04000000001976a914e896a034ea59dfda82f1e97bf2646eb7ab29642888aca52b0700",
			tx: Tx{
				version: 1,
				ins: []*txin.TxIn{
					txin.NewTxIn(
						&outpoint.OutPoint{
							Hash: util.Hash{
								0x0f, 0xc3, 0x77, 0x63, 0x8c, 0x9e,
								0x67, 0x7f, 0x49, 0x6a, 0xb0, 0x85,
								0x74, 0x4a, 0x85, 0x92, 0x99, 0xd6,
								0x2f, 0x61, 0x9c, 0x20, 0x89, 0x28,
								0x44, 0x3e, 0xcf, 0xfc, 0x1f, 0x1e,
								0xe5, 0x57,
							},
							Index: 0,
						},
						script.NewScriptRaw(
							[]byte{
								0x47, 0x30, 0x44, 0x02, 0x20, 0x07, 0xe1, 0x48, 0x0d, 0x5f, 0x1d, 0xdc,
								0x7f, 0x9f, 0x92, 0x5b, 0x98, 0x64, 0x0b, 0x48, 0x13, 0xd7, 0x9f, 0x94,
								0x25, 0x6c, 0x7b, 0x3e, 0x79, 0x99, 0xfa, 0xea, 0x94, 0xb2, 0x40, 0x28,
								0x83, 0x02, 0x20, 0x16, 0xb2, 0x7d, 0xf6, 0x6b, 0xe7, 0x35, 0x46, 0xb8,
								0xab, 0xa6, 0x85, 0x7d, 0x1c, 0x4a, 0xc0, 0xe8, 0xbb, 0x52, 0x85, 0x7e,
								0xe4, 0xcb, 0x8c, 0xa6, 0x9a, 0x79, 0x02, 0xc8, 0x22, 0x56, 0x84, 0x01,
								0x21, 0x02, 0xb2, 0x48, 0x1b, 0x78, 0x1e, 0x23, 0x01, 0x09, 0x61, 0x0b,
								0x7f, 0xee, 0x34, 0x73, 0x8f, 0x9e, 0x05, 0x1b, 0x6d, 0x4c, 0x4a, 0xdb,
								0xd0, 0xbf, 0xe6, 0xc0, 0xf2, 0x3a, 0xd2, 0xaa, 0xbb, 0x06,
							},
						),
						0xffffffff,
					),
				},
				outs: []*txout.TxOut{

					txout.NewTxOut(0x33a478,
						script.NewScriptRaw(
							[]byte{
								0x76, // OP_DUP
								0xa9, // OP_HASH160
								0x14, // OP_DATA_20
								0x03, 0xef, 0xb6, 0xc9,
								0xd3, 0x87, 0xb9, 0x7b,
								0x59, 0x8a, 0x26, 0x64,
								0x22, 0x5f, 0xe7, 0xb7,
								0x9a, 0x0a, 0xe0, 0x55,
								0x88, // OP_EQUALVERIFY
								0xac, // OP_CHECKSIG
							},
						)),

					txout.NewTxOut(0x40e1ec1,
						script.NewScriptRaw(
							[]byte{
								0x76, // OP_DUP
								0xa9, // OP_HASH160
								0x14, // OP_DATA_20
								0x96, 0xa0, 0x34, 0xea,
								0x59, 0xdf, 0xda, 0x82,
								0xf1, 0xe9, 0x7b, 0xf2,
								0x64, 0x6e, 0xb7, 0xab,
								0x29, 0x64, 0x28,
								0x88, // OP_EQUALVERIFY
								0xac, // OP_CHECKSIG
							},
						)),
				},
			},
		},
		{
			txHash: "e530ef9292137eb7cd493dc4dbcb4f0ae3f7b2cb5dd69b3e279075894f9ca899",
			txRaw:  "0200000001cdb40ec7ab06f9548e585d2711af4a4d36404d5462c2f9daa3400547766f6c2a000000006a473044022036d927551c0966bfaca7140a9795bf643c3781cf2ff250d8df62e89944ac2e5902206edf3646133d8d3e62f56161e342ac9ef5607c7b74deb897c04a8146cb4664b4012103e46a27ff7d97822c8c215fb8394714b134bd127911c6764da3d17ae8e83dcc39feffffff0228033100000000001976a914e168e5d549562741dc40f492b5cfdc746374c66388aca0860100000000001976a914b5bd079c4d57cc7fc28ecf8213a6b791625b818388aca62b0700",
			tx: Tx{
				version: 1,
				ins: []*txin.TxIn{
					txin.NewTxIn(
						&outpoint.OutPoint{
							Hash: util.Hash{
								0xcd, 0xb4, 0x0e, 0xc7, 0xab, 0x06,
								0xf9, 0x54, 0x8e, 0x58, 0x5d, 0x27,
								0x11, 0xaf, 0x4a, 0x4d, 0x36, 0x40,
								0x4d, 0x54, 0x62, 0xc2, 0xf9, 0xda,
								0xa3, 0x40, 0x05, 0x47, 0x76, 0x6f,
								0x6c, 0x2a,
							},
							Index: 0,
						},
						script.NewScriptRaw(
							[]byte{
								0x47, 0x30, 0x44, 0x02, 0x20, 0x36,
								0xd9, 0x27, 0x55, 0x1c, 0x09, 0x66,
								0xbf, 0xac, 0xa7, 0x14, 0x0a, 0x97,
								0x95, 0xbf, 0x64, 0x3c, 0x37, 0x81,
								0xcf, 0x2f, 0xf2, 0x50, 0xd8, 0xdf,
								0x62, 0xe8, 0x99, 0x44, 0xac, 0x2e,
								0x59, 0x02, 0x20, 0x6e, 0xdf, 0x36,
								0x46, 0x13, 0x3d, 0x8d, 0x3e, 0x62,
								0xf5, 0x61, 0x61, 0xe3, 0x42, 0xac,
								0x9e, 0xf5, 0x60, 0x7c, 0x7b, 0x74,
								0xde, 0xb8, 0x97, 0xc0, 0x4a, 0x81,
								0x46, 0xcb, 0x46, 0x64, 0xb4, 0x01,
								0x21, 0x03, 0xe4, 0x6a, 0x27, 0xff,
								0x7d, 0x97, 0x82, 0x2c, 0x8c, 0x21,
								0x5f, 0xb8, 0x39, 0x47, 0x14, 0xb1,
								0x34, 0xbd, 0x12, 0x79, 0x11, 0xc6,
								0x76, 0x4d, 0xa3, 0xd1, 0x7a, 0xe8,
								0xe8, 0x3d, 0xcc, 0x39,
							},
						),
						0xffffffff,
					),
				},
				outs: []*txout.TxOut{
					txout.NewTxOut(0x310328,
						script.NewScriptRaw(
							[]byte{
								0x76, // OP_DUP
								0xa9, // OP_HASH160
								0x14, // OP_DATA_20
								0xe1, 0x68, 0xe5, 0xd5,
								0x49, 0x56, 0x27, 0x41,
								0xdc, 0x40, 0xf4, 0x92,
								0xb5, 0xcf, 0xdc, 0x74,
								0x63, 0x74, 0xc6, 0x63,
								0x88, // OP_EQUALVERIFY
								0xac, // OP_CHECKSIG
							},
						)),
					txout.NewTxOut(0x186a0,
						script.NewScriptRaw(
							[]byte{
								0x76, // OP_DUP
								0xa9, // OP_HASH160
								0x14, // OP_DATA_20
								0xb5, 0xbd, 0x07, 0x9c,
								0x4d, 0x57, 0xcc, 0x7f,
								0xc2, 0x8e, 0xcf, 0x82,
								0x13, 0xa6, 0xb7, 0x91,
								0x62, 0x5b, 0x81, 0x83,
								0x88, // OP_EQUALVERIFY
								0xac, // OP_CHECKSIG
							},
						)),
				},
			},
		},
	}

	privateKey, err := crypto.DecodePrivateKey("L4rK1yDtCWekvXuE6oXD9jCYfFNV2cWRpVuPLBcCU2z8TrisoyY1")
	if err != nil {
		t.Error(err)
	}
	preTestTx := testTxs[0]
	testTx := testTxs[1]
	txHash, err := SignatureHash(&testTx.tx, preTestTx.tx.GetTxOut(0).GetScriptPubKey(),
		crypto.SigHashAll, 0, preTestTx.tx.GetTxOut(0).GetValue(), uint32(script.StandardScriptVerifyFlags))
	if err != nil {
		t.Error("chec signature failed")
	}
	signature, err := privateKey.Sign(txHash.GetCloneBytes())
	if err != nil {
		t.Error("chec signature failed")
	}
	ret := CheckSig(txHash, signature.Serialize(), privateKey.PubKey().ToBytes())

	if !ret {
		t.Error("chec signature failed")
	}
}
