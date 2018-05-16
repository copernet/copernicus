package chain

import (
	"bytes"

	"strconv"

	
	"github.com/btcboost/copernicus/persist/db"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/model/blockindex"


	"github.com/btcboost/copernicus/util"
	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/pow"
	"github.com/btcboost/copernicus/model/consensus"
)

type BlockTreeDB struct {
	dbw *db.DBWrapper
}


func NewBlockTreeDB(do *db.DBOption) *BlockTreeDB {
	if do == nil {
		return nil
	}
	dbw, err := db.NewDBWrapper(&db.DBOption{
		FilePath:  conf.GetDataPath() + "/blocks/index",
		CacheSize: do.CacheSize,
		Wipe:      false,
	})
	if err != nil {
		panic("init DBWrapper failed...")
	}
	return &BlockTreeDB{
		dbw: dbw,
	}
}

func (blockTreeDB *BlockTreeDB) ReadBlockFileInfo(file int) (*block.BlockFileInfo, error){
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, db.DbBlockFiles)
	tmp = strconv.AppendInt(tmp, int64(file), 10)
	buf, err := blockTreeDB.dbw.Read(tmp)
	if err != nil {
		panic("read failed ....")
	}
	bufs := bytes.NewBuffer(buf)
	bfi := new(block.BlockFileInfo)
	err = bfi.Unserialize(bufs)
	return bfi, err
}
func (blockTreeDB *BlockTreeDB) WriteReindexing(reindexing bool) error {
	if reindexing {
		return blockTreeDB.dbw.Write([]byte{db.DbReindexFlag}, []byte{1}, false)
	}
	return blockTreeDB.dbw.Erase([]byte{db.DbReindexFlag}, false)
}

func (blockTreeDB *BlockTreeDB) ReadReindexing() bool {
	reindexing := blockTreeDB.dbw.Exists([]byte{db.DbReindexFlag})
	return reindexing
}
func (blockTreeDB *BlockTreeDB) ReadLastBlockFile() ([]byte, error) {
	return blockTreeDB.dbw.Read([]byte{db.DbLastBlock})
}


func (blockTreeDB *BlockTreeDB) WriteBatchSync(fileInfoList []*block.BlockFileInfo, lastFile int, blockIndexes []*blockindex.BlockIndex) error {
	batch  := db.NewBatchWrapper(blockTreeDB.dbw)
	for _, v := range fileInfoList {
		tmp := make([]byte, 0, 100)
		tmp = append(tmp, db.DbLastBlock)
		tmp = strconv.AppendInt(tmp, int64(v.GetIndex()), 10)
		buf := bytes.NewBuffer(nil)
		if err := v.Serialize(buf); err != nil {
			return err
		}
		batch.Write(tmp, buf.Bytes())
	}
	buff := make([]byte, 0, 100)
	buff = strconv.AppendInt(buff, int64(lastFile), 10)
	batch.Write([]byte{db.DbLastBlock}, buff)

	for _, v := range blockIndexes {
		tmp := make([]byte, 0, 100)
		tmp = append(tmp, db.DbBlockIndex)
		buff := bytes.NewBuffer(nil)
		v.GetBlockHash().Serialize(buff)
		tmp = append(tmp, buff.Bytes()...)
		buff.Reset()
		if err := v.Serialize(buff); err != nil {
			return err
		}
		batch.Write(tmp, buff.Bytes())
	}

	return blockTreeDB.dbw.WriteBatch(batch, true)
}


func (blockTreeDB *BlockTreeDB) ReadTxIndex(txid *util.Hash) ([]byte, error) {
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, db.DbTxIndex)
	tmp = append(tmp, txid[:]...)
	return blockTreeDB.dbw.Read(tmp)
}

func (blockTreeDB *BlockTreeDB) WriteTxIndex(txIndexes map[util.Hash] block.DiskTxPos) error {
	var batch  = db.NewBatchWrapper(blockTreeDB.dbw)
	for k, v := range txIndexes {
		key := make([]byte, 0, 100)
		key = append(key, db.DbTxIndex)
		key = append(key, k[:]...)

		buf := bytes.NewBuffer(nil)
		if err := v.SerializeDiskTxPos(buf); err != nil {
			return err
		}
		batch.Write(key, buf.Bytes())
	}
	return blockTreeDB.dbw.WriteBatch(batch, false)
}



func (blockTreeDB *BlockTreeDB) WriteFlag(name string, value bool) error {
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, db.DbFlag)
	tmp = append(tmp, name...)
	if !value {
		return blockTreeDB.dbw.Write(tmp, []byte{'1'}, value)
	}
	return blockTreeDB.dbw.Write(tmp, []byte{'0'}, value)
}

func (blockTreeDB *BlockTreeDB) ReadFlag(name string) bool {
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, db.DbFlag)
	tmp = append(tmp, name...)
	b, err := blockTreeDB.dbw.Read(tmp)

	if b[0] == '1' && err == nil {
		return true
	}
	return false
}


//

// todo for iter and check key、 pow
func (blockTreeDB *BlockTreeDB) LoadBlockIndexGuts(f func(hash *util.Hash) *blockindex.BlockIndex) bool {
	cursor:=blockTreeDB.dbw.Iterator()
	defer cursor.Close()
	hash := util.Hash{}
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, db.DbBlockIndex)
	tmp = append(tmp, hash[:]...)
	cursor.Seek(tmp)

	// Load mapBlockIndex
	for cursor.Valid() {
		//todo:boost::this_thread::interruption_point();
		type key struct {
			b    byte
			hash util.Hash
		}
		k := cursor.GetKey()
		kk := key{}
		if k == nil || kk.b != db.DbBlockIndex {
			break
		}

		var diskindex blockindex.BlockIndex
		val := cursor.GetVal()
		if val == nil {
			logs.Error("LoadBlockIndex() : failed to read value")
			return false
		}
		diskindex.Unserialize(bytes.NewBuffer(val))


		if new(pow.Pow).CheckProofOfWork(diskindex.GetBlockHash(), diskindex.Header.Bits, &consensus.MainNetParams) {
			logs.Error("LoadBlockIndex(): CheckProofOfWork failed: %s", diskindex.ToString())
			return false
		}

		cursor.Next()
	}

	return true
}


