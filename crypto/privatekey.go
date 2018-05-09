package crypto

import (
	"fmt"

	"github.com/btcboost/copernicus/util/base58"
	"github.com/btcboost/secp256k1-go/secp256k1"
	"github.com/pkg/errors"
)

type PrivateKey struct {
	version    byte
	compressed bool
	bytes      []byte
}

const (
	PrivateKeyBytesLen      = 32
	DumpedPrivateKeyVersion = 128
)

func PrivateKeyFromBytes(privateKeyBytes []byte) *PrivateKey {

	privateKey := PrivateKey{
		//D:         new(big.Int).SetBytes(privateKeyBytes),
		bytes:   privateKeyBytes,
		version: DumpedPrivateKeyVersion,
	}
	return &privateKey
}

func (privateKey *PrivateKey) PubKey() *PublicKey {
	_, secp256k1PublicKey, err := secp256k1.EcPubkeyCreate(secp256k1Context, privateKey.bytes)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	publicKey := PublicKey{SecpPubKey: secp256k1PublicKey, Compressed: privateKey.compressed}
	return &publicKey
}

func (privateKey *PrivateKey) Sign(hash []byte) (*Signature, error) {
	_, signature, err := secp256k1.EcdsaSign(secp256k1Context, hash, privateKey.bytes)
	return (*Signature)(signature), err
}

func (privateKey *PrivateKey) Encode() []byte {

	if !privateKey.compressed {
		return privateKey.bytes
	}
	bytes := make([]byte, 0)
	bytes = append(bytes, privateKey.bytes...)
	bytes = append(bytes, 1)
	return bytes

}

func (privateKey *PrivateKey) ToString() string {

	privateKeyBytes := privateKey.Encode()

	privateKeyString := base58.CheckEncode(privateKeyBytes, privateKey.version)
	return privateKeyString

}

func DecodePrivateKey(encoded string) (*PrivateKey, error) {
	bytes, version, err := base58.CheckDecode(encoded)
	if err != nil {
		return nil, err
	}
	if version != DumpedPrivateKeyVersion {
		return nil, errors.Errorf("Mismatched version number ,trying to cross network , got version is %d", version)
	}
	var compressed bool
	if len(bytes) == PrivateKeyBytesLen+1 && bytes[PrivateKeyBytesLen] == 1 {
		compressed = true
		bytes = bytes[:PrivateKeyBytesLen]
	} else if len(bytes) == PrivateKeyBytesLen {
		compressed = false

	} else {
		return nil, errors.New("Wrong number of bytes a private key , not 32 or 33")
	}
	privateKey := PrivateKey{version: version, bytes: bytes, compressed: compressed}
	return &privateKey, nil

}
