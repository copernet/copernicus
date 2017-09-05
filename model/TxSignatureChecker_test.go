package model

import (
	"encoding/hex"

	"testing"

	"github.com/btcboost/copernicus/core"
)

func TestCheckSig(t *testing.T) {
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

	ret, err := CheckSig(&hash, vchSign, privateKey.PublicKey.SerializeCompressed())
	if err != nil {
		t.Error(err)
	} else {
		if !ret {
			t.Error("Error CheckSig() fail")
		}
	}

}
