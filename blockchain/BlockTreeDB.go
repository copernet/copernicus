package blockchain

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/orm"
	"github.com/btcboost/copernicus/orm/database"
	"github.com/btcboost/copernicus/utils"
)

type BlockTreeDB struct {
	database.DBBase
	bucketKey string
}

func GetFileKey(file int) []byte {
	key := fmt.Sprintf("%c%d", orm.DB_BLOCK_FILES, file)
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

func (blockTreeDB *BlockTreeDB) ReadTxIndex(txid *utils.Hash, pos *model.DiskTxPos) bool {
	// todo: not finish
	return true
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

func (blockTreeDB *BlockTreeDB) WriteBatchSync(fileInfo []*BlockFileInfo, latFile int, blockIndexes []*model.BlockIndex) bool {
	err := blockTreeDB.DBBase.Update([]byte(blockTreeDB.bucketKey), func(bucket database.Bucket) error {
		for _, f := range fileInfo {
			key := fmt.Sprintf("%c%d", orm.DB_BLOCK_FILES, f.index)
			buf := bytes.NewBuffer(nil)
			err := f.Serialize(buf)
			if err != nil {
				return err
			}
			bucket.Put([]byte(key), buf.Bytes())
		}
		bucket.Put([]byte{orm.DB_BLOCK_FILES}, []byte(strconv.Itoa(latFile)))
		for _, b := range blockIndexes {
			hashA := b.GetBlockHash()
			key := fmt.Sprintf("%c%s", orm.DB_BLOCK_INDEX, hashA.ToString())
			buf := bytes.NewBuffer(nil)
			diskBlock := NewDiskBlockIndex(b)
			err := diskBlock.Serialize(buf)
			if err != nil {
				return err
			}
			bucket.Put([]byte(key), buf.Bytes())

		}
		return nil

	})
	if err != nil {
		fmt.Println(err.Error())
	}
	return err == nil

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
