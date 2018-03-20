package blockchain

import (
	"bytes"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/utils"
)

type BlockTreeDB struct {
	dbw       *database.DBWrapper
	bucketKey string
}

func GetFileKey(file int) []byte {
	return nil
}

func (blockTreeDB *BlockTreeDB) ReadBlockFileInfo(file int) *BlockFileInfo {
	var v []byte
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
	return true
}

func (blockTreeDB *BlockTreeDB) ReadReindexing() bool {
	return true
}

func (blockTreeDB *BlockTreeDB) WriteBatchSync(fileInfo []*BlockFileInfo, latFile int, blockIndexes []*core.BlockIndex) bool {

	return true
}

func (blockTreeDB *BlockTreeDB) LoadBlockIndexGuts(f func(hash *utils.Hash) *core.BlockIndex) bool {
	return true // todo complete
}

func (blockTreeDB *BlockTreeDB) ReadLastBlockFile(file *int) bool {
	//todo return Read(DB_LAST_BLOCK, nFile)
	return true // todo complete
}

func (blockTreeDB *BlockTreeDB) ReadFlag(name string, value bool) bool {

	return true
}

func NewBlockTreeDB() *BlockTreeDB {
	blockTreeDB := new(BlockTreeDB)

	return blockTreeDB
}
