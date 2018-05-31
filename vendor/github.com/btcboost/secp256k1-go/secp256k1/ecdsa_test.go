package secp256k1

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

type EcdsaTestCase struct {
	PrivateKey string `yaml:"privkey"`
	Message    string `yaml:"msg"`
	Sig        string `yaml:"sig"`
}

func (t *EcdsaTestCase) GetPrivateKey() []byte {
	private, err := hex.DecodeString(t.PrivateKey)
	if err != nil {
		panic("Invalid private key")
	}
	return private
}
func (t *EcdsaTestCase) GetPublicKey(ctx *Context) *PublicKey {
	private := t.GetPrivateKey()
	_, pk, err := EcPubkeyCreate(ctx, private)
	if err != nil {
		panic(err)
	}
	return pk
}

func (t *EcdsaTestCase) GetMessage() []byte {
	msg, err := hex.DecodeString(t.Message)
	if err != nil {
		panic("Invalid msg32")
	}
	return msg
}
func (t *EcdsaTestCase) GetSigBytes() []byte {
	sig, err := hex.DecodeString(removeSigHash(t.Sig))
	if err != nil {
		panic("Invalid msg32")
	}
	return sig
}
func (t *EcdsaTestCase) GetSig(ctx *Context) *EcdsaSignature {
	sigb := t.GetSigBytes()
	_, sig, err := EcdsaSignatureParseDer(ctx, sigb)
	if err != nil {
		panic(err)
	}
	return sig
}

type EcdsaFixtures []EcdsaTestCase

func GetEcdsaFixtures() []EcdsaTestCase {
	source := readFile(EcdsaTestVectors)
	testCase := EcdsaFixtures{}
	err := yaml.Unmarshal(source, &testCase)
	if err != nil {
		panic(err)
	}
	return testCase
}

func Test_Ecdsa_InvalidSig(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	sig := newEcdsaSignature()
	pk := newPublicKey()
	r, err := EcdsaVerify(ctx, sig, []byte{}, pk)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
}

func Test_Ecdsa_Verify(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	fixtures := GetEcdsaFixtures()

	for i := 0; i < len(fixtures); i++ {
		testCase := fixtures[i]
		msg32 := testCase.GetMessage()
		pubkey := testCase.GetPublicKey(ctx)
		sig := testCase.GetSig(ctx)

		result, err := EcdsaVerify(ctx, sig, msg32, pubkey)
		spOK(t, result, err)
	}
}

func Test_Ecdsa_Sign(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	fixtures := GetEcdsaFixtures()
	for i := 0; i < len(fixtures); i++ {
		testCase := fixtures[i]
		msg32 := testCase.GetMessage()
		priv := testCase.GetPrivateKey()
		sigb := testCase.GetSigBytes()

		r, sig, err := EcdsaSign(ctx, msg32, priv)

		spOK(t, r, err)

		r, serialized, err := EcdsaSignatureSerializeDer(ctx, sig)
		assert.Equal(t, sigb, serialized)
	}
}
