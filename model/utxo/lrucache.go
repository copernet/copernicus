package utxo

import (
	"fmt"
	"unsafe"
	
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/util"
	"github.com/hashicorp/golang-lru"

)

type CoinsLruCache struct {
	db         CoinsDB
	hashBlock  util.Hash
	cacheCoins *lru.Cache
	dirtyCoins CoinsMap //写数据库临时缓存
}

var utxoLruTip CacheView

func InitUtxoLruTip(uc *UtxoConfig) {
	fmt.Printf("InitUtxoLruTip processing ....%v \n", uc)

	db := NewCoinsDB(uc.do)
	utxoLruTip = NewCoinsLruCache(*db)

}

func GetUtxoLruCacheInstance() CacheView {
	if utxoLruTip == nil {
		log.Error("utxoTip has not init!!")
	}
	return utxoLruTip
}

func NewCoinsLruCache(db CoinsDB) CacheView {
	c := new(CoinsLruCache)
	c.db = db
	cache, err := lru.New(1000000)
	if err != nil {
		log.Error("Error: NewCoinsLruCache err %#v", err)
		panic("Error: NewCoinsLruCache err")
	}
	c.cacheCoins = cache
	c.dirtyCoins = make(CoinsMap)
	return c
}

func (coinsCache *CoinsLruCache) GetCoin(outpoint *outpoint.OutPoint) *Coin {
	c, ok := coinsCache.cacheCoins.Get(*outpoint)
	if ok {
		fmt.Println("getCoin from cache")
		return c.(*Coin)
	}
	db := coinsCache.db
	coin, err := db.GetCoin(outpoint)
	if err != nil {
		log.Emergency("CoinsLruCache.GetCoin err:%#v", err)
		panic("get coin is failed!")
	}
	if coin == nil {
		return nil
	}
	coinsCache.cacheCoins.Add(*outpoint, coin)
	if coin.IsSpent() {
		// The parent only has an empty entry for this outpoint; we can consider
		// our version as fresh.
		coin.fresh = true
	}
	fmt.Println("getCoin from db")
	
	return coin
}

func (coinsCache *CoinsLruCache) HaveCoin(point *outpoint.OutPoint) bool {
	coin := coinsCache.GetCoin(point)
	return coin != nil && !coin.IsSpent()
}

func (coinsCache *CoinsLruCache) GetBestBlock() util.Hash {
	if coinsCache.hashBlock.IsNull() {
		hashBlock, err := coinsCache.db.GetBestBlock()
		if err != nil {
			log.Error("db.GetBestBlock err:%#v", err)
			panic("db.GetBestBlock read err")
		}
		coinsCache.hashBlock = *hashBlock
	}
	return coinsCache.hashBlock
}

func (coinsCache *CoinsLruCache) SetBestBlock(hash util.Hash) {
	coinsCache.hashBlock = hash
}

func (coinsCache *CoinsLruCache) UpdateCoins(tempCacheCoins *CoinsMap, hash *util.Hash) error {
	for point, tempCacheCoin := range *tempCacheCoins {
		// Ignore non-dirty entries (optimization).
		if tempCacheCoin.isMempoolCoin {
			log.Error("MempoolCoin  save to DB!!!  %#v", tempCacheCoin)
			panic("MempoolCoin  save to DB!!!")
		}
		if tempCacheCoin.dirty {
			coin, ok := coinsCache.cacheCoins.Get(point)
			// Lru could have deleted it from cache ,but ok.
			if !ok {
				if !(tempCacheCoin.fresh && tempCacheCoin.IsSpent()) {
					tempCacheCoin.dirty = true
					coinsCache.cacheCoins.Add(point, tempCacheCoin)
					coinsCache.dirtyCoins[point] = tempCacheCoin
					if tempCacheCoin.fresh {
						tempCacheCoin.fresh = true
					}
				}
			} else {
				globalCacheCoin := coin.(*Coin)
				if tempCacheCoin.fresh && !globalCacheCoin.IsSpent() {
					panic("FRESH flag misapplied to cache entry for base transaction with spendable outputs")
				}

				if globalCacheCoin.fresh && tempCacheCoin.IsSpent() {
					// The grandparent does not have an entry, and the child is
					// modified and being pruned. This means we can just delete
					// it from the parent.
					coinsCache.cacheCoins.Remove(point)
					_, ok = coinsCache.dirtyCoins[point]
					if ok {
						delete(coinsCache.dirtyCoins, point)
					}
				} else {
					*globalCacheCoin = *tempCacheCoin
					globalCacheCoin.dirty = true
					coinsCache.dirtyCoins[point] = globalCacheCoin
				}
			}
		}
		delete(*tempCacheCoins, point)
	}
	coinsCache.hashBlock = *hash
	return nil
}

func (coinsCache *CoinsLruCache) Flush() bool {
	println("flush=============")
	fmt.Printf("flush...coinsCache.cacheCoins====%#v \n  hashBlock====%#v", coinsCache.cacheCoins, coinsCache.hashBlock)
	ok := coinsCache.db.BatchWrite(&(coinsCache.dirtyCoins), coinsCache.hashBlock)

	//coinsCache.cacheCoins = make(CacheCoins)
	return ok == nil
}

//
//func (coinsCache *CoinsLruCache) AddCoin(point *outpoint.OutPoint, coin Coin, possibleOverwrite bool) {
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
//func (coinsCache *CoinsLruCache) SpendCoin(point *outpoint.OutPoint) (*Coin, error) {
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

func (coinsCache *CoinsLruCache) UnCache(point *outpoint.OutPoint) {
	c, ok := coinsCache.cacheCoins.Get(*point)
	coin := c.(*Coin)
	if ok && !coin.dirty && !coin.fresh {
		coinsCache.cacheCoins.Remove(*point)
		// donot delete from dirty map
	}
}

func (coinsCache *CoinsLruCache) GetCacheSize() int {
	return coinsCache.cacheCoins.Len()
}

func (coinsCache *CoinsLruCache) DynamicMemoryUsage() int64 {
	return int64(unsafe.Sizeof(coinsCache.cacheCoins))
}
//
//func (coinsCache *CoinsLruCache) GetOutputFor(tx *txin.TxIn) *txout.TxOut {
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
//func (coinsCache *CoinsLruCache) GetValueIn(tx *tx.Tx) amount.Amount {
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
//func (coinsCache *CoinsLruCache) HaveInputs(tx *tx.Tx) bool {
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
//func (coinsCache *CoinsLruCache) GetPriority(tx *tx.Tx, height uint32, chainInputValue *amount.Amount) float64 {
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
