package utxo

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/database/boltdb"
	"github.com/btcboost/copernicus/utils"
)

type CoinViewDB struct {
	boltdb.DBBase
	bucketKey string
}

func (coinViewDB *CoinViewDB) GetCoin(outpoint *core.OutPoint) (coin *Coin) {
	coinEntry := NewCoinEntry(outpoint)
	var v []byte
	err := coinViewDB.DBBase.View([]byte(coinViewDB.bucketKey), func(bucket boltdb.Bucket) error {
		v = bucket.Get(coinEntry.GetSerKey())
		return nil
	})
	buf := bytes.NewBuffer(v)
	coin, err = DeserializeCoin(buf)
	if err != nil {
		return nil
	}
	return coin
}

func (coinViewDB *CoinViewDB) HaveCoin(outpoint *core.OutPoint) bool {
	coinEntry := NewCoinEntry(outpoint)
	var v bool
	err := coinViewDB.DBBase.View([]byte(coinViewDB.bucketKey), func(bucket boltdb.Bucket) error {
		v = bucket.Exists(coinEntry.GetSerKey())
		return nil
	})
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return v

}

func (coinViewDB *CoinViewDB) SetBestBlock(hash *utils.Hash) {
	err := coinViewDB.DBBase.Update([]byte(coinViewDB.bucketKey), func(bucket boltdb.Bucket) error {
		err := bucket.Put([]byte{database.DB_BEST_BLOCK}, hash.GetCloneBytes())
		return err
	})
	if err != nil {
		fmt.Println(err.Error())
	}
}

func (coinViewDB *CoinViewDB) GetBestBlock() utils.Hash {
	var v []byte
	hash := utils.Hash{}
	err := coinViewDB.DBBase.View([]byte(coinViewDB.bucketKey), func(bucket boltdb.Bucket) error {
		v = bucket.Get([]byte{database.DB_BEST_BLOCK})
		return nil
	})
	if err != nil || v == nil {
		return hash
	}
	hash.SetBytes(v)
	return hash
}

func (coinViewDB *CoinViewDB) BatchWrite(mapCoins map[core.OutPoint]CoinsCacheEntry) (bool, error) {
	count := 0
	changed := 0
	for k, v := range mapCoins {
		if v.Flags&COIN_ENTRY_DIRTY == COIN_ENTRY_DIRTY {
			coinEntry := NewCoinEntry(&k)
			if v.Coin.IsSpent() {
				err := coinViewDB.DBBase.Update([]byte(coinViewDB.bucketKey), func(bucket boltdb.Bucket) error {
					err := bucket.Delete(coinEntry.GetSerKey())
					return err
				})
				if err != nil {
					return false, err
				}
			} else {
				b, err := v.Coin.GetSerialize()
				if err != nil {
					return false, err
				}
				err = coinViewDB.DBBase.Update([]byte(coinViewDB.bucketKey), func(bucket boltdb.Bucket) error {
					err := bucket.Put(coinEntry.GetSerKey(), b)
					return err
				})
				if err != nil {
					return false, err
				}
			}
			changed++
			delete(mapCoins, k)
		}
		count++

	}
	fmt.Println("coin", "committed %d changed transcation outputs (out of %d) to coin databse", changed, count)
	return true, nil

}

func (coinViewDB *CoinViewDB) EstimateSize() int {
	var size int
	err := coinViewDB.DBBase.View([]byte(coinViewDB.bucketKey), func(bucket boltdb.Bucket) error {
		size = bucket.EstimateSize()
		return nil
	})
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}
	return size
}

func (coinViewDB *CoinViewDB) Cursor() *CoinsViewCursor {
	var cursor boltdb.Cursor
	//todo CCoinsViewDB Cursor
	coinViewDB.View([]byte(coinViewDB.bucketKey), func(bucket boltdb.Bucket) error {

		for i := bucket.Cursor(); i.Last(); i.Next() {
			if i.Seek([]byte{database.DB_COIN}) {
				cursor = i
			}
		}
		return nil

	})
	coinsViewCursor := NewCoinsViewCursor(cursor, coinViewDB.GetBestBlock())
	return coinsViewCursor

}

func NewCoinViewDB() *CoinViewDB {
	coinViewDB := new(CoinViewDB)
	path := conf.AppConf.DataDir + string(filepath.Separator) + "chainstate"
	db, err := database.InitDB(database.DBBolt, path)
	if err != nil {
		panic(err)
	}
	_, err = db.CreateIfNotExists([]byte("chainstate"))
	if err != nil {
		panic(err)
	}
	coinViewDB.DBBase = db
	coinViewDB.bucketKey = "chainstate"

	return coinViewDB

}
