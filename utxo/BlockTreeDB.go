package utxo

import (
	"path/filepath"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/orm"
	"github.com/btcboost/copernicus/orm/database"
)

type BlockTreeDB struct {
	database.DBBase
	bucketKey string
}

func NewBlockTreeDB() *BlockTreeDB {
	blockTreeDB := new(BlockTreeDB)
	path := conf.AppConf.DataDir + string(filepath.Separator) + "blocks" + string(filepath.Separator) + "index"
	db, err := orm.InitDB(orm.DBBolt, path)
	if err != nil {
		panic(err)
	}
	_, err = db.CreateIfNotExists([]byte("index"))
	if err != nil {
		panic(err)
	}
	blockTreeDB.DBBase = db
	blockTreeDB.bucketKey = "index"
	return blockTreeDB
}
