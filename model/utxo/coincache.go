package utxo

import (
	"unsafe"
	"fmt"
	"github.com/btcboost/copernicus/log"

	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/outpoint"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/persist/db"
)
var utxoTip *CoinsCache


type UtxoConfig struct {
	do *db.DBOption
}

func InitUtxoTip(uc *UtxoConfig){
	fmt.Printf("initUtxo processing ....%v",uc)

	db := NewCoinsDB(uc.do)
	utxoTip = NewCoinCache(*db)

}


func GetUtxoCacheInstance() *CoinsCache{
	if utxoTip == nil{
		log.Error("utxoTip has not init!!")
	}
	return utxoTip
}


type CoinsCache struct {
	db               CoinsDB
	hashBlock        util.Hash
	cacheCoins        CoinsMap
	cachedCoinsUsage int64

}


func NewCoinCache(view CoinsDB) *CoinsCache {
	c := new(CoinsCache)
	c.db = view
	c.cacheCoins = *NewEmptyCoinsMap()
	c.cachedCoinsUsage = 0
	return c
}


func (coinsCache *CoinsCache) GetCoin(outpoint *outpoint.OutPoint) (*Coin, error) {
	coin, ok := coinsCache.cacheCoins[*outpoint]
	if ok {
		return coin, nil
	}
	db := coinsCache.db
	coin, err := db.GetCoin(outpoint)
	if err != nil{
		logs.Emergency("CoinsCache.GetCoin err:%#v", err)
		panic("get coin is failed!")
	}
	if err != nil||coin == nil {

		return nil, err
	}
	coinsCache.cacheCoins[*outpoint] = coin
	if coin.IsSpent() {
		// The parent only has an empty entry for this outpoint; we can consider
		// our version as fresh.
		coin.fresh = true
	}
	coinsCache.cachedCoinsUsage += coin.DynamicMemoryUsage()
	return coin, nil
}

func (coinsCache *CoinsCache) HaveCoin(point *outpoint.OutPoint) bool {
	coin, _ := coinsCache.GetCoin(point)
	return coin != nil && !coin.IsSpent()
}


func (coinsCache *CoinsCache) GetBestBlock() util.Hash {
	if coinsCache.hashBlock.IsNull() {
		hashBlock, err := coinsCache.db.GetBestBlock()
		if err != nil{
			log.Error("db.GetBestBlock err:%#v", err)
			panic("db.GetBestBlock read err")
		}
		coinsCache.hashBlock = *hashBlock
	}
	return coinsCache.hashBlock
}

func (coinsCache *CoinsCache) SetBestBlock(hash util.Hash) {
	coinsCache.hashBlock = hash
}

func (coinsCache *CoinsCache) EstimateSize() uint64 {
	return 0
}

func (coinsCache *CoinsCache) UpdateCoins(tempCacheCoins *CoinsMap, hash *util.Hash) error {
	for point, tempCacheCoin := range *tempCacheCoins {
		// Ignore non-dirty entries (optimization).
		if tempCacheCoin.dirty{
			globalCacheCoin, ok := coinsCache.cacheCoins[point]
			if !ok {
				if !(tempCacheCoin.fresh && tempCacheCoin.IsSpent()) {
					tempCacheCoin.dirty = true
					coinsCache.cacheCoins[point] = tempCacheCoin
					coinsCache.cachedCoinsUsage +=  tempCacheCoin.DynamicMemoryUsage()
					if tempCacheCoin.fresh {
						tempCacheCoin.fresh = true
					}
				}
			} else {
				if tempCacheCoin.fresh && !globalCacheCoin.IsSpent() {
					panic("FRESH flag misapplied to cache entry for base transaction with spendable outputs")
				}

				if globalCacheCoin.fresh && tempCacheCoin.IsSpent() {
					// The grandparent does not have an entry, and the child is
					// modified and being pruned. This means we can just delete
					// it from the parent.
					coinsCache.cachedCoinsUsage -= globalCacheCoin.DynamicMemoryUsage()
					delete(coinsCache.cacheCoins, point)
				} else {
					coinsCache.cachedCoinsUsage -= globalCacheCoin.DynamicMemoryUsage()
					globalCacheCoin = tempCacheCoin
					coinsCache.cachedCoinsUsage += globalCacheCoin.DynamicMemoryUsage()
					globalCacheCoin.dirty =  true
				}
			}
		}
		delete(*tempCacheCoins, point)
	}
	coinsCache.hashBlock = *hash
	return nil
}

