package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestPrivKeys(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{
			name: "check curve",
			key: []byte{
				0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
				0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
				0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
				0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
			},
		},
	}

	for _, test := range tests {
		privateKey := PrivateKeyFromBytes(test.key)

		_, err := ParsePubKey(privateKey.PubKey().SerializeUncompressed())
		if err != nil {
			t.Errorf("%s privkey: %v", test.name, err)
			continue
		}

		hash := [32]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9}
		sig, err := privateKey.Sign(hash[:])
		if err != nil {
			t.Errorf("%s could not sign: %v", test.name, err)
			continue
		}

		if !sig.Verify(hash[:], privateKey.PubKey()) {
			t.Errorf("%s could not verify: %v", test.name, err)
			continue
		}

		serializedKey := privateKey.Encode()
		if !bytes.Equal(serializedKey, test.key) {
			t.Errorf("%s unexpected serialized bytes - got: %x, "+
				"want: %x", test.name, serializedKey, test.key)
		}
	}
}

func TestEncodePrivateKey(t *testing.T) {
	originalKey, err := DecodePrivateKey("5KYZdUEo39z3FPrtuX2QbbwGnNP5zTd7yyr2SC1j299sBCnWjss")
	if err != nil {
		t.Error(err)
	}
	if originalKey.compressed {
		t.Errorf("5KYZdUEo39z3FPrtuX2QbbwGnNP5zTd7yyr2SC1j299sBCnWjss is UnCompreeed key")
	}
	if "5KYZdUEo39z3FPrtuX2QbbwGnNP5zTd7yyr2SC1j299sBCnWjss" != originalKey.ToString() {
		t.Errorf("private key toString is err  : %s", originalKey.ToString())
	}
	privKeyHex := hex.EncodeToString(originalKey.bytes)
	if privKeyHex != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("hex(%s) of private key is error", privKeyHex)
	}
	pubKey := originalKey.PubKey()
	if pubKey.ToHexString() !=
		"04a34b99f22c790c4e36b2b3c2c35a36db06226e41c692fc82b8b56ac1c540c5bd5b8dec5235a0fa8722476c7709c02559e3aa73aa03918ba2d492eea75abea235" {
		t.Errorf("public key hex is error(%s)", pubKey.ToHexString())
	}

	originalKey.compressed = true
	if "L4rK1yDtCWekvXuE6oXD9jCYfFNV2cWRpVuPLBcCU2z8TrisoyY1" != originalKey.ToString() {
		t.Errorf("private key toString is err  : %s", originalKey.ToString())
	}
	privKeyHex = hex.EncodeToString(originalKey.bytes)
	if privKeyHex != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("hex(%s) of private key is error", privKeyHex)
	}
	if !originalKey.compressed {
		t.Errorf("L4rK1yDtCWekvXuE6oXD9jCYfFNV2cWRpVuPLBcCU2z8TrisoyY1 is Compreeed key")
	}
	pubKey = originalKey.PubKey()
	if pubKey.ToHexString() !=
		"03a34b99f22c790c4e36b2b3c2c35a36db06226e41c692fc82b8b56ac1c540c5bd" {
		t.Errorf("public key hex is error(%s)", pubKey.ToHexString())
	}

}

func TestInitPrivateKeyVersion(t *testing.T) {
	InitPrivateKeyVersion(39)
	privateKey := PrivateKeyFromBytes([]byte{})
	if privateKey.version != 39 {
		t.Errorf("TestInitAddressParam test failed, privateKeyVer(%v) not init", privateKey.version)
	}
	InitPrivateKeyVersion(DumpedPrivateKeyVersion)
	privateKey = PrivateKeyFromBytes([]byte{})
	if privateKey.version != DumpedPrivateKeyVersion {
		t.Errorf("TestInitAddressParam test failed, privateKeyVer(%v) not init", privateKey.version)
	}
}
