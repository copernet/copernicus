package utxo

import (
	"github.com/btcboost/copernicus/model/outpoint"
	"fmt"
	"github.com/btcboost/copernicus/util"
)
type CoinsMap struct {
	cacheCoins   map[outpoint.OutPoint]*Coin
}

func NewEmptyCoinsMap()*CoinsMap{
	cacheCoins := make(map[outpoint.OutPoint]*Coin)
	cm := CoinsMap{cacheCoins:cacheCoins}
	return &cm
}
func (ctc *CoinsMap) GetCoin(outpoint *outpoint.OutPoint)(*Coin){
	coin, _ := ctc.cacheCoins[*outpoint]
	return coin
}


func (coinsCache *CoinsMap) UnCache(point *outpoint.OutPoint) {
	_, ok := coinsCache.cacheCoins[*point]
	if ok {
		delete(coinsCache.cacheCoins, *point)
	}
}
func (coinsCache *CoinsMap) Flush(hashBlock util.Hash) bool {
	println("flush=============")
	fmt.Printf("flush...coinsCache.cacheCoins====%#v \n  hashBlock====%#v",coinsCache.cacheCoins,hashBlock)
	ok := GetUtxoCacheInstance().UpdateCoins(coinsCache, &hashBlock)
	coinsCache.cacheCoins = make(map[outpoint.OutPoint]*Coin)
	return ok == nil
}


func (coinsCache *CoinsMap) AddCoin(point *outpoint.OutPoint, coin Coin, possibleOverwrite bool) {
	if coin.IsSpent() {
		panic("param coin should not be null")
	}
	// 脚本不可花
	if !coin.GetTxOut().IsSpendable() {
		return
	}
	fresh := false
	_, ok := coinsCache.cacheCoins[*point]
	if !ok {
		coinsCache.cacheCoins[*point] = NewEmptyCoin()
	}

	if !possibleOverwrite {
		if !coinsCache.cacheCoins[*point].IsSpent() {
			panic("Adding new coin that replaces non-pruned entry")
		}
		fresh = coinsCache.cacheCoins[*point].dirty == false
	}

	*coinsCache.cacheCoins[*point] = coin

	if fresh {
		coinsCache.cacheCoins[*point].dirty = true
		coinsCache.cacheCoins[*point].fresh = true
	} else {
		coinsCache.cacheCoins[*point].dirty = true
	}
}

func (coinsCache *CoinsMap) SpendCoin(point *outpoint.OutPoint) *Coin {
	coin := coinsCache.GetCoin(point)
	if coin == nil {
		return coin
	}
	if coin.fresh {
		delete(coinsCache.cacheCoins, *point)
	} else {
		coin.dirty = true
		coin.Clear()
	}
	return coin
}

// different from GetCoin, if not get coin, FetchCoin will get coin from global cache
func (coinsMap *CoinsMap)FetchCoin(out *outpoint.OutPoint) *Coin{
	coin := coinsMap.GetCoin(out)
	if coin != nil{
		return coin
	}
	coin, _ = GetUtxoCacheInstance().GetCoin(out)
	return coin
}