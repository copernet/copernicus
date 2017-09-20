package model

import (
	"testing"

	"bytes"
	"encoding/hex"

	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

//address : 1F3sAm6ZtwLAUnj7d38pGFxtP3RVEvtsbV;	 privateKey : L4rK1yDtCWekvXuE6oXD9jCYfFNV2cWRpVuPLBcCU2z8TrisoyY1
var testsTx = []struct {
	txHash string
	txRaw  string
	tx     Tx
}{

	{
		txHash: "d8441d9535608c347341a161754a22c546d516923111b0a44670618640be7d0b",
		txRaw:  "01000000018ef0a4665b62f74b24a2627cf6722066f4afe97f6262812eb8546d9c89e49819040000006b483045022100f056cfc363f139b1bf43b3be6190b28c459dc3ac1e52d71b144565779499142102206b23309ae0966f3dc9f8f59562a53e510531e98351545b48547878d0a78568f30121029bbfd642696c538e36bbb09e1221e45535e7a33d3387bb15304d72a7086c084dffffffff0236042003000000001976a914491020f6d1431203a75a291c4c432d27a27ea8be88ac5f2a1a01000000001976a9149a1c78a507689f6f54b847ad1cef1e614ee23f1e88ac00000000",
		tx: Tx{
			Version: 1,
			Ins: []*TxIn{
				{
					PreviousOutPoint: &OutPoint{
						Hash: &utils.Hash{
							0x8e, 0xf0, 0xa4, 0x66, 0x5b, 0x62, 0xf7, 0x4b, 0x24, 0xa2, 0x62,
							0x7c, 0xf6, 0x72, 0x20, 0x66, 0xf4, 0xaf, 0xe9, 0x7f, 0x62, 0x62,
							0x81, 0x2e, 0xb8, 0x54, 0x6d, 0x9c, 0x89, 0xe4, 0x98, 0x19,
						},
						Index: 4,
					},
					Script: &Script{
						bytes: []byte{
							0x48, 0x30, 0x45, 0x02, 0x21, 0x00, 0xf0, 0x56, 0xcf, 0xc3, 0x63, 0xf1, 0x39, 0xb1,
							0xbf, 0x43, 0xb3, 0xbe, 0x61, 0x90, 0xb2, 0x8c, 0x45, 0x9d, 0xc3, 0xac, 0x1e, 0x52,
							0xd7, 0x1b, 0x14, 0x45, 0x65, 0x77, 0x94, 0x99, 0x14, 0x21, 0x02, 0x20, 0x6b, 0x23,
							0x30, 0x9a, 0xe0, 0x96, 0x6f, 0x3d, 0xc9, 0xf8, 0xf5, 0x95, 0x62, 0xa5, 0x3e, 0x51,
							0x05, 0x31, 0xe9, 0x83, 0x51, 0x54, 0x5b, 0x48, 0x54, 0x78, 0x78, 0xd0, 0xa7, 0x85,
							0x68, 0xf3, 0x01, 0x21, 0x02, 0x9b, 0xbf, 0xd6, 0x42, 0x69, 0x6c, 0x53, 0x8e, 0x36,
							0xbb, 0xb0, 0x9e, 0x12, 0x21, 0xe4, 0x55, 0x35, 0xe7, 0xa3, 0x3d, 0x33, 0x87, 0xbb,
							0x15, 0x30, 0x4d, 0x72, 0xa7, 0x08, 0x6c, 0x08, 0x4d,
						},
					},
					Sequence: 0xffffffff,
				},
			},
			Outs: []*TxOut{
				{
					Value: 0x3200436,
					Script: &Script{
						bytes: []byte{
							OP_DUP, OP_HASH160,
							0x14,
							0x49, 0x10, 0x20, 0xf6, 0xd1, 0x43, 0x12, 0x03, 0xa7, 0x5a,
							0x29, 0x1c, 0x4c, 0x43, 0x2d, 0x27, 0xa2, 0x7e, 0xa8, 0xbe,
							OP_EQUALVERIFY,
							OP_CHECKSIG,
						},
					},
				},
				{
					Value: 0x11A2A5F,
					Script: &Script{
						bytes: []byte{
							OP_DUP,
							OP_HASH160,
							0x14,
							0x9a, 0x1c, 0x78, 0xa5, 0x07, 0x68, 0x9f, 0x6f, 0x54, 0xb8,
							0x47, 0xad, 0x1c, 0xef, 0x1e, 0x61, 0x4e, 0xe2, 0x3f, 0x1e,
							OP_EQUALVERIFY,
							OP_CHECKSIG,
						},
					},
				},
			},
		},
	},
	{
		txHash: "694a7c075e60985f4349b8f5a4f10b55d7412eddc87980d57ffac61329bd4352",
		txRaw:  "01000000010b7dbe4086617046a4b011319216d546c5224a7561a14173348c6035951d44d8010000006b4830450221009c3530e4cf7bdb194c3075818ffd88634c71635245c745e714490ce02c891cce022052c77f683ffe92b3dbecbf89207d331917b1ce9db07e55cec0748fc736676c06012103a34b99f22c790c4e36b2b3c2c35a36db06226e41c692fc82b8b56ac1c540c5bdffffffff014f031a01000000001976a914170a13c86ece92a9b902dbb5f367e361f227f08f88ac00000000",
		tx: Tx{
			Version: 1,
			Ins: []*TxIn{
				{
					PreviousOutPoint: &OutPoint{
						Hash: &utils.Hash{
							0x0b, 0x7d, 0xbe, 0x40, 0x86, 0x61, 0x70, 0x46, 0xa4, 0xb0,
							0x11, 0x31, 0x92, 0x16, 0xd5, 0x46, 0xc5, 0x22, 0x4a, 0x75,
							0x61, 0xa1, 0x41, 0x73, 0x34, 0x8c, 0x60, 0x35, 0x95, 0x1d,
							0x44, 0xd8,
						},
						Index: 1,
					},
					Script: &Script{
						bytes: []byte{
							0x48, 0x30, 0x45, 0x02, 0x21, 0x00, 0x9c, 0x35, 0x30, 0xe4, 0xcf, 0x7b, 0xdb, 0x19,
							0x4c, 0x30, 0x75, 0x81, 0x8f, 0xfd, 0x88, 0x63, 0x4c, 0x71, 0x63, 0x52, 0x45, 0xc7,
							0x45, 0xe7, 0x14, 0x49, 0x0c, 0xe0, 0x2c, 0x89, 0x1c, 0xce, 0x02, 0x20, 0x52, 0xc7,
							0x7f, 0x68, 0x3f, 0xfe, 0x92, 0xb3, 0xdb, 0xec, 0xbf, 0x89, 0x20, 0x7d, 0x33, 0x19,
							0x17, 0xb1, 0xce, 0x9d, 0xb0, 0x7e, 0x55, 0xce, 0xc0, 0x74, 0x8f, 0xc7, 0x36, 0x67,
							0x6c, 0x06, 0x01, 0x21, 0x03, 0xa3, 0x4b, 0x99, 0xf2, 0x2c, 0x79, 0x0c, 0x4e, 0x36,
							0xb2, 0xb3, 0xc2, 0xc3, 0x5a, 0x36, 0xdb, 0x06, 0x22, 0x6e, 0x41, 0xc6, 0x92, 0xfc,
							0x82, 0xb8, 0xb5, 0x6a, 0xc1, 0xc5, 0x40, 0xc5, 0xbd,
						},
					},
					Sequence: 0xffffffff,
				},
			},
			Outs: []*TxOut{
				{
					Value: 0x11A034F, //0.18481999
					Script: &Script{
						bytes: []byte{
							OP_DUP,
							OP_HASH160,
							0x14,
							0x17, 0x0a, 0x13, 0xc8, 0x6e, 0xce, 0x92, 0xa9, 0xb9, 0x02,
							0xdb, 0xb5, 0xf3, 0x67, 0xe3, 0x61, 0xf2, 0x27, 0xf0, 0x8f,
							OP_EQUALVERIFY,
							OP_CHECKSIG,
						},
					},
				},
			},
		},
	},
}

func TestParseOpCode(t *testing.T) {
	for _, test := range testsTx {
		tx := test.tx
		for _, out := range tx.Outs {
			stk, err := out.Script.ParseScript()
			if err != nil {
				t.Error(err)
			}
			if len(stk) != 5 {
				t.Errorf("parse opcode is error , count is %d", len(stk))
			}

		}
		for _, in := range tx.Ins {
			stk, err := in.Script.ParseScript()
			if err != nil {
				t.Error(err)
			}
			if len(stk) != 2 {
				t.Errorf("parse opcode is error , count is %d", len(stk))
			}

		}
	}
}

func TestInterpreterVerify(t *testing.T) {
	interpreter := Interpreter{
		stack: algorithm.NewStack(),
	}
	preTestTx := testsTx[0].tx
	testTx := testsTx[1].tx
	flag := core.SIGHASH_ALL
	ret, err := interpreter.Verify(&testTx, 0, testTx.Ins[0].Script, preTestTx.Outs[1].Script, uint32(flag))
	if err != nil {
		t.Error(err)
	}
	if !ret {
		t.Errorf("Tx Verify() fail")
	}
}

func TestMultiSigScript(t *testing.T) {
	//TxData from TxID : 8a2bc89d336cc3b6e285be55fa0e13ed35e929f35e20159753c1dace98ba1823
	txRawData := "0100000002c0b6ee52a5b2e8f73d772f9ec50be0d8010de4b07012bdbaff854351787420bd55000000fdfe0000483045022100d6148e16525140d56198c8421d958c4d084e8df94df58491b95b1d408f2eede40220691e189459cd36c99a5860785e53da5c579291386ce3e4ef740be7515396aea101483045022100ae47baeff7aa65c1205ac0d258067a678852c90b00986260cd0bdc1fcc11107c02207cbfd5cc4a8a46182ff2a2af8df4984805859d29c0bad526e3ed2fd45eb28d56014c695221036287850ad4184be9d5a6a26a9aba838455bce5f36c5a26a365c206e7216af27921022966bebc71211b843b8aa08a03f8bae874e1984332b60b41913559d72ff396342102a1e1483c3658b43016ad3b911a3fd439fd7d35b5b9e49b3c51dc5eec6447bc6653aeffffffff16dfd6c699124aff405e7729c21be6356eab2eb61cc11ce155ca88e08b03461308000000fdfd000047304402205eac2e77b0ab7c0439da5426c47a94070db0a48fc4b7d2b8ee6292ff26a4c91e02205ec162f1da86995cf2f64696b6c7209a8e81aef695b4dd60353bbe0f2143c3bb01483045022100ff5339e3cc1c0dc06f35aa1388781a9468541d9a414cc720447e57e1f513439c022032b5fea48fa0a3e82a537f4961e1839fc7cfe8c5f8c75196f3c5753cbf4b4715014c695221025638a131239abea7d79b5b8e96cf02acd963f39e0fad5d734693e86ceba7242a210281c3e2c15589043088ad3648def70c9ffbb72393d7fc39dbe8399c7f4a425bfd2103acfa87898ca1a09c542007902a59e87ee1e8d10bf52fe07e0d2ce16010a8c49353aeffffffff02ae4671480600000017a91436e7511762967e993f73254fb122bdb533b96aaa8754e7e100000000001976a914197c7d334338386469b25b7b1de06203189998eb88ac00000000"
	txRawDataByte, err := hex.DecodeString(txRawData)
	if err != nil {
		t.Error(err)
		return
	}
	buf := bytes.NewBuffer(nil)
	buf.Write(txRawDataByte)
	testTx, err := DeserializeTx(buf)
	if err != nil {
		t.Error(err)
		return
	}

	pubScript := "a914722ad96d2d4dac77162fb4027e0feb385bf06c0787"
	pubScriptByte, err := hex.DecodeString(pubScript)
	if err != nil {
		t.Error(err)
		return
	}
	prePubScript := NewScriptWithRaw(pubScriptByte)

	interpreter := Interpreter{
		stack: algorithm.NewStack(),
	}
	flag := OP_CHECKMULTISIG
	ret, err := interpreter.Verify(testTx, 0, testTx.Ins[0].Script, prePubScript, uint32(flag))
	if err != nil {
		t.Error(err)
	}
	if !ret {
		t.Errorf("Tx Verify() fail")
	}

}
