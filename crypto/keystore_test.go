package crypto

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func getTestPrivateKey() *PrivateKey {
	key := []byte{
		0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
		0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
		0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
		0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
	}
	privateKey := PrivateKeyFromBytes(key)
	return privateKey
}

func TestKeyPair(t *testing.T) {
	privateKey := getTestPrivateKey()
	keyPair := NewKeyPair(privateKey)

	assert.Equal(t, privateKey, keyPair.GetPrivateKey())
	assert.Equal(t, privateKey.PubKey(), keyPair.GetPublicKey())
	assert.Equal(t, string(privateKey.PubKey().ToHash160()), keyPair.GetKeyID())
}

func TestKeyStore_GetKeyPair(t *testing.T) {
	privateKey := getTestPrivateKey()
	publicKey := privateKey.PubKey()
	keyHash := publicKey.ToHash160()

	keyStore := NewKeyStore()
	keyStore.AddKey(privateKey)

	keyPair := keyStore.GetKeyPair(keyHash)
	assert.Equal(t, privateKey, keyPair.GetPrivateKey())

	keyPairInvalid := keyStore.GetKeyPair(keyHash[1:])
	assert.Nil(t, keyPairInvalid)
}

func TestKeyStore_GetKeyPairByPubKey(t *testing.T) {
	privateKey := getTestPrivateKey()
	publicKey := privateKey.PubKey()

	keyStore := NewKeyStore()
	keyStore.AddKey(privateKey)

	keyPair := keyStore.GetKeyPairByPubKey(publicKey.ToBytes())
	assert.Equal(t, privateKey, keyPair.GetPrivateKey())
}

func TestKeyStore_GetKeyPairs(t *testing.T) {
	privateKey := getTestPrivateKey()
	publicKey := privateKey.PubKey()
	keyHash := publicKey.ToHash160()

	keyStore := NewKeyStore()
	keyStore.AddKey(privateKey)

	keyHashList := make([][]byte, 0, 1)
	keyHashList = append(keyHashList, keyHash)
	keyPairs := keyStore.GetKeyPairs(keyHashList)

	assert.Equal(t, 1, len(keyPairs))
	if 1 == len(keyPairs) {
		assert.Equal(t, privateKey, keyPairs[0].GetPrivateKey())
	}
}

func TestKeyStore_AddKeyPairs(t *testing.T) {
	privateKey := getTestPrivateKey()
	publicKey := privateKey.PubKey()
	keyHash := publicKey.ToHash160()

	keyStore := NewKeyStore()
	keyStore.AddKey(privateKey)

	keyHashList := make([][]byte, 0, 1)
	keyHashList = append(keyHashList, keyHash)
	keyPairs := keyStore.GetKeyPairs(keyHashList)

	keyStoreNew := NewKeyStore()
	keyStoreNew.AddKeyPairs(keyPairs)
	assert.Equal(t, keyStore, keyStoreNew)
}
