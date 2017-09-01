package model

import (
	"encoding/hex"

	"testing"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

func TestCheckSig(t *testing.T) {
	//The rawTx form blockChain b5f8da1ea9e02ec3cc0765f9600f49945e94ed4b0c88ed0648896bf3e213205d
	rawTxExceptSigStr := "010000000156211389e5410a9fd1fc684ea3a852b8cee07fd15398689d99441b98bfa76e290000000000ffffffff0280969800000000001976a914fdc7990956642433ea75cabdcc0a9447c5d2b4ee88acd0e89600000000001976a914d6c492056f3f99692b56967a42b8ad44ce76b67a88ac00000000"
	rawTxExceptSig, err := hex.DecodeString(rawTxExceptSigStr)
	if err != nil {
		t.Error(err)
	}
	hashTmp := core.DoubleSha256Bytes(rawTxExceptSig)
	var hash utils.Hash
	copy(hash[:], hashTmp)

	publicKeyStr := "0477c075474b6798c6e2254d3d06c1ae3b91318ca5cc62d18398697208549f798e28efb6c55971a1de68cca81215dd53686c31ad8155cdc03563bf3f73ce87b4aa"
	vchPubKey, err := hex.DecodeString(publicKeyStr)
	if err != nil {
		t.Error(err)
	}

	scriptSigStr := "3046022100f9da4f53a6a4a8317f6e7e9cd9a7b76e0f5e95dcdf70f1b1e2b3548eaa3a6975022100858d48aed79da8873e09b0e41691f7f3e518ce9a88ea3d03f7b32eb818f60688"
	vchSig, err := hex.DecodeString(scriptSigStr)
	if err != nil {
		t.Error(err)
	}

	/*
		hashStr := "b5f8da1ea9e02ec3cc0765f9600f49945e94ed4b0c88ed0648896bf3e213205d"
		bytes, err := hex.DecodeString(hashStr)
		if err != nil {
			t.Error(err)
		}
		copy(hash[:], bytes)
	*/

	_, err = CheckSig(&hash, vchSig, vchPubKey)

}

func TestCheckSequence(t *testing.T) {
	pkBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2d4f8720ee63e502ee2869afab7de234b80c")
	if err != nil {
		t.Error(err)
	}
	privateKey := core.PrivateKeyFromBytes(pkBytes)
	if privateKey == nil {
		t.Error("create private fail")
		return
	}

	message := []byte{1, 2, 3, 4, 5, 6, 7}
	messageHash := core.DoubleSha256Bytes(message)
	signature, err := privateKey.Sign(messageHash)
	if err != nil {
		t.Error(err)
	}

	ret := signature.Verify(messageHash, privateKey.PublicKey)
	if !ret {
		t.Error("verify fail")
	}

}
