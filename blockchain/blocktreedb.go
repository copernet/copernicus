package blockchain

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/database/boltdb"
	"github.com/btcboost/copernicus/utils"
)

type BlockTreeDB struct {
	boltdb.DBBase
	bucketKey string
}

func GetFileKey(file int) []byte {
	key := fmt.Sprintf("%c%d", database.DB_BLOCK_FILES, file)
	return []byte(key)
}

func (blockTreeDB *BlockTreeDB) ReadBlockFileInfo(file int) *BlockFileInfo {
	var v []byte
	err := blockTreeDB.DBBase.View([]byte(blockTreeDB.bucketKey), func(bucket boltdb.Bucket) error {
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

func (blockTreeDB *BlockTreeDB) ReadTxIndex(txid *utils.Hash, pos *core.DiskTxPos) bool {
	// todo: not finish
	return true
}

func (blockTreeDB *BlockTreeDB) WriteReindexing(reindexing bool) bool {
	err := blockTreeDB.DBBase.Update([]byte(blockTreeDB.bucketKey), func(bucket boltdb.Bucket) error {
		var err error
		if reindexing {
			err = bucket.Put([]byte{database.DB_REINDEX_FLAG}, []byte{'1'})
		} else {
			err = bucket.Delete([]byte{database.DB_REINDEX_FLAG})
		}
		return err
	})
	return err == nil

}

func (blockTreeDB *BlockTreeDB) ReadReindexing() bool {
	var v []byte
	err := blockTreeDB.DBBase.View([]byte(blockTreeDB.bucketKey), func(bucket boltdb.Bucket) error {
		v = bucket.Get([]byte{database.DB_REINDEX_FLAG})
		return nil
	})
	if err != nil {
		return false
	}
	return v != nil
}

func (blockTreeDB *BlockTreeDB) WriteBatchSync(fileInfo []*BlockFileInfo, latFile int, blockIndexes []*core.BlockIndex) bool {
	err := blockTreeDB.DBBase.Update([]byte(blockTreeDB.bucketKey), func(bucket boltdb.Bucket) error {
		for _, f := range fileInfo {
			key := fmt.Sprintf("%c%d", database.DB_BLOCK_FILES, f.index)
			buf := bytes.NewBuffer(nil)
			err := f.Serialize(buf)
			if err != nil {
				return err
			}
			bucket.Put([]byte(key), buf.Bytes())
		}
		bucket.Put([]byte{database.DB_BLOCK_FILES}, []byte(strconv.Itoa(latFile)))
		for _, b := range blockIndexes {
			hashA := b.GetBlockHash()
			key := fmt.Sprintf("%c%s", database.DB_BLOCK_INDEX, hashA.ToString())
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

func (blockTreeDB *BlockTreeDB) LoadBlockIndexGuts(f func(hash *utils.Hash) *core.BlockIndex) bool {
	return true // todo complete
}

func (blockTreeDB *BlockTreeDB) ReadLastBlockFile(file *int) bool {
	//todo return Read(DB_LAST_BLOCK, nFile)
	return true // todo complete
}

func (blockTreeDB *BlockTreeDB) ReadFlag(name string, value bool) bool {
	// todo complete
	//char ch
	//if (!Read(std::make_pair(DB_FLAG, name), ch)) return false
	//fValue = ch == '1'
	return true
}

func NewBlockTreeDB() *BlockTreeDB {
	blockTreeDB := new(BlockTreeDB)
	path := conf.AppConf.DataDir + string(filepath.Separator) + "blocks" + string(filepath.Separator) + "index"
	db, err := database.InitDB(database.DBBolt, path)
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
