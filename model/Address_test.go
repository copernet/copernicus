package model

import (
	"encoding/hex"
	"testing"
)

func TestPublicKeyToAddress(t *testing.T) {
	publicKey := "03a34b99f22c790c4e36b2b3c2c35a36db06226e41c692fc82b8b56ac1c540c5bd"
	bytes, err := hex.DecodeString(publicKey)
	if err != nil {
		t.Fatal(err)
		return
	}
	address, err := AddressFromPublicKey(bytes, AddressVerPubKey(false))
	if err != nil {
		t.Fatal(err)
		return
	}
	hash160 := make([]byte, 20)
	copy(hash160[:], address.hash160[:])
	hash160Hex := hex.EncodeToString(hash160)
	if hash160Hex != "9a1c78a507689f6f54b847ad1cef1e614ee23f1e" {
		t.Errorf("hash160Hex is wrong 9a1c78a507689f6f54b847ad1cef1e614ee23f1e  --  %s", hash160Hex)
		return
	}
	if address.addressStr != "1F3sAm6ZtwLAUnj7d38pGFxtP3RVEvtsbV" {
		t.Errorf("address is wrong 1F3sAm6ZtwLAUnj7d38pGFxtP3RVEvtsbV  --  %s", address.addressStr)
		return
	}
}
