package utxo

import (
	"bytes"

	"fmt"

	"path/filepath"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/orm"
	"github.com/btcboost/copernicus/orm/database"
	"github.com/btcboost/copernicus/utils"
)

type CoinViewDB struct {
	database.DBBase
	bucketKey string
}

func (coinViewDB *CoinViewDB) GetCoin(outpoint *model.OutPoint) (coin *Coin) {
	coinEntry := NewCoinEntry(outpoint)
	var v []byte
	err := coinViewDB.DBBase.View([]byte(coinViewDB.bucketKey), func(bucket database.Bucket) error {
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

func (coinViewDB *CoinViewDB) HaveCoin(outpoint *model.OutPoint) bool {
	coinEntry := NewCoinEntry(outpoint)
	var v bool
	err := coinViewDB.DBBase.View([]byte(coinViewDB.bucketKey), func(bucket database.Bucket) error {
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
	err := coinViewDB.DBBase.Update([]byte(coinViewDB.bucketKey), func(bucket database.Bucket) error {
		err := bucket.Put([]byte{orm.DB_BEST_BLOCK}, hash.GetCloneBytes())
		return err
	})
	if err != nil {
		fmt.Println(err.Error())
	}
}

func (coinViewDB *CoinViewDB) GetBestBlock() *utils.Hash {
	var v []byte
	hash := new(utils.Hash)
	err := coinViewDB.DBBase.View([]byte(coinViewDB.bucketKey), func(bucket database.Bucket) error {
		v = bucket.Get([]byte{orm.DB_BEST_BLOCK})
		return nil
	})
	if err != nil || v == nil {
		return hash
	}
	hash.SetBytes(v)
	return hash
}

func (coinViewDB *CoinViewDB) BatchWrite(mapCoins map[model.OutPoint]CoinsCacheEntry) (bool, error) {
	count := 0
	changed := 0
	for k, v := range mapCoins {
		if v.Flags&COIN_ENTRY_DIRTY == COIN_ENTRY_DIRTY {
			coinEntry := NewCoinEntry(&k)
			if v.Coin.IsSpent() {
				err := coinViewDB.DBBase.Update([]byte(coinViewDB.bucketKey), func(bucket database.Bucket) error {
					err := bucket.Delete(coinEntry.GetSerKey())
					return err
				})
				if err != nil {
					return false, err
				}
			} else {
				bytes, err := v.Coin.GetSerialize()
				if err != nil {
					return false, err
				}
				err = coinViewDB.DBBase.Update([]byte(coinViewDB.bucketKey), func(bucket database.Bucket) error {
					err := bucket.Put(coinEntry.GetSerKey(), bytes)
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

func (coinViewDB *CoinViewDB) EstimateSize() uint16 {
	return 0
}

func NewCoinViewDB() *CoinViewDB {
	coinViewDB := new(CoinViewDB)
	path := conf.AppConf.DataDir + string(filepath.Separator) + "chainstate"
	db, err := orm.InitDB(orm.DBBolt, path)
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
