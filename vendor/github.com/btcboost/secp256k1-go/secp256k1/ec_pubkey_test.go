package secp256k1

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

type PubkeyCreateTestCase struct {
	PrivateKey      string `yaml:"seckey"`
	CompressedKey   string `yaml:"compressed"`
	UncompressedKey string `yaml:"pubkey"`
}

func (t *PubkeyCreateTestCase) GetPrivateKey() []byte {
	private, err := hex.DecodeString(t.PrivateKey)
	if err != nil {
		panic("Invalid private key")
	}
	return private
}
func (t *PubkeyCreateTestCase) GetCompressed() []byte {
	compressed, err := hex.DecodeString(t.CompressedKey)
	if err != nil {
		panic(err)
	}
	return compressed
}
func (t *PubkeyCreateTestCase) GetUncompressed() []byte {
	uncompressed, err := hex.DecodeString(t.UncompressedKey)
	if err != nil {
		panic(err)
	}
	return uncompressed
}

type PubkeyCreateFixtures []PubkeyCreateTestCase

func GetPubkeyCreateFixtures() PubkeyCreateFixtures {
	source := readFile(PubkeyCreateTestVectors)
	testCase := PubkeyCreateFixtures{}
	err := yaml.Unmarshal(source, &testCase)
	if err != nil {
		panic(err)
	}
	return testCase
}

func TestPubkeyCreateFixtures(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	fixtures := GetPubkeyCreateFixtures()

	for i := 0; i < len(fixtures); i++ {
		description := desc(i)
		t.Run(description, func(t *testing.T) {
			testCase := fixtures[i]
			priv := testCase.GetPrivateKey()

			r, publicKey, err := EcPubkeyCreate(ctx, priv)
			spOK(t, r, err)

			r, serializedComp, err := EcPubkeySerialize(ctx, publicKey, EcCompressed)
			spOK(t, r, err)
			assert.Equal(t, testCase.GetCompressed(), serializedComp)

			r, serializedUncomp, err := EcPubkeySerialize(ctx, publicKey, EcUncompressed)
			spOK(t, r, err)
			assert.Equal(t, testCase.GetUncompressed(), serializedUncomp)
		})
	}
}

func TestPubkeyParseFixtures(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	fixtures := GetPubkeyCreateFixtures()

	for i := 0; i < len(fixtures); i++ {
		description := desc(i)
		t.Run(description, func(t *testing.T) {
			assertCanReadAndWritePublicKey(t, ctx, fixtures[i].GetUncompressed(), EcUncompressed)
			assertCanReadAndWritePublicKey(t, ctx, fixtures[i].GetCompressed(), EcCompressed)
		})
	}
}

type PubkeyTweakAddTestCase struct {
	PublicKey string `yaml:"publicKey"`
	Tweak     string `yaml:"tweak"`
	Tweaked   string `yaml:"tweaked"`
}

func (t *PubkeyTweakAddTestCase) GetPublicKeyBytes() []byte {
	public, err := hex.DecodeString(t.PublicKey)
	if err != nil {
		panic("Invalid private key")
	}
	return public
}
func (t *PubkeyTweakAddTestCase) GetPublicKey(ctx *Context) *PublicKey {
	bytes := t.GetPublicKeyBytes()
	_, pubkey, err := EcPubkeyParse(ctx, bytes)
	if err != nil {
		panic(err)
	}
	return pubkey
}
func (t *PubkeyTweakAddTestCase) GetTweak() []byte {
	tweak, err := hex.DecodeString(t.Tweak)
	if err != nil {
		panic(err)
	}
	return tweak
}
func (t *PubkeyTweakAddTestCase) GetTweaked() []byte {
	tweaked, err := hex.DecodeString(t.Tweaked)
	if err != nil {
		panic(err)
	}
	return tweaked
}

type PubkeyTweakAddFixtures []PubkeyTweakAddTestCase

func GetPubkeyTweakAddFixtures() PubkeyTweakAddFixtures {
	source := readFile(PubkeyTweakAddTestVectors)
	testCase := PubkeyTweakAddFixtures{}
	err := yaml.Unmarshal(source, &testCase)
	if err != nil {
		panic(err)
	}
	return testCase
}

func TestPubkeyTweakAddFixtures(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	fixtures := GetPubkeyTweakAddFixtures()

	for i := 0; i < 1; i++ {
		description := desc(i)
		t.Run(description, func(t *testing.T) {
			fixture := fixtures[i]
			pubkey := fixture.GetPublicKey(ctx)
			tweak := fixture.GetTweak()

			r, err := EcPubkeyTweakAdd(ctx, pubkey, tweak)
			spOK(t, r, err)

			r, serialized, err := EcPubkeySerialize(ctx, pubkey, EcUncompressed)
			spOK(t, r, err)

			assert.Equal(t, fixture.GetTweaked(), serialized)
		})
	}
}

