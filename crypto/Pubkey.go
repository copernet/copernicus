package crypto

import (
	"github.com/pkg/errors"
	"github.com/btccom/secp256k1-go/secp256k1"
)

var (
	errPublicKeySerialize = errors.New("spcp256k1 public key serialize error")
)

func ParsePubKey(pubKeyStr []byte) (*PublicKey, error) {
	_, pubKey, err := secp256k1.EcPubkeyParse(secp256k1Context, pubKeyStr)
	return (*PublicKey)(pubKey), err
}

type PublicKey secp256k1.PublicKey

func (p *PublicKey) ToSecp256k() *secp256k1.PublicKey {
	return p.ToSecp256k()
}

func (pubKey *PublicKey) SerializeUncompressed() []byte {
	_, serializedComp, err := secp256k1.EcPubkeySerialize(secp256k1Context, pubKey.ToSecp256k(), secp256k1.EcUncompressed)
	if err != nil {
		panic(errPublicKeySerialize)
	}
	return serializedComp
}

func (pubKey *PublicKey) SerializeCompressed() []byte {
	_, serializedComp, err := secp256k1.EcPubkeySerialize(secp256k1Context, pubKey.ToSecp256k(), secp256k1.EcCompressed)
	if err != nil {
		panic(errPublicKeySerialize)
	}
	return serializedComp
}