func (coinsCache *CoinsCache) Flush() bool {
	println("flush=============")
	fmt.Printf("flush...coinsCache.cacheCoins====%#v \n  hashBlock====%#v",coinsCache.cacheCoins,coinsCache.hashBlock)
	ok := coinsCache.db.BatchWrite(&coinsCache.cacheCoins, coinsCache.hashBlock)
	//coinsCache.cacheCoins = make(CacheCoins)
	coinsCache.cachedCoinsUsage = 0
	return ok == nil
}
//
//func (coinsCache *CoinsCache) AddCoin(point *outpoint.OutPoint, coin Coin, possibleOverwrite bool) {
//	if coin.IsSpent() {
//		panic("param coin should not be null")
//	}
//	if !coin.GetTxOut().IsSpendable() {
//		return
//	}
//	fresh := false
//	_, ok := coinsCache.cacheCoins[*point]
//	if !ok {
//		coinsCache.cacheCoins[*point] = NewEmptyCoin()
//	} else {
//		coinsCache.cachedCoinsUsage -= coinsCache.cacheCoins[*point].DynamicMemoryUsage()
//	}
//
//	if !possibleOverwrite {
//		if !coinsCache.cacheCoins[*point].IsSpent() {
//			panic("Adding new coin that replaces non-pruned entry")
//		}
//		fresh = coinsCache.cacheCoins[*point].dirty == false
//	}
//
//	*coinsCache.cacheCoins[*point] = coin
//
//	if fresh {
//		coinsCache.cacheCoins[*point].dirty = true
//		coinsCache.cacheCoins[*point].fresh = true
//	} else {
//		coinsCache.cacheCoins[*point].dirty = true
//	}
//	coinsCache.cachedCoinsUsage += coinsCache.cacheCoins[*point].DynamicMemoryUsage()
//}
//
//func (coinsCache *CoinsCache) SpendCoin(point *outpoint.OutPoint) (*Coin, error) {
//	entry, err := coinsCache.GetCoin(point)
//	if entry == nil || err != nil {
//		return entry, err
//	}
//
//	if entry.fresh {
//		delete(coinsCache.cacheCoins, *point)
//		coinsCache.cachedCoinsUsage -= entry.DynamicMemoryUsage()
//	} else {
//		entry.dirty = true
//		entry.Clear()
//	}
//	return entry, err
//}

func (coinsCache *CoinsCache) UnCache(point *outpoint.OutPoint) {
	coin, ok := coinsCache.cacheCoins[*point]
	if ok && !coin.dirty && !coin.fresh{
		coinsCache.cachedCoinsUsage -= coin.DynamicMemoryUsage()
		delete(coinsCache.cacheCoins, *point)
	}
}

func (coinsCache *CoinsCache) GetCacheSize() int {
	return len(coinsCache.cacheCoins)
}

func (coinsCache *CoinsCache) DynamicMemoryUsage() int64 {
	return int64(unsafe.Sizeof(coinsCache.cacheCoins))
}
//
//func (coinsCache *CoinsCache) GetOutputFor(tx *txin.TxIn) *txout.TxOut {
//	point := outpoint.OutPoint{
//		Hash:  tx.PreviousOutPoint.Hash,
//		Index: tx.PreviousOutPoint.Index,
//	}
//	coin, _ := coinsCache.GetCoin(&point)
//	if coin.IsSpent() {
//		panic("coin should not be null")
//	}
//	return coin.txOut
//}
//
//func (coinsCache *CoinsCache) GetValueIn(tx *tx.Tx) amount.Amount {
//	if tx.IsCoinBase() {
//		return amount.Amount(0)
//	}
//
//	var result int64
//	for _, item := range tx.GetIns() {
//		result += coinsCache.GetOutputFor(item).GetValue()
//	}
//	return amount.Amount(result)
//}
//
//func (coinsCache *CoinsCache) HaveInputs(tx *tx.Tx) bool {
//	if tx.IsCoinBase() {
//		return true
//	}
//	point := outpoint.OutPoint{}
//	for _, item := range tx.GetIns() {
//		point.Hash = item.PreviousOutPoint.Hash
//
//		point.Index = item.PreviousOutPoint.Index
//		if !coinsCache.HaveCoin(&point) {
//			return false
//		}
//	}
//	return true
//}
//
//func (coinsCache *CoinsCache) GetPriority(tx *tx.Tx, height uint32, chainInputValue *amount.Amount) float64 {
//	if tx.IsCoinBase() {
//		return 0.0
//	}
//	var result float64
//	for _, item := range tx.Ins {
//		coin,_ := coinsCache.GetCoin(item.PreviousOutPoint)
//		if coin.IsSpent() {
//			continue
//		}
//		if coin.GetHeight() <= height {
//			result += float64(coin.GetTxOut().GetValue() * int64(height-coin.GetHeight()))
//			*chainInputValue += amount.Amount(coin.GetTxOut().GetValue())
//		}
//	}
//	return tx.ComputePriority(result, 0)
//}

