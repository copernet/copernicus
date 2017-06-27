package crypto

import (
	"github.com/btccom/copernicus/secp256k1/secp256k1"
)

type Signature secp256k1.EcdsaSignature

func (sig * Signature) toLibEcdsaSignature() *secp256k1.EcdsaSignature{
	return (*secp256k1.EcdsaSignature)(sig)
}

func (sig * Signature) Serialize() []byte {
	_, serializedSig, _ := secp256k1.EcdsaSignatureSerializeDer(secp256k1Context, sig.toLibEcdsaSignature())
	return serializedSig
}

func (sig * Signature) Verify(hash []byte, pubKey *PublicKey) bool {
	correct, _ := secp256k1.EcdsaVerify(secp256k1Context, sig.toLibEcdsaSignature(),
		hash, (*secp256k1.PublicKey)(pubKey))
	return correct == 1
}

func ParseDERSignature(signature []byte) (*Signature, error) {
	_, sig, err :=  secp256k1.EcdsaSignatureParseDer(secp256k1Context, signature)
	if err != nil {
		return nil, err
	}
	return (*Signature)(sig), nil
}

func ParseSignature(signature []byte) (*Signature, error) {
	_, sig, err :=  secp256k1.EcdsaSignatureParseCompact(secp256k1Context, signature)
	if err != nil {
		return nil, err
	}
	return (*Signature)(sig), nil
}

func Sign(privKey *PrivateKey, hash []byte) ([]byte, error){
	_, signature, err :=  secp256k1.EcdsaSign(secp256k1Context, hash, privKey.D.Bytes())
	if err != nil{
		return nil, err
	}
	_, ret, _ := secp256k1.EcdsaSignatureSerializeDer(secp256k1Context, signature)
	return ret, nil
}
