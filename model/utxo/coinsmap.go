package utxo

import (
	"encoding/hex"
	"fmt"

	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/util"
)

type CoinsMap struct {
	cacheCoins map[outpoint.OutPoint]*Coin
}

func (cm *CoinsMap) GetMap() map[outpoint.OutPoint]*Coin {
	return cm.cacheCoins
}

func NewEmptyCoinsMap() *CoinsMap {
	cm := new(CoinsMap)
	cm.cacheCoins = make(map[outpoint.OutPoint]*Coin)
	return cm
}

func (cm *CoinsMap) AccessCoin(outpoint *outpoint.OutPoint) *Coin {
	entry := cm.GetCoin(outpoint)
	if entry == nil {
		return NewEmptyCoin()
	}
	return entry
}

func (cm *CoinsMap) GetCoin(outpoint *outpoint.OutPoint) *Coin {
	coin := cm.cacheCoins[*outpoint]
	return coin
}

func (cm *CoinsMap) UnCache(point *outpoint.OutPoint) {
	_, ok := cm.cacheCoins[*point]
	if ok {
		delete(cm.cacheCoins, *point)
	}
}

func DisplayCoinMap(cm *CoinsMap) {
	for k, v := range cm.GetMap() {
		if v.GetScriptPubKey() != nil {
			fmt.Printf("k=%+v, v=%+v, script=%s\n", k.String(), v, hex.EncodeToString(v.GetScriptPubKey().GetData()))
		} else {
			fmt.Printf("k=%+v, v=%+v\n", k.String(), v)
		}
	}
}

func (cm *CoinsMap) Flush(hashBlock util.Hash) bool {
	ok := GetUtxoCacheInstance().UpdateCoins(cm, &hashBlock)
	cm.cacheCoins = make(map[outpoint.OutPoint]*Coin)
	return ok == nil
}

func (cm *CoinsMap) AddCoin(point *outpoint.OutPoint, coin *Coin, possibleOverwrite bool) {
	if coin.IsSpent() {
		panic("add a spent coin")
	}
	//script is not spend
	if !coin.IsSpendable() {
		return
	}

	//if !possibleOverwrite {
	//	oldcoin := cm.FetchCoin(point)
	//	if oldcoin != nil {
	//		panic("Adding new coin that is in coincache or db")
	//	}
	//}

	coin.dirty = false
	coin.fresh = true
	cm.cacheCoins[*point] = coin

}

// SpendCoin spend a specified coin
func (cm *CoinsMap) SpendCoin(point *outpoint.OutPoint) *Coin {
	coin := cm.GetCoin(point)
	if coin == nil {
		return nil
	}
	if coin.fresh {
		if coin.dirty {
			if coin.IsSpent() {
				panic("spend a spent coin! ")
			} else {
				coin.Clear()
				return coin
			}
		} else {
			delete(cm.cacheCoins, *point)
		}
	} else {
		coin.dirty = true
		coin.Clear()
	}
	return coin
}

// FetchCoin different from GetCoin, if not get coin, FetchCoin will get coin from global cache
func (cm *CoinsMap) FetchCoin(out *outpoint.OutPoint) *Coin {
	coin := cm.GetCoin(out)
	if coin != nil {
		return coin
	}
	coin = GetUtxoCacheInstance().GetCoin(out)
	if coin == nil {
		log.Error("not found coin by outpoint(%v)", out)
		return nil
	}
	newCoin := coin.DeepCopy()
	if newCoin.IsSpent() {
		panic("coin from db should not be spent")
	}
	cm.cacheCoins[*out] = newCoin
	return newCoin
}

// SpendGlobalCoin different from GetCoin, if not get coin, FetchCoin will get coin from global cache
func (cm *CoinsMap) SpendGlobalCoin(out *outpoint.OutPoint) *Coin {
	coin := cm.FetchCoin(out)
	if coin == nil {
		return nil
	}
	copied := coin.DeepCopy()
	if cm.SpendCoin(out) != nil {
		return copied
	}
	return nil
}
