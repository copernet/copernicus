package core

import (
	"github.com/jjz/secp256k1-go/secp256k1"
	"github.com/pkg/errors"
	"reflect"
)

var (
	errPublicKeySerialize = errors.New("secp256k1 public key serialize error")
)

type PublicKey secp256k1.PublicKey

func ParsePubKey(pubKeyStr []byte) (*PublicKey, error) {
	_, pubKey, err := secp256k1.EcPubkeyParse(secp256k1Context, pubKeyStr)
	return (*PublicKey)(pubKey), err
}

func (publicKey *PublicKey) ToSecp256k() *secp256k1.PublicKey {
	return (*secp256k1.PublicKey)(publicKey)
}

func (publicKey *PublicKey) SerializeUncompressed() []byte {
	_, serializedComp, err := secp256k1.EcPubkeySerialize(secp256k1Context, publicKey.ToSecp256k(), secp256k1.EcUncompressed)
	if err != nil {
		panic(errPublicKeySerialize)
	}
	return serializedComp
}

func (publicKey *PublicKey) SerializeCompressed() []byte {
	_, serializedComp, err := secp256k1.EcPubkeySerialize(secp256k1Context, publicKey.ToSecp256k(), secp256k1.EcCompressed)
	if err != nil {
		panic(errPublicKeySerialize)
	}
	return serializedComp
}

func (publicKey *PublicKey) IsEqual(otherPublicKey *PublicKey) bool {
	publicKeyBytes := publicKey.SerializeUncompressed()
	otherBytes := otherPublicKey.SerializeUncompressed()
	return reflect.DeepEqual(publicKeyBytes, otherBytes)
}
