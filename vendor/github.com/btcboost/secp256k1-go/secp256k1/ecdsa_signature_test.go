package secp256k1

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSignatureParseDerFixtures(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	fixtures := GetEcdsaFixtures()

	for i := 0; i < len(fixtures); i++ {
		sigBytes := fixtures[i].GetSigBytes()
		r, sig, err := EcdsaSignatureParseDer(ctx, sigBytes)
		spOK(t, r, err)

		r, serialized, err := EcdsaSignatureSerializeDer(ctx, sig)
		spOK(t, r, err)

		assert.Equal(t, sigBytes, serialized)
	}
}

func TestSignatureParseCompactFixtures(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	fixtures := GetEcdsaFixtures()

	for i := 0; i < len(fixtures); i++ {
		sigBytes := fixtures[i].GetSigBytes()
		r, sig, err := EcdsaSignatureParseDer(ctx, sigBytes)
		spOK(t, r, err)

		r, serialized, err := EcdsaSignatureSerializeDer(ctx, sig)
		spOK(t, r, err)

		assert.Equal(t, sigBytes, serialized)
	}
}

func TestParseCompactRequires64Bytes(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	bad := []byte(`a`)
	r, sig, err := EcdsaSignatureParseCompact(ctx, bad)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Nil(t, sig)
	assert.Equal(t, ErrorCompactSigSize, err.Error())
}

func Test_EcdsaSignatureParseCompact(t *testing.T) {
	ctx, err := ContextCreate(uint(ContextSign | ContextVerify))
	if err != nil {
		panic(err)
	}

	str := "fe5fe404f3d8c21e1204a08c38ff3912d43c5a22541d2f1cdc4977cbcad240015a3b6e9040f62cacf016df4fef9412091592e4908e5e3a7bd2a42a4d1be01951"
	sigByte, err := hex.DecodeString(str)

	s, sig, err := EcdsaSignatureParseCompact(ctx, sigByte)
	if err != nil {
		panic(err)
	}

	assert.IsType(t, EcdsaSignature{}, *sig)
	assert.Equal(t, 1, s)
	assert.NoError(t, err)

	s, out, err := EcdsaSignatureSerializeCompact(ctx, sig)
	assert.Equal(t, 1, s)
	assert.NoError(t, err)
	assert.Equal(t, str, hex.EncodeToString(out))
}

func TestParseCompactMustBeValid(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	bad, err := hex.DecodeString(`FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364142`)
	if err != nil {
		panic(err)
	}

	r, sig, err := EcdsaSignatureParseCompact(ctx, bad)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Nil(t, sig)
	assert.Equal(t, ErrorCompactSigParse, err.Error())
}

func TestParseDerMustBeValid(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	bad, err := hex.DecodeString(`30440220132382ca59240c2e14ee7ff61d90fc63276325f4cbe8169fc53ade4a407c2fc802204d86fbe3`)
	if err != nil {
		panic(err)
	}

	r, sig, err := EcdsaSignatureParseDer(ctx, bad)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Nil(t, sig)
	assert.Equal(t, ErrorDerSigParse, err.Error())
}

func TestSignRequiresProperMsg32(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	msg32 := []byte(`abcd`)
	priv, _ := hex.DecodeString(`abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234`)

	r, sig, err := EcdsaSign(ctx, msg32, priv)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Nil(t, sig)
	assert.Equal(t, ErrorMsg32Size, err.Error())
}

func TestSignRequiresProperPrivateKey(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	priv := []byte(`abcd`)
	msg32, _ := hex.DecodeString(`abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234`)

	r, sig, err := EcdsaSign(ctx, msg32, priv)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Nil(t, sig)
	assert.Equal(t, ErrorPrivateKeySize, err.Error())
}

func TestSignReturnsAnError(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	priv, _ := hex.DecodeString(`FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364142`)
	msg32, _ := hex.DecodeString(`FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141`)

	r, sig, err := EcdsaSign(ctx, msg32, priv)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Nil(t, sig)
	assert.Equal(t, ErrorProducingSignature, err.Error())
}
