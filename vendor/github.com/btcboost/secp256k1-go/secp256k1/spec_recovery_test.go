package secp256k1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSpecRecoverableSignature(t *testing.T) {
	context, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	msg32 := testingRand(32)
	priv := testingRand(32)

	r, sig, err := EcdsaSignRecoverable(context, msg32, priv)
	spOK(t, r, err)

	// Converted signature can be verified against pubkey

	r, plain, err := EcdsaRecoverableSignatureConvert(context, sig)
	spOK(t, r, err)

	r, pub, err := EcPubkeyCreate(context, priv)
	spOK(t, r, err)

	r, err = EcdsaVerify(context, plain, msg32, pub)
	spOK(t, r, err)

	// Can serialize recoverable signature

	r, sig64, recid, err := EcdsaRecoverableSignatureSerializeCompact(context, sig)
	spOK(t, r, err)

	assert.NotEmpty(t, sig64)

	r, sigParsed, err := EcdsaRecoverableSignatureParseCompact(context, sig64, recid)
	spOK(t, r, err)

	assert.Equal(t, sig, sigParsed)

	// Recovers correct public key

	r, pubkeyRec, err := EcdsaRecover(context, sig, msg32)
	spOK(t, r, err)

	assert.Equal(t, pub, pubkeyRec)
}
