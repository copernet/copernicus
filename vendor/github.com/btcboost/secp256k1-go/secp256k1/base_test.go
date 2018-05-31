package secp256k1

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"testing"
)

const (
	EcdsaTestVectors           = "sign_vectors.yaml"
	PubkeyCreateTestVectors    = "pubkey_vectors.yaml"
	PubkeyTweakAddTestVectors  = "pubkey_tweak_add_vectors.yaml"
	PubkeyTweakMulTestVectors  = "pubkey_tweak_mul_vectors.yaml"
	PrivkeyTweakAddTestVectors = "privkey_tweak_add_vectors.yaml"
	PrivkeyTweakMulTestVectors = "privkey_tweak_mul_vectors.yaml"
	TestCaseFmt                = "Test case %d"
)

func desc(i int) string {
	return fmt.Sprintf(TestCaseFmt, i)
}

func spOK(t *testing.T, result int, err error) {
	assert.NoError(t, err)
	assert.Equal(t, 1, result)
}

func readFile(filename string) []byte {
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return source
}

func removeSigHash(sig string) string {
	return strings.TrimSuffix(sig, "01")
}

func assertCanReadAndWritePublicKey(t *testing.T, ctx *Context, pkBytes []byte, flag uint) {
	r, pubkey, err := EcPubkeyParse(ctx, pkBytes)
	spOK(t, r, err)

	r, serialized, err := EcPubkeySerialize(ctx, pubkey, flag)
	spOK(t, r, err)
	assert.Equal(t, pkBytes, serialized)
}
