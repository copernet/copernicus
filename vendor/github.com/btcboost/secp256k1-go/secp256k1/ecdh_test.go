package secp256k1

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEcdhCatchesOverflow(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	alice, _ := hex.DecodeString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364142")
	bob := []byte{0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40}

	r, Bob, err := EcPubkeyCreate(ctx, bob)
	spOK(t, r, err)

	r, _, err = Ecdh(ctx, Bob, alice)
	assert.Equal(t, 0, r)
	assert.Error(t, err)
	assert.Equal(t, ErrorEcdh, err.Error())
}

func TestEcdhInvalidKey(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	alice, _ := hex.DecodeString("")
	bob := []byte{0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40, 0x40}

	r, Bob, err := EcPubkeyCreate(ctx, bob)
	spOK(t, r, err)

	r, _, err = Ecdh(ctx, Bob, alice)
	assert.Equal(t, 0, r)
	assert.Error(t, err)
	assert.Equal(t, ErrorPrivateKeySize, err.Error())
}