type PubkeyTweakMulTestCase struct {
	PublicKey string `yaml:"publicKey"`
	Tweak     string `yaml:"tweak"`
	Tweaked   string `yaml:"tweaked"`
}

func (t *PubkeyTweakMulTestCase) GetPublicKeyBytes() []byte {
	public, err := hex.DecodeString(t.PublicKey)
	if err != nil {
		panic("Invalid private key")
	}
	return public
}
func (t *PubkeyTweakMulTestCase) GetPublicKey(ctx *Context) *PublicKey {
	bytes := t.GetPublicKeyBytes()
	_, pubkey, err := EcPubkeyParse(ctx, bytes)
	if err != nil {
		panic(err)
	}
	return pubkey
}
func (t *PubkeyTweakMulTestCase) GetTweak() []byte {
	tweak, err := hex.DecodeString(t.Tweak)
	if err != nil {
		panic(err)
	}
	return tweak
}
func (t *PubkeyTweakMulTestCase) GetTweaked() []byte {
	tweaked, err := hex.DecodeString(t.Tweaked)
	if err != nil {
		panic(err)
	}
	return tweaked
}

type PubkeyTweakMulFixtures []PubkeyTweakMulTestCase

func GetPubkeyTweakMulFixtures() PubkeyTweakMulFixtures {
	source := readFile(PubkeyTweakMulTestVectors)
	testCase := PubkeyTweakMulFixtures{}
	err := yaml.Unmarshal(source, &testCase)
	if err != nil {
		panic(err)
	}
	return testCase
}

func TestPubkeyTweakMulFixtures(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	fixtures := GetPubkeyTweakMulFixtures()

	for i := 0; i < 1; i++ {
		description := desc(i)
		t.Run(description, func(t *testing.T) {
			fixture := fixtures[i]
			pubkey := fixture.GetPublicKey(ctx)
			tweak := fixture.GetTweak()

			r, err := EcPubkeyTweakMul(ctx, pubkey, tweak)
			spOK(t, r, err)

			r, serialized, err := EcPubkeySerialize(ctx, pubkey, EcUncompressed)
			spOK(t, r, err)

			assert.Equal(t, fixture.GetTweaked(), serialized)
		})
	}
}

func TestSecretKeyMustBeValid(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	numTests := 2

	tests := make([]string, numTests)
	tests[0] = "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141"
	tests[1] = "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364142"

	for i := 0; i < numTests; i++ {
		description := desc(i)
		t.Run(description, func(t *testing.T) {
			privkey, _ := hex.DecodeString(tests[i])
			r, pubkey, err := EcPubkeyCreate(ctx, privkey)
			assert.Error(t, err)
			assert.Equal(t, 0, r)
			assert.Nil(t, pubkey)
			assert.Equal(t, ErrorPublicKeyCreate, err.Error())
		})
	}
}
func TestPubKeyParseStringMustBeValid(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	numTests := 6

	tests := make([]string, numTests)
	tests[0] = ""
	// x exceeds curve order
	tests[1] = "02FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364142"
	tests[2] = "04FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364142FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141"
	tests[3] = "04FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364142"
	tests[4] = "04726fa5b19e9406aaa46ee22fd9e81a09dd5eb7c87505b93a11efcf4b945e778c"
	tests[5] = "99726fa5b19e9406aaa46ee22fd9e81a09dd5eb7c87505b93a11efcf4b945e778c"

	for i := 0; i < numTests; i++ {
		description := desc(i)
		t.Run(description, func(t *testing.T) {
			hexBytes, _ := hex.DecodeString(tests[i])
			r, pubkey, err := EcPubkeyParse(ctx, hexBytes)

			assert.Error(t, err, description)
			assert.Equal(t, 0, r, i, description)
			assert.Nil(t, pubkey, i, description)
			l := len(hexBytes)
			if l != LenCompressed && l != LenUncompressed {
				assert.Equal(t, ErrorPublicKeySize, err.Error(), description)
			} else {
				assert.Equal(t, ErrorPublicKeyParse, err.Error(), description)
			}
		})

	}
}

func TestPubkeyCreateChecksSize(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	badKey, _ := hex.DecodeString("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")

	r, pubkey, err := EcPubkeyCreate(ctx, badKey)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Nil(t, pubkey)
	assert.Equal(t, ErrorPrivateKeySize, err.Error())
}

func TestPubkeyTweakAddChecksTweakSize(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	pubkey, _ := hex.DecodeString("03e9a06e539d6bf5cf1ca5c41b59121fa3df07a338322405a312c67b6349a707e9")
	_, pk, err := EcPubkeyParse(ctx, pubkey)
	if err != nil {
		panic(err)
	}

	badTweak, _ := hex.DecodeString("AAAA")

	r, err := EcPubkeyTweakAdd(ctx, pk, badTweak)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Equal(t, ErrorTweakSize, err.Error())
}

