package utxo

import (
	"math/rand"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

const (
	dirty = 1 << iota
	fresh
)

type coinCacheEntry struct {
	coin  Coin
	flags byte
}

type coinMap struct {
	cache  map[uint64]coinCacheEntry
	k0, k1 uint64
}

func newCoinMap() *coinMap {
	return &coinMap{
		k0:    rand.Uint64(),
		k1:    rand.Uint64(),
		cache: make(map[uint64]coinCacheEntry),
	}
}

func (cm *coinMap) find(out core.OutPoint) (coinCacheEntry, bool) {
	key := utils.SipHashExtra(cm.k0, cm.k1, out.Hash[:], out.Index)
	ret, ok := cm.cache[key]
	return ret, ok
}

func (cm *coinMap) add(out core.OutPoint, entry coinCacheEntry) bool {
	key := utils.SipHashExtra(cm.k0, cm.k1, out.Hash[:], out.Index)
	_, ok := cm.cache[key]
	if !ok {
		cm.cache[key] = entry
	}
	return !ok
}

func (cm *coinMap) del(out core.OutPoint) bool {
	key := utils.SipHashExtra(cm.k0, cm.k1, out.Hash[:], out.Index)
	_, ok := cm.cache[key]
	if ok {
		delete(cm.cache, key)
	}
	return ok
}
