package utxo

import (
	//"fmt"
	"unsafe"

	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/util"
	"github.com/hashicorp/golang-lru"
	"github.com/syndtr/goleveldb/leveldb"
)

type CoinsLruCache struct {
	db         CoinsDB
	hashBlock  util.Hash
	cacheCoins *lru.Cache
	dirtyCoins map[outpoint.OutPoint]*Coin //write database temporary cache
}

func (coinsCache *CoinsLruCache) GetCoinsDB() CoinsDB {
	return coinsCache.db
}

func (coinsCache *CoinsLruCache) GetMap() map[outpoint.OutPoint]*Coin {
	return coinsCache.dirtyCoins
}

func InitUtxoLruTip(uc *UtxoConfig) {
	db := newCoinsDB(uc.Do)
	utxoTip = newCoinsLruCache(*db)
}

func newCoinsLruCache(db CoinsDB) CacheView {
	c := new(CoinsLruCache)
	c.db = db
	cache, err := lru.New(1000000)
	if err != nil {
		log.Error("Error: NewCoinsLruCache err %#v", err)
		panic("Error: NewCoinsLruCache err")
	}
	c.cacheCoins = cache
	c.dirtyCoins = make(map[outpoint.OutPoint]*Coin)
	return c
}

func (coinsCache *CoinsLruCache) GetCoin(outpoint *outpoint.OutPoint) *Coin {
	c, ok := coinsCache.cacheCoins.Get(*outpoint)
	if ok {
		log.Info("getCoin from cache")
		return c.(*Coin)
	}
	db := coinsCache.db
	coin, err := db.GetCoin(outpoint)
	if err != nil && err == leveldb.ErrNotFound {
		return nil
	}
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

	return coin
}

func (coinsCache *CoinsLruCache) HaveCoin(point *outpoint.OutPoint) bool {
	coin := coinsCache.GetCoin(point)
	return coin != nil && !coin.IsSpent()
}

func (coinsCache *CoinsLruCache) GetBestBlock() (util.Hash, error) {
	if coinsCache.hashBlock.IsNull() {
		hashBlock, err := coinsCache.db.GetBestBlock()
		if err == leveldb.ErrNotFound {
			//genesisHash := chain.GetInstance().GetParams().GenesisBlock.GetHash()
			//coinsCache.hashBlock = genesisHash
			//return coinsCache.hashBlock, nil
			return util.Hash{}, err
		}
		if err != nil {
			log.Error("db.GetBestBlock err:%#v", err)
			panic("db.GetBestBlock read err")
		}
		log.Debug("GetBestBlock: set coinsCache's besthash to %s from DB", hashBlock.String())
		coinsCache.hashBlock = *hashBlock
	}
	return coinsCache.hashBlock, nil
}

func (coinsCache *CoinsLruCache) UpdateCoins(cm *CoinsMap, hash *util.Hash) error {
	tempCacheCoins := cm.cacheCoins
	for point, tempCacheCoin := range tempCacheCoins {
		// Ignore non-dirty entries (optimization).
		if tempCacheCoin.isMempoolCoin {
			log.Error("MempoolCoin  save to DB!!!  %#v", tempCacheCoin)
			panic("MempoolCoin  save to DB!!!")
		}
		if tempCacheCoin.dirty || tempCacheCoin.fresh {
			coin, ok := coinsCache.cacheCoins.Get(point)
			// Lru could have deleted it from cache ,but ok.
			if !ok {
				if tempCacheCoin.fresh {
					tempCacheCoin.dirty = true
					coinsCache.cacheCoins.Add(point, tempCacheCoin)
					//ret := coinsCache.cacheCoins.Add(point, tempCacheCoin)
					//if !ret {
					//	log.Error("lruCache:add coin failed, please check")
					//}
					coinsCache.dirtyCoins[point] = tempCacheCoin
				} else {
					panic("newcoin is dirty and not fresh, but oldcoin is not exist")
				}
			} else {
				globalCacheCoin := coin.(*Coin)
				if tempCacheCoin.fresh && globalCacheCoin.IsSpent() {
					panic("newcoin fresh and oldcoin has spent")
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
					tempCacheCoin.dirty = true
					coinsCache.cacheCoins.Add(point, tempCacheCoin)
					coinsCache.dirtyCoins[point] = tempCacheCoin
				}
			}
		}
		delete(cm.cacheCoins, point)
	}
	log.Debug("UpdateCoins: set besthash to %s", hash.String())
	coinsCache.hashBlock = *hash
	return nil
}

func (coinsCache *CoinsLruCache) Flush() bool {
	log.Debug("flush utxo: bestblockhash:%s", coinsCache.hashBlock.String())

	if len(coinsCache.dirtyCoins) > 0 || !coinsCache.hashBlock.IsNull() {
		ok := coinsCache.db.BatchWrite(coinsCache.dirtyCoins, coinsCache.hashBlock)
		if ok == nil {
			coinsCache.cacheCoins.Purge()
		} else {
			panic("CoinsLruCache.flush err:")
		}
	}
	return true
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
