package model

import (
	"encoding/hex"

	"testing"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

var testTxs = []struct {
	txHash string
	txRaw  string
	tx     Tx
}{
	{
		txHash: "0c0b35173995b41171f252f175c40fbe0716887629ff8504427a69c348562b37",
		txRaw:  "0100000001d59d34c1b2eb98b7107a50bdfefdb0c0d5a61effd27196e7299c82c4c0aea791000000006a47304402206d002de6f70106ab1575af58e51b3eadc57c9415dcdc03afe79b842a11453ed4022003f64e670bba6b96ead2d2f9887006b2d017d94a12a1de6519c41e1705cb6003012103a34b99f22c790c4e36b2b3c2c35a36db06226e41c692fc82b8b56ac1c540c5bdffffffff010c40000000000000232103db3c3977c5165058bf38c46f72d32f4e872112dbafc13083a948676165cd1603ac00000000",
		tx: Tx{
			Version: 1,
			Ins: []*TxIn{
				{
					PreviousOutPoint: &OutPoint{
						Hash: &utils.Hash{
							0x03, 0x2e, 0x38, 0xe9, 0xc0, 0xa8, 0x4c, 0x60,
							0x46, 0xd6, 0x87, 0xd1, 0x05, 0x56, 0xdc, 0xac,
							0xc4, 0x1d, 0x27, 0x5e, 0xc5, 0x5f, 0xc0, 0x07,
							0x79, 0xac, 0x88, 0xfd, 0xf3, 0x57, 0xa1, 0x87,
						},
						Index: 0,
					},
					Script: &Script{
						bytes: []byte{
							0x49, // OP_DATA_73
							0x30, 0x46, 0x02, 0x21, 0x00, 0xc3, 0x52, 0xd3,
							0xdd, 0x99, 0x3a, 0x98, 0x1b, 0xeb, 0xa4, 0xa6,
							0x3a, 0xd1, 0x5c, 0x20, 0x92, 0x75, 0xca, 0x94,
							0x70, 0xab, 0xfc, 0xd5, 0x7d, 0xa9, 0x3b, 0x58,
							0xe4, 0xeb, 0x5d, 0xce, 0x82, 0x02, 0x21, 0x00,
							0x84, 0x07, 0x92, 0xbc, 0x1f, 0x45, 0x60, 0x62,
							0x81, 0x9f, 0x15, 0xd3, 0x3e, 0xe7, 0x05, 0x5c,
							0xf7, 0xb5, 0xee, 0x1a, 0xf1, 0xeb, 0xcc, 0x60,
							0x28, 0xd9, 0xcd, 0xb1, 0xc3, 0xaf, 0x77, 0x48,
							0x01, // 73-byte signature
							0x41, // OP_DATA_65
							0x04, 0xf4, 0x6d, 0xb5, 0xe9, 0xd6, 0x1a, 0x9d,
							0xc2, 0x7b, 0x8d, 0x64, 0xad, 0x23, 0xe7, 0x38,
							0x3a, 0x4e, 0x6c, 0xa1, 0x64, 0x59, 0x3c, 0x25,
							0x27, 0xc0, 0x38, 0xc0, 0x85, 0x7e, 0xb6, 0x7e,
							0xe8, 0xe8, 0x25, 0xdc, 0xa6, 0x50, 0x46, 0xb8,
							0x2c, 0x93, 0x31, 0x58, 0x6c, 0x82, 0xe0, 0xfd,
							0x1f, 0x63, 0x3f, 0x25, 0xf8, 0x7c, 0x16, 0x1b,
							0xc6, 0xf8, 0xa6, 0x30, 0x12, 0x1d, 0xf2, 0xb3,
							0xd3, // 65-byte pubkey

						},
					},
					Sequence: 0xffffffff,
				},
			},
			Outs: []*TxOut{
				{
					Value: 0x2123e300, // 556000000
					Script: &Script{
						bytes: []byte{
							0x76, // OP_DUP
							0xa9, // OP_HASH160
							0x14, // OP_DATA_20
							0xc3, 0x98, 0xef, 0xa9, 0xc3, 0x92, 0xba, 0x60,
							0x13, 0xc5, 0xe0, 0x4e, 0xe7, 0x29, 0x75, 0x5e,
							0xf7, 0xf5, 0x8b, 0x32,
							0x88, // OP_EQUALVERIFY
							0xac, // OP_CHECKSIG
						},
					},
				},
				{
					Value: 0x108e20f00, // 4444000000
					Script: &Script{
						bytes: []byte{
							0x76, // OP_DUP
							0xa9, // OP_HASH160
							0x14, // OP_DATA_20
							0x94, 0x8c, 0x76, 0x5a, 0x69, 0x14, 0xd4, 0x3f,
							0x2a, 0x7a, 0xc1, 0x77, 0xda, 0x2c, 0x2f, 0x6b,
							0x52, 0xde, 0x3d, 0x7c,
							0x88, // OP_EQUALVERIFY
							0xac, // OP_CHECKSIG
						},
					},
				},
			},
		},
	},
}

func TestCheckSequence(t *testing.T) {
	pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f8720ee63e502ee2869afab7de234b80c")
	if err != nil {
		t.Error(err)
	}
	privateKey := core.PrivateKeyFromBytes(pkBytes)
	if privateKey == nil {
		t.Error("create privateKey fail")
		return
	}

	message := []byte{1, 2, 3, 4, 5, 6, 7}
	messageHash := core.DoubleSha256Bytes(message)
	signature, err := privateKey.Sign(messageHash)
	if err != nil {
		t.Error(err)
	}
	vchSign := signature.Serialize()

	hash := core.DoubleSha256Hash(message)

	ret, err := CheckSig(hash, vchSign, privateKey.PubKey().SerializeCompressed())
	if err != nil {
		t.Error(err)
	} else {
		if !ret {
			t.Error("Error CheckSig() fail")
		}
	}

}

func TestSignHash(t *testing.T) {

	//pkBytes, err := base58.CheckDecode("22a47fa09a223f2aa079edf85a7c2d4f8720ee63e502ee2869afab7de234b80c")
	//if err != nil {
	//	t.Error(err)
	//}
	//privateKey := core.PrivateKe(pkBytes)
	//preTx := new(Tx)
	//buf := new(bytes.Buffer)
	//buf.WriteString("0100000001eda796b7f4f6d451cec6faa95723f8f9f70f6c67e30e839aa1b31dd28e67c327000000006b48304502210099e090227d9e1064a5a5846b2bd9083d971631b76e40bcae3e7fa65d92e29f0502203d8f8e957136112375093615d861fe900ddad74b3e008e1eeefb8010f3feb865012102254c17974d904271baa6cddbdcc3698d5e3b2a2165b76963630b2e6fd10397d2ffffffff02c44b0000000000001976a9149a1c78a507689f6f54b847ad1cef1e614ee23f1e88ac70cc0100000000001976a91466fd5a8fa1126720a4c9f4d8bd7b02e616f279e688ac00000000")
	//preTx.Deserialize(buf)
	//
	//tx := new(Tx)
	//txBuf := new(bytes.Buffer)
	//txBuf.WriteString("0100000001d59d34c1b2eb98b7107a50bdfefdb0c0d5a61effd27196e7299c82c4c0aea791000000006a47304402206d002de6f70106ab1575af58e51b3eadc57c9415dcdc03afe79b842a11453ed4022003f64e670bba6b96ead2d2f9887006b2d017d94a12a1de6519c41e1705cb6003012103a34b99f22c790c4e36b2b3c2c35a36db06226e41c692fc82b8b56ac1c540c5bdffffffff010c40000000000000232103db3c3977c5165058bf38c46f72d32f4e872112dbafc13083a948676165cd1603ac00000000")

}
