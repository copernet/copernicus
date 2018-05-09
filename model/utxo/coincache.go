package utxo

import (
	"unsafe"

	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/util"
	"fmt"
	"copernicus/core"
)
var utxoTip *CoinsCache


type UtxoConfig struct {

}

func InitUtxoTip(conf *UtxoConfig){
	fmt.Printf("initUtxo processing ....%v",conf)
	GetUtxoCacheInstance()
}

type Coin struct {
	Coin  *utxo.Coin
	dirty bool //是否修改过
	fresh bool //父cache中不存在
}

func NewCoinsCacheEntry(coin *utxo.Coin) *CoinsCacheEntry{
	return &CoinsCacheEntry{Coin:coin}
}



func GetUtxoCacheInstance() *CoinsViewCache{
	if UtxoTip == nil{
		db := new(CoinsViewDB)
		UtxoTip = NewCoinViewCache(db)
	}
	return UtxoTip
}
type CoinsCacheMap map[core.OutPoint]*CoinsCacheEntry



type CoinsCache struct {
	db             utxo.CoinsDB
	hashBlock        util.Hash
	coinsCacheMap       CoinsCacheMap
	cachedCoinsUsage int64
}

func NewCoinViewCache(view coinsView) *CoinsViewCache {
	c := new(CoinsViewCache)
	c.Base = view
	c.cachedCoinsUsage = 0
	return c
}

//func (coinsViewCache *CoinsViewCache) AccessCoin(point *core.OutPoint) *utxo.Coin {
//	entry := coinsViewCache.FetchCoin(point)
//	if entry == nil {
//		return nil
//	}
//	return entry.Coin
//}

func (coinsViewCache *CoinsViewCache) FetchCoin(outpoint *core.OutPoint) *CoinsCacheEntry {
	entry, ok := coinsViewCache.cacheCoins[*outpoint]
	if ok {
		return entry
	}
    base := coinsViewCache.Base
	coin, err := base.GetCoin(outpoint)
	if err != nil||coin == nil {
		return nil
	}
	newEntry := NewCoinsCacheEntry(coin)
	coinsViewCache.cacheCoins[*outpoint] = newEntry
	if newEntry.Coin.IsSpent() {
		// The parent only has an empty entry for this outpoint; we can consider
		// our version as fresh.
		newEntry.fresh = true
	}
	coinsViewCache.cachedCoinsUsage += newEntry.Coin.DynamicMemoryUsage()
	return newEntry
}

func (coinsViewCache *CoinsViewCache) GetCoin(point *core.OutPoint) (*utxo.Coin, error) {
	entry := coinsViewCache.FetchCoin(point)
	if entry == nil {
		return nil, nil
	}
	return entry.Coin, nil
}

func (coinsViewCache *CoinsViewCache) HaveCoin(point *core.OutPoint) bool {
	entry := coinsViewCache.FetchCoin(point)
	return entry != nil && !entry.Coin.IsSpent()
}

func (coinsViewCache *CoinsViewCache) HaveCoinInCache(point *core.OutPoint) bool {
	_, ok := coinsViewCache.cacheCoins[*point]
	return ok
}

func (coinsViewCache *CoinsViewCache) GetBestBlock() util.Hash {
	if coinsViewCache.hashBlock.IsNull() {
		coinsViewCache.hashBlock = coinsViewCache.Base.GetBestBlock()
	}
	return coinsViewCache.hashBlock
}

func (coinsViewCache *CoinsViewCache) SetBestBlock(hash util.Hash) {
	coinsViewCache.hashBlock = hash
}

func (coinsViewCache *CoinsViewCache) EstimateSize() uint64 {
	return 0
}

func (coinsViewCache *CoinsViewCache) BatchWrite(cacheCoins *CacheCoins, hash *util.Hash) error {
	for point, item := range *cacheCoins {
		// Ignore non-dirty entries (optimization).
		if item.dirty{
			itUs, ok := coinsViewCache.cacheCoins[point]
			if !ok {
				if !(item.fresh && item.Coin.IsSpent()) {
					entry := &CoinsCacheEntry{Coin: item.Coin, dirty:true}
					coinsViewCache.cacheCoins[point] = entry
					coinsViewCache.cachedCoinsUsage +=  entry.Coin.DynamicMemoryUsage()
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
					coinsViewCache.cachedCoinsUsage -= itUs.Coin.DynamicMemoryUsage()
					delete(coinsViewCache.cacheCoins, point)
				} else {
					coinsViewCache.cachedCoinsUsage -= itUs.Coin.DynamicMemoryUsage()
					itUs.Coin = item.Coin
					coinsViewCache.cachedCoinsUsage += itUs.Coin.DynamicMemoryUsage()
					itUs.dirty =  true
				}
			}
		}
		delete(*cacheCoins, point)
	}
	cacheCoins = new(CacheCoins)
	coinsViewCache.hashBlock = *hash
	return nil
}

func (coinsViewCache *CoinsViewCache) Flush() bool {
	println("flush=============")
	fmt.Printf("flush...coinsViewCache.cacheCoins====%#v \n  hashBlock====%#v",coinsViewCache.cacheCoins,coinsViewCache.hashBlock)
	ok := coinsViewCache.Base.BatchWrite(&coinsViewCache.cacheCoins, &coinsViewCache.hashBlock)
	//coinsViewCache.cacheCoins = make(CacheCoins)
	coinsViewCache.cachedCoinsUsage = 0
	return ok == nil
}

