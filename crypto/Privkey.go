package crypto

import (
	"github.com/btccom/secp256k1-go/secp256k1"
	"math/big"
)

type PrivateKey struct {
	PublicKey
	D *big.Int
}

const (
	PrivateKeyBytesLen = 32
)

func PrivateKeyFromBytes(privateKeyBytes []byte) (*PrivateKey, *PublicKey) {
	_, publicKey, err := secp256k1.EcPubkeyCreate(secp256k1Context, privateKeyBytes)
	if err != nil {
		return nil, nil
	}
	privateKey := PrivateKey{
		PublicKey: (PublicKey)(*publicKey),
		D:         new(big.Int).SetBytes(privateKeyBytes),
	}
	
	return &privateKey, &privateKey.PublicKey
}

func (p *PrivateKey) PubKey() *PublicKey {
	return &p.PublicKey
}

func (p *PrivateKey) Sign(hash []byte) (*Signature, error) {
	_, signature, err := secp256k1.EcdsaSign(secp256k1Context, hash, p.D.Bytes())
	return (*Signature)(signature), err
}

func (p *PrivateKey) Serialize() []byte {
	b := make([]byte, 0, PrivateKeyBytesLen)
	return paddedAppend(PrivateKeyBytesLen, b, p.D.Bytes())
}

func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
