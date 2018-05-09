package utxo

import (
	"unsafe"
	"fmt"

	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/util/amount"
	"github.com/btcboost/copernicus/model/txin"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/txout"
)
var utxoTip *CoinsCache


type UtxoConfig struct {

}

func InitUtxoTip(conf *UtxoConfig){
	fmt.Printf("initUtxo processing ....%v",conf)
	GetUtxoCacheInstance()
}

type CoinsCacheValue struct {
	Coin  *Coin
	dirty bool //是否修改过
	fresh bool //父cache中不存在
}

func NewCoinsCacheValue(coin *Coin) *CoinsCacheValue{
	return &CoinsCacheValue{Coin:coin}
}



func GetUtxoCacheInstance() *CoinsCache{
	if utxoTip == nil{
		db := new(CoinsDB)
		utxoTip = NewCoinCache(*db)
	}
	return utxoTip
}
type CoinsCacheMap map[outpoint.OutPoint]*CoinsCacheValue



type CoinsCache struct {
	db               CoinsDB
	hashBlock        util.Hash
	cacheCoins       CoinsCacheMap
	cachedCoinsUsage int64
}

func NewCoinCache(view CoinsDB) *CoinsCache {
	c := new(CoinsCache)
	c.db = view
	c.cachedCoinsUsage = 0
	return c
}

//func (coinsCache *CoinsCache) AccessCoin(point *outpoint.OutPoint) *Coin {
//	entry := coinsCache.FetchCoin(point)
//	if entry == nil {
//		return nil
//	}
//	return entry.Coin
//}

func (coinsCache *CoinsCache) FetchCoin(outpoint *outpoint.OutPoint) *CoinsCacheValue {
	entry, ok := coinsCache.cacheCoins[*outpoint]
	if ok {
		return entry
	}
    db := coinsCache.db
	coin, err := db.GetCoin(outpoint)
	if err != nil||coin == nil {
		return nil
	}
	newEntry := NewCoinsCacheValue(coin)
	coinsCache.cacheCoins[*outpoint] = newEntry
	if newEntry.Coin.IsSpent() {
		// The parent only has an empty entry for this outpoint; we can consider
		// our version as fresh.
		newEntry.fresh = true
	}
	coinsCache.cachedCoinsUsage += newEntry.Coin.DynamicMemoryUsage()
	return newEntry
}

func (coinsCache *CoinsCache) GetCoin(point *outpoint.OutPoint) (*Coin, error) {
	entry := coinsCache.FetchCoin(point)
	if entry == nil {
		return nil, nil
	}
	return entry.Coin, nil
}

func (coinsCache *CoinsCache) HaveCoin(point *outpoint.OutPoint) bool {
	entry := coinsCache.FetchCoin(point)
	return entry != nil && !entry.Coin.IsSpent()
}

func (coinsCache *CoinsCache) HaveCoinInCache(point *outpoint.OutPoint) bool {
	_, ok := coinsCache.cacheCoins[*point]
	return ok
}

func (coinsCache *CoinsCache) GetBestBlock() util.Hash {
	if coinsCache.hashBlock.IsNull() {
		coinsCache.hashBlock = coinsCache.db.GetBestBlock()
	}
	return coinsCache.hashBlock
}

func (coinsCache *CoinsCache) SetBestBlock(hash util.Hash) {
	coinsCache.hashBlock = hash
}

func (coinsCache *CoinsCache) EstimateSize() uint64 {
	return 0
}

func (coinsCache *CoinsCache) BatchWrite(cacheCoins *CoinsCacheMap, hash *util.Hash) error {
	for point, item := range *cacheCoins {
		// Ignore non-dirty entries (optimization).
		if item.dirty{
			itUs, ok := coinsCache.cacheCoins[point]
			if !ok {
				if !(item.fresh && item.Coin.IsSpent()) {
					entry := &CoinsCacheValue{Coin: item.Coin, dirty:true}
					coinsCache.cacheCoins[point] = entry
					coinsCache.cachedCoinsUsage +=  entry.Coin.DynamicMemoryUsage()
					if item.fresh {
						entry.fresh = true
					}
				}
			} else {
				if item.fresh && !itUs.Coin.IsSpent() {
					panic("FRESH flag misapplied to cache entry for base transaction with spendable outputs")
				}

				if itUs.fresh && item.Coin.IsSpent() {
					// The grandparent does not have an entry, and the child is
					// modified and being pruned. This means we can just delete
					// it from the parent.
					coinsCache.cachedCoinsUsage -= itUs.Coin.DynamicMemoryUsage()
					delete(coinsCache.cacheCoins, point)
				} else {
					coinsCache.cachedCoinsUsage -= itUs.Coin.DynamicMemoryUsage()
					itUs.Coin = item.Coin
					coinsCache.cachedCoinsUsage += itUs.Coin.DynamicMemoryUsage()
					itUs.dirty =  true
				}
			}
		}
		delete(*cacheCoins, point)
	}
	cacheCoins = new(CoinsCacheMap)
	coinsCache.hashBlock = *hash
	return nil
}

