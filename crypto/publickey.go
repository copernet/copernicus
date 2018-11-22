package crypto

import (
	"encoding/hex"
	"errors"
	"reflect"

	"github.com/copernet/copernicus/util"
	"github.com/copernet/secp256k1-go/secp256k1"
)

var (
	errPublicKeySerialize = errors.New("secp256k1 public key serialize error")
)

type PublicKey struct {
	SecpPubKey *secp256k1.PublicKey
	Compressed bool
}

func ParsePubKey(pubKeyStr []byte) (*PublicKey, error) {
	_, pubKey, err := secp256k1.EcPubkeyParse(secp256k1Context, pubKeyStr)
	publicKey := PublicKey{SecpPubKey: pubKey, Compressed: IsCompressedPubKey(pubKeyStr)}
	return &publicKey, err
}

func (publicKey *PublicKey) ToSecp256k() *secp256k1.PublicKey {
	return publicKey.SecpPubKey
}

func (publicKey *PublicKey) ToHexString() string {
	bytes := publicKey.ToBytes()
	return hex.EncodeToString(bytes)
}

func (publicKey *PublicKey) ToHash160() []byte {
	bytes := publicKey.ToBytes()
	return util.Hash160(bytes)
}

func (publicKey *PublicKey) ToBytes() []byte {
	if publicKey.Compressed {
		return publicKey.SerializeCompressed()
	}
	return publicKey.SerializeUncompressed()
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

func (publicKey *PublicKey) Verify(hash *util.Hash, vchSig []byte) (bool, error) {
	if !publicKey.isValid() {
		return false, nil
	}

	_, pubKey, err := secp256k1.EcPubkeyParse(secp256k1Context, publicKey.ToBytes())
	if err != nil {
		return false, nil
	}

	if len(vchSig) == 0 {
		return false, nil
	}

	_, ecdsaSignature, err := secp256k1.EcdsaSignatureParseDerLax(secp256k1Context, vchSig)
	if err != nil {
		return false, nil
	}

	secp256k1.EcdsaSignatureNormalize(secp256k1Context, ecdsaSignature, ecdsaSignature)

	result, err := secp256k1.EcdsaVerify(secp256k1Context, ecdsaSignature, hash.GetCloneBytes(), pubKey)

	if result == 0 {
		return false, nil
	}

	if err != nil {
		return false, nil
	}

	return true, nil
}

func (publicKey *PublicKey) isValid() bool {
	header := publicKey.ToBytes()[0]
	return getLen(header) > 0
}

func getLen(header uint8) uint {
	if header == 2 || header == 3 {
		return 33
	}

	if header == 4 || header == 6 {
		return 65
	}

	return 0
}

func IsCompressedOrUncompressedPubKey(bytes []byte) bool {
	if len(bytes) < 33 {
		return false
	}
	if bytes[0] == 0x04 {
		if len(bytes) != 65 {
			return false
		}
	} else if bytes[0] == 0x02 || bytes[0] == 0x03 {
		if len(bytes) != 33 {
			return false
		}
	} else {
		return false
	}
	return true
}

func IsCompressedPubKey(bytes []byte) bool {
	if len(bytes) != 33 {
		return false
	}
	if bytes[0] != 0x02 && bytes[0] != 0x03 {
		return false
	}
	return true
}