func (coinsViewCache *CoinsViewCache) AddCoin(point *core.OutPoint, coin Coin, possibleOverwrite bool) {
	if coin.IsSpent() {
		panic("param coin should not be null")
	}
	if coin.GetTxOut().GetScriptPubKey().IsUnspendable() {
		return
	}
	fresh := false
	_, ok := coinsViewCache.cacheCoins[*point]
	if !ok {
		coinsViewCache.cacheCoins[*point] = &CoinsCacheEntry{Coin: NewEmptyCoin()}
	} else {
		coinsViewCache.cachedCoinsUsage -= coinsViewCache.cacheCoins[*point].Coin.DynamicMemoryUsage()
	}

	if !possibleOverwrite {
		if !coinsViewCache.cacheCoins[*point].Coin.IsSpent() {
			panic("Adding new coin that replaces non-pruned entry")
		}
		fresh = coinsViewCache.cacheCoins[*point].dirty == false
	}

	*coinsViewCache.cacheCoins[*point].Coin = coin

	if fresh {
		coinsViewCache.cacheCoins[*point].dirty = true
		coinsViewCache.cacheCoins[*point].fresh = true
	} else {
		coinsViewCache.cacheCoins[*point].dirty = true
	}
	coinsViewCache.cachedCoinsUsage += coinsViewCache.cacheCoins[*point].Coin.DynamicMemoryUsage()
}

func (coinsViewCache *CoinsViewCache) SpendCoin(point *core.OutPoint, coin *utxo.Coin) bool {
	entry := coinsViewCache.FetchCoin(point)
	if entry == nil {
		return false
	}

	if coin != nil {
		*coin = *entry.Coin
	}
	if entry.fresh {
		delete(coinsViewCache.cacheCoins, *point)
		coinsViewCache.cachedCoinsUsage -= entry.Coin.DynamicMemoryUsage()
	} else {
		entry.dirty = true
		entry.Coin.Clear()
	}
	return true
}

func (coinsViewCache *CoinsViewCache) UnCache(point *core.OutPoint) {
	tmpEntry, ok := coinsViewCache.cacheCoins[*point]
	if ok && !tmpEntry.dirty && !tmpEntry.fresh{
		coinsViewCache.cachedCoinsUsage -= tmpEntry.Coin.DynamicMemoryUsage()
		delete(coinsViewCache.cacheCoins, *point)
	}
}

func (coinsViewCache *CoinsViewCache) GetCacheSize() int {
	return len(coinsViewCache.cacheCoins)
}

func (coinsViewCache *CoinsViewCache) DynamicMemoryUsage() int64 {
	return int64(unsafe.Sizeof(coinsViewCache.cacheCoins))
}

func (coinsViewCache *CoinsViewCache) GetOutputFor(tx *core.TxIn) *core.TxOut {
	point := core.OutPoint{
		Hash:  tx.PreviousOutPoint.Hash,
		Index: tx.PreviousOutPoint.Index,
	}
	coin, _ := coinsViewCache.GetCoin(&point)
	if coin.IsSpent() {
		panic("coin should not be null")
	}
	return coin.txOut
}

func (coinsViewCache *CoinsViewCache) GetValueIn(tx *core.Tx) util.Amount {
	if tx.IsCoinBase() {
		return util.Amount(0)
	}

	var result int64
	for _, item := range tx.Ins {
		result += coinsViewCache.GetOutputFor(item).GetValue()
	}
	return util.Amount(result)
}

func (coinsViewCache *CoinsViewCache) HaveInputs(tx *core.Tx) bool {
	if tx.IsCoinBase() {
		return true
	}
	point := core.OutPoint{}
	for _, item := range tx.Ins {
		point.Hash = item.PreviousOutPoint.Hash

		point.Index = item.PreviousOutPoint.Index
		if !coinsViewCache.HaveCoin(&point) {
			return false
		}
	}
	return true
}

func (coinsViewCache *CoinsViewCache) GetPriority(tx *core.Tx, height uint32, chainInputValue *util.Amount) float64 {
	if tx.IsCoinBase() {
		return 0.0
	}
	var result float64
	for _, item := range tx.Ins {
		coin,_ := coinsViewCache.GetCoin(item.PreviousOutPoint)
		if coin.IsSpent() {
			continue
		}
		if coin.GetHeight() <= height {
			result += float64(coin.GetTxOut().GetValue() * int64(height-coin.GetHeight()))
			*chainInputValue += util.Amount(coin.GetTxOut().GetValue())
		}
	}
	return tx.ComputePriority(result, 0)
}

func AddCoins(cache CoinsViewCache, tx core.Tx, height int) {
	isCoinbase := tx.IsCoinBase()
	txid := tx.Hash
	for i, out := range tx.Outs {
		// Pass fCoinbase as the possible_overwrite flag to AddCoin, in order to
		// correctly deal with the pre-BIP30 occurrences of duplicate coinbase
		// transactions.
		point := core.OutPoint{Hash: txid, Index: uint32(i)}
		coin := NewCoin(out, uint32(height), isCoinbase)
		fmt.Printf("coin======%#v \n",coin.txOut.GetValue())
		cache.AddCoin(&point, *coin, isCoinbase)
	}
}

func AccessByTxid(coinsViewCache *CoinsViewCache, hash *util.Hash) *utxo.Coin {
	out := core.OutPoint{ *hash,  0}
	for int(out.Index) < 11000 { // todo modify to be precise
		alternate,_ := coinsViewCache.GetCoin(&out)
		if !alternate.IsSpent() {
			return alternate
		}
		out.Index++
	}
	return NewEmptyCoin()
}