func (coinsCache *CoinsCache) Flush() bool {
	println("flush=============")
	fmt.Printf("flush...coinsCache.cacheCoins====%#v \n  hashBlock====%#v",coinsCache.cacheCoins,coinsCache.hashBlock)
	ok := coinsCache.db.BatchWrite(&coinsCache.cacheCoins, &coinsCache.hashBlock)
	//coinsCache.cacheCoins = make(CacheCoins)
	coinsCache.cachedCoinsUsage = 0
	return ok == nil
}

func (coinsCache *CoinsCache) AddCoin(point *outpoint.OutPoint, coin Coin, possibleOverwrite bool) {
	if coin.IsSpent() {
		panic("param coin should not be null")
	}
	if coin.GetTxOut().GetScriptPubKey().IsUnspendable() {
		return
	}
	fresh := false
	_, ok := coinsCache.cacheCoins[*point]
	if !ok {
		coinsCache.cacheCoins[*point] = &CoinsCacheValue{Coin: NewEmptyCoin()}
	} else {
		coinsCache.cachedCoinsUsage -= coinsCache.cacheCoins[*point].Coin.DynamicMemoryUsage()
	}

	if !possibleOverwrite {
		if !coinsCache.cacheCoins[*point].Coin.IsSpent() {
			panic("Adding new coin that replaces non-pruned entry")
		}
		fresh = coinsCache.cacheCoins[*point].dirty == false
	}

	*coinsCache.cacheCoins[*point].Coin = coin

	if fresh {
		coinsCache.cacheCoins[*point].dirty = true
		coinsCache.cacheCoins[*point].fresh = true
	} else {
		coinsCache.cacheCoins[*point].dirty = true
	}
	coinsCache.cachedCoinsUsage += coinsCache.cacheCoins[*point].Coin.DynamicMemoryUsage()
}

func (coinsCache *CoinsCache) SpendCoin(point *outpoint.OutPoint, coin *Coin) bool {
	entry := coinsCache.FetchCoin(point)
	if entry == nil {
		return false
	}

	if coin != nil {
		*coin = *entry.Coin
	}
	if entry.fresh {
		delete(coinsCache.cacheCoins, *point)
		coinsCache.cachedCoinsUsage -= entry.Coin.DynamicMemoryUsage()
	} else {
		entry.dirty = true
		entry.Coin.Clear()
	}
	return true
}

func (coinsCache *CoinsCache) UnCache(point *outpoint.OutPoint) {
	tmpEntry, ok := coinsCache.cacheCoins[*point]
	if ok && !tmpEntry.dirty && !tmpEntry.fresh{
		coinsCache.cachedCoinsUsage -= tmpEntry.Coin.DynamicMemoryUsage()
		delete(coinsCache.cacheCoins, *point)
	}
}

func (coinsCache *CoinsCache) GetCacheSize() int {
	return len(coinsCache.cacheCoins)
}

func (coinsCache *CoinsCache) DynamicMemoryUsage() int64 {
	return int64(unsafe.Sizeof(coinsCache.cacheCoins))
}

func (coinsCache *CoinsCache) GetOutputFor(tx *txin.TxIn) *txout.TxOut {
	point := outpoint.OutPoint{
		Hash:  tx.PreviousOutPoint.Hash,
		Index: tx.PreviousOutPoint.Index,
	}
	coin, _ := coinsCache.GetCoin(&point)
	if coin.IsSpent() {
		panic("coin should not be null")
	}
	return coin.txOut
}

func (coinsCache *CoinsCache) GetValueIn(tx *tx.Tx) amount.Amount {
	if tx.IsCoinBase() {
		return amount.Amount(0)
	}

	var result int64
	for _, item := range tx.GetIns() {
		result += coinsCache.GetOutputFor(item).GetValue()
	}
	return amount.Amount(result)
}

func (coinsCache *CoinsCache) HaveInputs(tx *tx.Tx) bool {
	if tx.IsCoinBase() {
		return true
	}
	point := outpoint.OutPoint{}
	for _, item := range tx.GetIns() {
		point.Hash = item.PreviousOutPoint.Hash

		point.Index = item.PreviousOutPoint.Index
		if !coinsCache.HaveCoin(&point) {
			return false
		}
	}
	return true
}
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

