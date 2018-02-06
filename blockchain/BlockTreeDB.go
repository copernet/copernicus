package blockchain

import (
	"path/filepath"

	"bytes"
	"fmt"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/orm"
	"github.com/btcboost/copernicus/orm/database"
)

type BlockTreeDB struct {
	database.DBBase
	bucketKey string
}

func GetFileKey(file int) []byte {
	key := fmt.Sprintf("%s%d", orm.DB_BLOCK_FILES, file)
	return []byte(key)
}

func (blockTreeDB *BlockTreeDB) ReadBlockFileInfo(file int) *BlockFileInfo {
	var v []byte
	err := blockTreeDB.DBBase.View([]byte(blockTreeDB.bucketKey), func(bucket database.Bucket) error {
		v = bucket.Get(GetFileKey(file))
		return nil
	})
	if err != nil || v == nil {
		return nil
	}
	buf := bytes.NewBuffer(v)
	blockFileInfo, err := DeserializeBlockFileInfo(buf)
	if err != nil {
		return nil
	}
	return blockFileInfo

}

func (blockTreeDB *BlockTreeDB) WriteReindexing(reindexing bool) bool {
	err := blockTreeDB.DBBase.Update([]byte(blockTreeDB.bucketKey), func(bucket database.Bucket) error {
		var err error
		if reindexing {
			err = bucket.Put([]byte{orm.DB_REINDEX_FLAG}, []byte{'1'})
		} else {
			err = bucket.Delete([]byte{orm.DB_REINDEX_FLAG})
		}
		return err
	})
	return err == nil

}

func (blockTreeDB *BlockTreeDB) ReadReindexing() bool {
	var v []byte
	err := blockTreeDB.DBBase.View([]byte(blockTreeDB.bucketKey), func(bucket database.Bucket) error {
		v = bucket.Get([]byte{orm.DB_REINDEX_FLAG})
		return nil
	})
	if err != nil {
		return false
	}
	return v != nil
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
