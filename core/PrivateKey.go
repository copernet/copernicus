package core

import (
	"btcutil/base58"

	"github.com/btcboost/secp256k1-go/secp256k1"
	"github.com/pkg/errors"
)

type PrivateKey struct {
	version    int
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
		return nil
	}
	publicKey := PublicKey{SecpPubKey: secp256k1PublicKey, Compressed: privateKey.compressed}
	return &publicKey
}

func (privateKey *PrivateKey) Sign(hash []byte) (*Signature, error) {
	_, signature, err := secp256k1.EcdsaSign(secp256k1Context, hash, privateKey.bytes)
	return (*Signature)(signature), err
}

func (privateKey *PrivateKey) Serialize() []byte {
	b := make([]byte, 0, PrivateKeyBytesLen)
	return paddedAppend(PrivateKeyBytesLen, b, privateKey.bytes)
}

func (privateKey *PrivateKey) ToString() string {
	addressBytes := make([]byte, 1+PrivateKeyBytesLen+4)
	addressBytes[0] = byte(privateKey.version)
	privateKeyBytes := privateKey.Serialize()
	copy(addressBytes[1:], privateKeyBytes[:])
	check := DoubleSha256Bytes(privateKeyBytes)
	copy(addressBytes[1+PrivateKeyBytesLen:], check[:4])
	return base58.Encode(addressBytes)

}

func DecodePrivatKey(encoded string) (*PrivateKey, error) {
	bytes, version, err := base58.CheckDecode(encoded)
	if err != nil {
		return nil, err
	}
	if version != DumpedPrivateKeyVersion {
		return nil, errors.Errorf("Mismatched version number ,trying to cross network , got version is %d", version)
	}
	var compressed bool
	if len(bytes) == PrivateKeyBytesLen+1 && bytes[0] == 1 {
		compressed = true
		bytes = bytes[1:]
	} else if len(bytes) == PrivateKeyBytesLen {
		compressed = false

	} else {
		return nil, errors.New("Wrong number of bytes a private key , not 32 or 33")
	}
	privatetKey := PrivateKey{version: int(version), bytes: bytes, compressed: compressed}
	return &privatetKey, nil

}

func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
