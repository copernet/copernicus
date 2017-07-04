package core

import (
	"github.com/jjz/secp256k1-go/secp256k1"
	"math/big"
)

type PrivateKey struct {
	PublicKey *PublicKey
	D         *big.Int
}

const (
	PrivateKeyBytesLen = 32
)

func PrivateKeyFromBytes(privateKeyBytes []byte) *PrivateKey {
	_, secp256k1PublicKey, err := secp256k1.EcPubkeyCreate(secp256k1Context, privateKeyBytes)
	if err != nil {
		return nil
	}
	privateKey := PrivateKey{
		PublicKey: (*PublicKey)(secp256k1PublicKey),
		D:         new(big.Int).SetBytes(privateKeyBytes),
	}
	return &privateKey
}

func (privateKey *PrivateKey) PubKey() *PublicKey {
	return privateKey.PublicKey
}

func (privateKey *PrivateKey) Sign(hash []byte) (*Signature, error) {
	_, signature, err := secp256k1.EcdsaSign(secp256k1Context, hash, privateKey.D.Bytes())
	return (*Signature)(signature), err
}

func (privateKey *PrivateKey) Serialize() []byte {
	b := make([]byte, 0, PrivateKeyBytesLen)
	return paddedAppend(PrivateKeyBytesLen, b, privateKey.D.Bytes())
}

func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
