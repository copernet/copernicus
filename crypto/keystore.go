package crypto

import (
	"github.com/copernet/copernicus/util"
	"sync"
)

type KeyPair struct {
	keyID      []byte
	publicKey  *PublicKey
	privateKey *PrivateKey
}

func NewKeyPair(privateKey *PrivateKey) *KeyPair {
	pubKey := privateKey.PubKey()
	pubKeyHash := pubKey.ToHash160()
	return &KeyPair{
		keyID:      pubKeyHash,
		publicKey:  pubKey,
		privateKey: privateKey,
	}
}

func (kd *KeyPair) GetKeyID() string {
	return string(kd.keyID)
}

func (kd *KeyPair) GetPublicKey() *PublicKey {
	return kd.publicKey
}

func (kd *KeyPair) GetPrivateKey() *PrivateKey {
	return kd.privateKey
}

type KeyStore struct {
	lock *sync.RWMutex
	keys map[string]*KeyPair
}

func NewKeyStore() *KeyStore {
	return &KeyStore{
		lock: new(sync.RWMutex),
		keys: make(map[string]*KeyPair),
	}
}

func (ks *KeyStore) AddKey(privateKey *PrivateKey) {
	keyPair := NewKeyPair(privateKey)

	ks.lock.Lock()
	defer ks.lock.Unlock()

	ks.keys[keyPair.GetKeyID()] = keyPair
}

func (ks *KeyStore) GetKeyPair(pubKeyHash []byte) *KeyPair {
	ks.lock.RLock()
	defer ks.lock.RUnlock()

	if keyPair, ok := ks.keys[string(pubKeyHash)]; ok {
		return keyPair
	}
	return nil
}

func (ks *KeyStore) GetKeyPairByPubKey(pubKey []byte) *KeyPair {
	pubKeyHash := util.Hash160(pubKey)
	return ks.GetKeyPair(pubKeyHash)
}

func (ks *KeyStore) GetKeyPairs(pubKeyHashList [][]byte) []*KeyPair {
	keys := make([]*KeyPair, 0, len(pubKeyHashList))

	ks.lock.RLock()
	defer ks.lock.RUnlock()

	for _, pubKeyHash := range pubKeyHashList {
		if keyPair, ok := ks.keys[string(pubKeyHash)]; ok {
			keys = append(keys, keyPair)
		}
	}
	return keys
}

func (ks *KeyStore) AddKeyPairs(keys []*KeyPair) {
	ks.lock.Lock()
	defer ks.lock.Unlock()

	for _, keyPair := range keys {
		ks.keys[keyPair.GetKeyID()] = keyPair
	}
}
