package model

import (
	"encoding/hex"

	"testing"

	"github.com/btcboost/copernicus/core"
)

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

	ret, err := CheckSig(hash, vchSign, privateKey.PublicKey.SerializeCompressed())
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
