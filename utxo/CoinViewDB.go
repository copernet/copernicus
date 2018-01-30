package utxo

import (
	"bytes"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/orm"
	"github.com/btcboost/copernicus/orm/database"
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
