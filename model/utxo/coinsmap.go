package utxo

import (
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/util"
)

type CoinsMap struct {
	cacheCoins map[outpoint.OutPoint]*Coin
	hashBlock  util.Hash
}

func NewEmptyCoinsMap() *CoinsMap {
	cm := new(CoinsMap)
	cm.cacheCoins = make(map[outpoint.OutPoint]*Coin)
	return cm
}

func (cm CoinsMap) AccessCoin(outpoint *outpoint.OutPoint) *Coin {
	entry := cm.GetCoin(outpoint)
	if entry == nil {
		return NewEmptyCoin()
	}
	return entry
}

func (cm CoinsMap) GetCoin(outpoint *outpoint.OutPoint) *Coin {
	coin := cm.cacheCoins[*outpoint]
	return coin
}

func (coinsCache CoinsMap) UnCache(point *outpoint.OutPoint) {
	_, ok := coinsCache.cacheCoins[*point]
	if ok {
		delete(coinsCache.cacheCoins, *point)
	}
}

func (coinsCache CoinsMap) Flush(hashBlock util.Hash) bool {
	//println("flush=============")
	//fmt.Printf("flush...coinsCache.====%#v \n  hashBlock====%#v", coinsCache, hashBlock)
	ok := GetUtxoCacheInstance().UpdateCoins(&coinsCache, &hashBlock)
	coinsCache.cacheCoins = make(map[outpoint.OutPoint]*Coin)
	return ok == nil
}

func (coinsCache CoinsMap) AddCoin(point *outpoint.OutPoint, coin *Coin) {
	if coin.IsSpent() {
		panic("param coin should not be null")
	}
	//script is not spend
	txout := coin.GetTxOut()
	if !txout.IsSpendable() {
		return
	}
	fresh := false

	if true {
		oldCoin, ok := coinsCache.cacheCoins[*point]
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
	coinsCache.cacheCoins[*point] = newcoin

}

func (cm *CoinsMap) SetBestBlock(hash util.Hash) {
	cm.hashBlock = hash
}

func (coinsCache CoinsMap) SpendCoin(point *outpoint.OutPoint) *Coin {
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

// FetchCoin different from GetCoin, if not get coin, FetchCoin will get coin from global cache
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
	coinsMap.cacheCoins[*out] = newCoin
	return newCoin
}

// SpendGlobalCoin different from GetCoin, if not get coin, FetchCoin will get coin from global cache
func (coinsMap CoinsMap) SpendGlobalCoin(out *outpoint.OutPoint) *Coin {
	coin := coinsMap.FetchCoin(out)
	if coin == nil {
		return coin
	}

	return coinsMap.SpendCoin(out)
}
