package utxo

import (
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

func (coinsCache *CoinsLruCache) RemoveCoins(point *outpoint.OutPoint) {
	if point != nil && coinsCache.GetCoin(point) != nil {
		coinsCache.cacheCoins.Remove(*point)
		log.Debug("remove coin is:%v", coinsCache.GetCoin(point))
	}
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
		log.Debug("GetBestBlock: set coinsCache's besthash to %s from DB", hashBlock)
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
	log.Debug("UpdateCoins: set besthash to %s", hash)
	coinsCache.hashBlock = *hash
	return nil
}

func (coinsCache *CoinsLruCache) Flush() bool {
	log.Debug("flush utxo: bestblockhash:%s", coinsCache.hashBlock)

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

func (coinsCache *CoinsLruCache) AccessByTxID(hash *util.Hash) *Coin {
	out := outpoint.OutPoint{Hash: *hash, Index: 0}
	for int(out.Index) < 11000 { // todo modify to be precise
		alternate := coinsCache.GetCoin(&out)
		if alternate != nil && !alternate.IsSpent() {
			return alternate
		}
		out.Index++
	}
	return nil
}

func (coinsCache *CoinsLruCache) GetCacheSize() int {
	return coinsCache.cacheCoins.Len()
}

func (coinsCache *CoinsLruCache) DynamicMemoryUsage() int64 {
	return int64(unsafe.Sizeof(coinsCache.cacheCoins))
}
