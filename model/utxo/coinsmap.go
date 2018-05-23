package utxo

import (
	"fmt"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/util"
)

type CoinsMap map[outpoint.OutPoint]*Coin

func NewEmptyCoinsMap() *CoinsMap {
	cacheCoins := make(CoinsMap)
	return &cacheCoins
}
func (ctc CoinsMap) GetCoin(outpoint *outpoint.OutPoint) *Coin {
	coin := ctc[*outpoint]
	return coin
}

func (coinsCache CoinsMap) UnCache(point *outpoint.OutPoint) {
	_, ok := coinsCache[*point]
	if ok {
		delete(coinsCache, *point)
	}
}
func (coinsCache CoinsMap) Flush(hashBlock util.Hash) bool {
	println("flush=============")
	fmt.Printf("flush...coinsCache.====%#v \n  hashBlock====%#v", coinsCache, hashBlock)
	ok := GetUtxoCacheInstance().UpdateCoins(&coinsCache, &hashBlock)
	coinsCache = make(map[outpoint.OutPoint]*Coin)
	return ok == nil
}

func (coinsCache CoinsMap) AddCoin(point *outpoint.OutPoint, coin *Coin) {
	if coin.IsSpent() {
		panic("param coin should not be null")
	}
	// 脚本不可花
	txout := coin.GetTxOut()
	if !txout.IsSpendable() {
		return
	}
	fresh := false

	if true {
		oldCoin, ok := coinsCache[*point]
		if ok {
			//exist old Coin in cache
			if oldCoin.IsSpent() {
				panic("Adding new coin that replaces non-pruned entry")
			}
			fresh = !oldCoin.dirty
		} else {
			fresh = true
		}
	}
	newcoin := coin
	newcoin.dirty = true
	if fresh {
		newcoin.fresh = true
	}
	coinsCache[*point] = newcoin

}

func (coinsCache CoinsMap) SpendCoin(point *outpoint.OutPoint) *Coin {
	coin := coinsCache.GetCoin(point)
	if coin == nil {
		return coin
	}
	if coin.fresh {
		delete(coinsCache, *point)
	} else {
		coin.dirty = true
		coin.Clear()
	}
	return coin
}

// different from GetCoin, if not get coin, FetchCoin will get coin from global cache
func (coinsMap CoinsMap) FetchCoin(out *outpoint.OutPoint) *Coin {
	coin := coinsMap.GetCoin(out)
	if coin != nil {
		return coin
	}
	coin = GetUtxoCacheInstance().GetCoin(out)
	newCoin := coin.DeepCopy()
	if newCoin.IsSpent() {
		newCoin.fresh = true
		newCoin.dirty = false
	}
	coinsMap[*out] = newCoin
	return newCoin
}

// different from GetCoin, if not get coin, FetchCoin will get coin from global cache
func (coinsMap CoinsMap) SpendGlobalCoin(out *outpoint.OutPoint) *Coin {
	coin := coinsMap.FetchCoin(out)
	if coin == nil {
		return coin
	}

	return coinsMap.SpendCoin(out)
}
