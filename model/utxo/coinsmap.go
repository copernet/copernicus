package utxo

import (
	"github.com/copernet/copernicus/model/outpoint"
)

type CoinsMap struct {
	cacheCoins map[outpoint.OutPoint]*Coin
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

func (cm CoinsMap) UnCache(point *outpoint.OutPoint) {
	_, ok := cm.cacheCoins[*point]
	if ok {
		delete(cm.cacheCoins, *point)
	}
}

func (cm CoinsMap) AddCoin(point *outpoint.OutPoint, coin *Coin, possibleOverwrite bool) {
	if coin.IsSpent() {
		panic("param coin should not be null")
	}
	//script is not spend
	txout := coin.GetTxOut()
	if !txout.IsSpendable() {
		return
	}
	fresh := false

	if !possibleOverwrite {
		oldCoin, ok := cm.cacheCoins[*point]
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
	cm.cacheCoins[*point] = newcoin

}

func (cm CoinsMap) SpendCoin(point *outpoint.OutPoint) *Coin {
	coin := cm.GetCoin(point)
	if coin == nil {
		return coin
	}
	if coin.fresh {
		delete(cm.cacheCoins, *point)
	} else {
		coin.dirty = true
		coin.Clear()
	}
	return coin
}

// FetchCoin different from GetCoin, if not get coin, FetchCoin will get coin from global cache
func (cm CoinsMap) FetchCoin(out *outpoint.OutPoint) *Coin {
	coin := cm.GetCoin(out)
	if coin != nil {
		return coin
	}
	coin = GetUtxoCacheInstance().GetCoin(out)
	newCoin := coin.DeepCopy()
	if newCoin.IsSpent() {
		panic("coin from db should not be spent")
		//newCoin.fresh = true
		//newCoin.dirty = false
	}
	cm.cacheCoins[*out] = newCoin
	return newCoin
}

// SpendGlobalCoin different from GetCoin, if not get coin, FetchCoin will get coin from global cache
func (cm CoinsMap) SpendGlobalCoin(out *outpoint.OutPoint) *Coin {
	coin := cm.FetchCoin(out)
	if coin == nil {
		return coin
	}

	return cm.SpendCoin(out)
}