func TestPubkeyTweakAddChecksForZero(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	priv, _ := hex.DecodeString("e9a06e539d6bf5cf1ca5c41b59121fa3df07a338322405a312c67b6349a707e9")
	r, pub, err := EcPubkeyCreate(ctx, priv)
	spOK(t, r, err)

	tweak, _ := hex.DecodeString("e9a06e539d6bf5cf1ca5c41b59121fa3df07a338322405a312c67b6349a707e9")
	r, err = EcPrivkeyNegate(ctx, tweak)
	spOK(t, r, err)

	r, err = EcPubkeyTweakAdd(ctx, pub, tweak)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Equal(t, ErrorTweakingPublicKey, err.Error())
}

func TestPubkeyTweakMulChecksTweakSize(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	pubkey, _ := hex.DecodeString("03e9a06e539d6bf5cf1ca5c41b59121fa3df07a338322405a312c67b6349a707e9")
	_, pk, err := EcPubkeyParse(ctx, pubkey)
	if err != nil {
		panic(err)
	}

	badTweak, _ := hex.DecodeString("AAAA")

	r, err := EcPubkeyTweakMul(ctx, pk, badTweak)
	assert.Error(t, err)
	assert.Equal(t, 0, r)
	assert.Equal(t, ErrorTweakSize, err.Error())
}

func TestPubkeyNegate(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	privkey := []byte(`abcd1234abcd1234abcd1234abcd1234`)

	// (-1*k)*G == -1*(k*G)
	privkeyCopy := []byte(`abcd1234abcd1234abcd1234abcd1234`)
	r, err := EcPrivkeyNegate(ctx, privkeyCopy)
	spOK(t, r, err)
	r, LHS, err := EcPubkeyCreate(ctx, privkeyCopy)
	spOK(t, r, err)

	r, rhs, err := EcPubkeyCreate(ctx, privkey)
	spOK(t, r, err)
	r, err = EcPubkeyNegate(ctx, rhs)
	assert.Equal(t, 1, r)
	assert.NoError(t, err)

	assert.Equal(t, LHS, rhs)

}

func TestPubkeyCombine(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	privkey := []byte(`abcd1234abcd1234abcd1234abcd1234`)

	// privkey * G
	r, pubkey, err := EcPubkeyCreate(ctx, privkey)
	spOK(t, r, err)

	// tweak * G
	tweak := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1}
	r, tweakPub, err := EcPubkeyCreate(ctx, tweak)
	spOK(t, r, err)

	// tweakedPriv: privkey + tweak
	// tweakedToPoint: (privkey+tweak) * G
	tweakedPriv := privkey
	r, err = EcPrivkeyTweakAdd(ctx, tweakedPriv, tweak)
	spOK(t, r, err)
	r, tweakedToPoint, err := EcPubkeyCreate(ctx, tweakedPriv)
	spOK(t, r, err)

	vPoint := []*PublicKey{pubkey, tweakPub}
	r, combinedPoint, err := EcPubkeyCombine(ctx, vPoint)
	spOK(t, r, err)

	assert.Equal(t, tweakedToPoint, combinedPoint)
}

func TestPubkeyCombineRequiresAtLeastOne(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	vPoint := []*PublicKey{}
	r, combinedPoint, err := EcPubkeyCombine(ctx, vPoint)
	assert.Error(t, err)
	assert.Nil(t, combinedPoint)
	assert.Equal(t, 0, r)
}

func TestPubkeyCombineWithOneReturnsSame(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	privkey := []byte(`abcd1234abcd1234abcd1234abcd1234`)

	// privkey * G
	r, pubkey, err := EcPubkeyCreate(ctx, privkey)
	spOK(t, r, err)

	vPoint := []*PublicKey{pubkey}
	r, combinedPoint, err := EcPubkeyCombine(ctx, vPoint)
	spOK(t, r, err)

	assert.Equal(t, pubkey, combinedPoint)
}

func TestPubkeyCombineInvalidSum(t *testing.T) {
	ctx, err := ContextCreate(ContextSign | ContextVerify)
	if err != nil {
		panic(err)
	}

	privkey := []byte(`abcd1234abcd1234abcd1234abcd1234`)

	// privkey * G
	r, pubkey, err := EcPubkeyCreate(ctx, privkey)
	spOK(t, r, err)

	// -privkey*G
	r, pubkeyNegate, err := EcPubkeyCreate(ctx, privkey)
	spOK(t, r, err)
	r, err = EcPubkeyNegate(ctx, pubkeyNegate)
	spOK(t, r, err)

	// (privkey-privkey)*G
	vPoint := []*PublicKey{pubkey, pubkeyNegate}
	r, combinedPoint, err := EcPubkeyCombine(ctx, vPoint)
	assert.Error(t, err)
	assert.Nil(t, combinedPoint)
	assert.Equal(t, 0, r)
	assert.Equal(t, ErrorPublicKeyCombine, err.Error())
}
