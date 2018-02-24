package utxo

import (
	"unsafe"

	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type CacheCoins map[model.OutPoint]*CoinsCacheEntry

type CoinsView interface {
	GetCoin(point *model.OutPoint, coin *Coin) bool
	HaveCoin(point *model.OutPoint) bool
	GetBestBlock() utils.Hash
	BatchWrite(coinsMap CacheCoins, hash *utils.Hash) bool
	EstimateSize() uint64
}

type CoinsViewCache struct {
	Base             CoinsView
	hashBlock        utils.Hash
	CacheCoins       CacheCoins
	cachedCoinsUsage int64
}

func (coinsViewCache *CoinsViewCache) AccessCoin(point *model.OutPoint) *Coin {
	entry := coinsViewCache.FetchCoin(point)
	if entry == nil {
		return NewEmptyCoin()
	}
	return entry.Coin
}

func (coinsViewCache *CoinsViewCache) FetchCoin(point *model.OutPoint) *CoinsCacheEntry {
	entry, ok := coinsViewCache.CacheCoins[*point]
	if ok {
		return entry
	}

	tmp := NewEmptyCoin()
	if !coinsViewCache.Base.GetCoin(point, tmp) {
		return nil
	}

	newEntry := NewCoinsCacheEntry(tmp)
	coinsViewCache.CacheCoins[*point] = newEntry
	if newEntry.Coin.IsSpent() {
		// The parent only has an empty entry for this outpoint; we can consider
		// our version as fresh.
		newEntry.Flags = COIN_ENTRY_FRESH
	}
	coinsViewCache.cachedCoinsUsage += newEntry.Coin.DynamicMemoryUsage()
	return newEntry
}

func (coinsViewCache *CoinsViewCache) GetCoin(point *model.OutPoint, coin *Coin) bool {
	entry := coinsViewCache.FetchCoin(point)
	if entry == nil {
		return false
	}
	tmp := DeepCopyCoin(entry.Coin)
	coin.HeightAndIsCoinBase = tmp.HeightAndIsCoinBase
	coin.TxOut = tmp.TxOut
	return true
}

func (coinsViewCache *CoinsViewCache) HaveCoin(point *model.OutPoint) bool {
	entry := coinsViewCache.FetchCoin(point)
	return entry != nil && !entry.Coin.IsSpent()
}

func (coinsViewCache *CoinsViewCache) HaveCoinInCache(point *model.OutPoint) bool {
	_, ok := coinsViewCache.CacheCoins[*point]
	return ok
}

func (coinsViewCache *CoinsViewCache) GetBestBlock() utils.Hash {
	if coinsViewCache.hashBlock.IsNull() {
		coinsViewCache.hashBlock = coinsViewCache.Base.GetBestBlock()
	}
	return coinsViewCache.hashBlock
}

func (coinsViewCache *CoinsViewCache) SetBestBlock(hash utils.Hash) {
	coinsViewCache.hashBlock = hash
}

func (coinsViewCache *CoinsViewCache) EstimateSize() uint64 {
	return 0
}

func (coinsViewCache *CoinsViewCache) BatchWrite(cacheCoins CacheCoins, hash *utils.Hash) bool {
	for point, item := range cacheCoins {
		// Ignore non-dirty entries (optimization).
		if item.Flags&COIN_ENTRY_DIRTY != 0 {
			itUs, ok := coinsViewCache.CacheCoins[point]
			if !ok {
				if !(item.Flags&COIN_ENTRY_FRESH != 0 && item.Coin.IsSpent()) {
					entry := &CoinsCacheEntry{Coin: item.Coin, Flags: COIN_ENTRY_DIRTY}
					coinsViewCache.CacheCoins[point] = entry
					coinsViewCache.cachedCoinsUsage += entry.Coin.DynamicMemoryUsage()
					if item.Flags&COIN_ENTRY_FRESH != 0 {
						entry.Flags |= COIN_ENTRY_FRESH
					}
				}
			} else {
				if item.Flags&COIN_ENTRY_FRESH != 0 && !itUs.Coin.IsSpent() {
					panic("FRESH flag misapplied to cache entry for base transaction with spendable outputs")
				}

				if itUs.Flags&COIN_ENTRY_FRESH != 0 && item.Coin.IsSpent() {
					coinsViewCache.cachedCoinsUsage -= itUs.Coin.DynamicMemoryUsage()
					delete(coinsViewCache.CacheCoins, point)
				} else {
					coinsViewCache.cachedCoinsUsage -= itUs.Coin.DynamicMemoryUsage()
					*itUs.Coin = DeepCopyCoin(item.Coin)
					coinsViewCache.cachedCoinsUsage += itUs.Coin.DynamicMemoryUsage()
					itUs.Flags |= COIN_ENTRY_DIRTY
				}
			}
		}
	}
	cacheCoins = make(CacheCoins)
	coinsViewCache.hashBlock = *hash
	return true
}

func (coinsViewCache *CoinsViewCache) Flush() bool {
	ok := coinsViewCache.Base.BatchWrite(coinsViewCache.CacheCoins, &coinsViewCache.hashBlock)
	//coinsViewCache.cacheCoins = make(CacheCoins)
	coinsViewCache.cachedCoinsUsage = 0
	return ok
}

func (coinsViewCache *CoinsViewCache) AddCoin(point *model.OutPoint, coin Coin, possibleOverwrite bool) {
	if coin.IsSpent() {
		panic("param coin should not be null")
	}
	if coin.TxOut.Script.IsUnspendable() {
		return
	}
	fresh := false
	_, ok := coinsViewCache.CacheCoins[*point]
	if !ok {
		coinsViewCache.CacheCoins[*point] = &CoinsCacheEntry{Coin: &Coin{TxOut: model.NewTxOut(-1, []byte{})}}
	} else {
		coinsViewCache.cachedCoinsUsage -= coinsViewCache.CacheCoins[*point].Coin.DynamicMemoryUsage()
	}

	if !possibleOverwrite {
		if !coinsViewCache.CacheCoins[*point].Coin.IsSpent() {
			panic("Adding new coin that replaces non-pruned entry")
		}
		fresh = coinsViewCache.CacheCoins[*point].Flags&COIN_ENTRY_DIRTY == 0
	}

	*coinsViewCache.CacheCoins[*point].Coin = DeepCopyCoin(&coin)

	if fresh {
		coinsViewCache.CacheCoins[*point].Flags |= COIN_ENTRY_DIRTY | COIN_ENTRY_FRESH
	} else {
		coinsViewCache.CacheCoins[*point].Flags |= COIN_ENTRY_DIRTY | 0
	}
	coinsViewCache.cachedCoinsUsage += coinsViewCache.CacheCoins[*point].Coin.DynamicMemoryUsage()
}

func (coinsViewCache *CoinsViewCache) SpendCoin(point *model.OutPoint, coin *Coin) bool {
	entry := coinsViewCache.FetchCoin(point)
	if entry == nil {
		return false
	}

	if coin != nil {
		*coin = *entry.Coin
	}
	if entry.Flags&COIN_ENTRY_FRESH != 0 {
		delete(coinsViewCache.CacheCoins, *point)
		coinsViewCache.cachedCoinsUsage -= entry.Coin.DynamicMemoryUsage()
	} else {
		entry.Flags |= COIN_ENTRY_DIRTY
		entry.Coin.Clear()
	}
	return true
}

func (coinsViewCache *CoinsViewCache) UnCache(point *model.OutPoint) {
	tmpEntry, ok := coinsViewCache.CacheCoins[*point]
	if ok && tmpEntry.Flags == 0 {
		coinsViewCache.cachedCoinsUsage -= tmpEntry.Coin.DynamicMemoryUsage()
		delete(coinsViewCache.CacheCoins, *point)
	}
}

func (coinsViewCache *CoinsViewCache) GetCacheSize() int {
	return len(coinsViewCache.CacheCoins)
}

func (coinsViewCache *CoinsViewCache) DynamicMemoryUsage() int64 {
	return int64(unsafe.Sizeof(coinsViewCache.CacheCoins))
}

func (coinsViewCache *CoinsViewCache) GetOutputFor(tx *model.TxIn) *model.TxOut {
	point := model.OutPoint{
		Hash:  tx.PreviousOutPoint.Hash,
		Index: tx.PreviousOutPoint.Index,
	}
	coin := coinsViewCache.AccessCoin(&point)
	if coin.IsSpent() {
		panic("coin should not be null")
	}
	return coin.TxOut
}

func (coinsViewCache *CoinsViewCache) GetValueIn(tx *model.Tx) btcutil.Amount {
	if tx.IsCoinBase() {
		return btcutil.Amount(0)
	}

	var result int64
	for _, item := range tx.Ins {
		result += coinsViewCache.GetOutputFor(item).Value
	}
	return btcutil.Amount(result)
}

func (coinsViewCache *CoinsViewCache) HaveInputs(tx model.Tx) bool {
	if tx.IsCoinBase() {
		return true
	}
	point := model.OutPoint{}
	for _, item := range tx.Ins {
		point.Hash = item.PreviousOutPoint.Hash
		point.Index = item.PreviousOutPoint.Index
		if !coinsViewCache.HaveCoin(&point) {
			return false
		}
	}
	return true
}

func (coinsViewCache *CoinsViewCache) GetPriority(tx *model.Tx, height uint32, chainInputValue *btcutil.Amount) float64 {
	if tx.IsCoinBase() {
		return 0.0
	}
	var result float64
	point := model.OutPoint{}
	for _, item := range tx.Ins {
		point.Hash = item.PreviousOutPoint.Hash
		point.Index = item.PreviousOutPoint.Index
		coin := coinsViewCache.AccessCoin(&point)
		if coin.IsSpent() {
			continue
		}
		if coin.GetHeight() <= height {
			result += float64(coin.TxOut.Value * int64(height-coin.GetHeight()))
			*chainInputValue += btcutil.Amount(coin.TxOut.Value)
		}
	}
	return tx.ComputePriority(result, 0)
}

func AddCoins(cache CoinsViewCache, tx model.Tx, height int) {
	isCoinbase := tx.IsCoinBase()
	txid := tx.Hash
	for i, out := range tx.Outs {
		// Pass fCoinbase as the possible_overwrite flag to AddCoin, in order to
		// correctly deal with the pre-BIP30 occurrences of duplicate coinbase
		// transactions.
		point := model.OutPoint{Hash: txid, Index: uint32(i)}
		coin := NewCoin(out, uint32(height), isCoinbase)
		cache.AddCoin(&point, *coin, isCoinbase)
	}
}

func AccessByTxid(coinsViewCache *CoinsViewCache, hash *utils.Hash) *Coin {
	out := model.OutPoint{Hash: *hash, Index: 0}
	for int(out.Index) < 11000 { // todo modify to be precise
		alternate := coinsViewCache.AccessCoin(&out)
		if !alternate.IsSpent() {
			return alternate
		}
		out.Index++
	}
	return NewEmptyCoin()
}
