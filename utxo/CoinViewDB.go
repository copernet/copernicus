package utxo

import (
	"bytes"
	
	"fmt"
	
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/orm"
	"github.com/btcboost/copernicus/orm/database"
	"github.com/btcboost/copernicus/utils"
)

type CoinViewDB struct {
	database.DBBase
	database.Bucket
}

func (coinViewDB *CoinViewDB) GetCoin(outpoint *model.OutPoint) (coin *Coin) {
	coinEntry := NewCoinEntry(outpoint)
	v := coinViewDB.Bucket.Get(coinEntry.GetSerKey())
	buf := bytes.NewBuffer(v)
	coin, err := DeserializeCoin(buf)
	if err != nil {
		return nil
	}
	return coin
}

func (coinViewDB *CoinViewDB) HaveCoin(outpoint *model.OutPoint) bool {
	coinEntry := NewCoinEntry(outpoint)
	return coinViewDB.Bucket.Exists(coinEntry.GetSerKey())
	
}

func (coinViewDB *CoinViewDB) BatchWrite(mapCoins map[model.OutPoint]CoinsCacheEntry, blockHash utils.Hash) (bool, error) {
	count := 0
	changed := 0
	for k, v := range mapCoins {
		if v.Flags&COIN_ENTRY_DIRTY == COIN_ENTRY_DIRTY {
			coinEntry := NewCoinEntry(&k)
			if v.Coin.IsSpent() {
				coinViewDB.Bucket.Delete(coinEntry.GetSerKey())
			} else {
				bytes, err := v.Coin.GetSerialize()
				if err != nil {
					return false, err
				}
				err = coinViewDB.Bucket.Put(coinEntry.GetSerKey(), bytes)
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

func NewCoinViewDB() *CoinViewDB {
	coinViewDB := new(CoinViewDB)
	path := conf.AppConf.DataDir + "/chainstate"
	db, err := orm.InitDB(orm.DBBolt, path)
	if err != nil {
		panic(err)
	}
	bucket, err := db.CreateIfNotExists([]byte("chainstate"))
	if err != nil {
		panic(err)
	}
	coinViewDB.DBBase = db
	coinViewDB.Bucket = bucket
	
	return coinViewDB
	
}
