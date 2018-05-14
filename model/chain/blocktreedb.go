package chain

import (
	"bytes"

	"strconv"

	
	"github.com/btcboost/copernicus/persist/db"

	"github.com/btcboost/copernicus/conf"
	. "github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"encoding/binary"
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

func (blockTreeDB *BlockTreeDB) ReadBlockFileInfo(file int) (*BlockFileInfo, error){
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, db.DbBlockFiles)
	tmp = strconv.AppendInt(tmp, int64(file), 10)
	buf, err := blockTreeDB.dbw.Read(tmp)
	if err != nil {
		panic("read failed ....")
	}
	bufs := bytes.NewBuffer(buf)
	bfi := new(BlockFileInfo)
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


func (blockTreeDB *BlockTreeDB) WriteBatchSync(fileInfoList []*BlockFileInfo, lastFile int, blockIndexes []*blockindex.BlockIndex) error {
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
	file := IntToBytes(lastFile)
	batch.Write([]byte{db.DbLastBlock}, file)

	for _, v := range blockIndexes {
		tmp := make([]byte, 0, 100)
		tmp = append(tmp, db.DbBlockIndex)
		buf := bytes.NewBuffer(nil)
		v.GetBlockHash().Serialize(buf)
		tmp = append(tmp, buf.Bytes()...)
		if err := v.Serialize(buf); err != nil {
			return err
		}
		batch.Write(tmp, buf.Bytes())
	}

	return blockTreeDB.dbw.WriteBatch(batch, true)
}

//
//type writeTxIndex struct {
//	hash utils.Hash
//	pos  core.DiskTxPos
//}
//func (blockTreeDB *BlockTreeDB) ReadTxIndex(txid *utils.Hash) ([]byte, error) {
//	tmp := make([]byte, 0, 100)
//	tmp = append(tmp, db.DbTxIndex)
//	tmp = append(tmp, txid[:]...)
//	return blockTreeDB.dbw.Read(tmp)
//}
//
//func (blockTreeDB *BlockTreeDB) WriteTxIndex(ect []*writeTxIndex) error {
//	var batch *db.BatchWrapper
//	for _, v := range ect {
//		key := make([]byte, 0, 100)
//		key = append(key, db.DbTxIndex)
//		key = append(key, v.hash[:]...)
//
//		buf := bytes.NewBuffer(nil)
//		if err := v.pos.SerializeDiskTxPos(buf); err != nil {
//			return err
//		}
//		batch.Write(key, buf.Bytes())
//	}
//	return blockTreeDB.dbw.WriteBatch(batch, false)
//}
//
//
//
//
//
////
////type bkFileInfo struct {
////	i       int
////	bkfInfo *BlockFileInfo
////}
//
func IntToBytes(i int) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}
//
//
//func (blockTreeDB *BlockTreeDB) LoadBlockIndexGuts(f func(hash *utils.Hash) *core.BlockIndex) bool {
//	cursor:=blockTreeDB.dbw.Iterator()
//	defer cursor.Close()
//	hash := utils.Hash{}
//	tmp := make([]byte, 0, 100)
//	tmp = append(tmp, db.DbBlockIndex)
//	tmp = append(tmp, hash[:]...)
//	cursor.Seek(tmp)
//
//	// Load mapBlockIndex
//	for cursor.Valid() {
//		//todo:boost::this_thread::interruption_point();
//		type key struct {
//			b    byte
//			hash utils.Hash
//		}
//		k := cursor.GetKey()
//		kk := key{}
//		if k == nil || kk.b != db.DbBlockIndex {
//			break
//		}
//
//		var diskindex DiskBlockIndex
//		val := cursor.GetVal()
//		if val == nil {
//			logs.Error("LoadBlockIndex() : failed to read value")
//			return false
//		}
//		// Construct block index object
//		indexNew := InsertBlockIndex(diskindex.GetBlockHash())
//		indexNew.Prev = InsertBlockIndex(&diskindex.hashPrev)
//		indexNew.Height = diskindex.Height
//		indexNew.File = diskindex.File
//		indexNew.DataPos = diskindex.DataPos
//		indexNew.UndoPos = diskindex.UndoPos
//		indexNew.Header.Version = diskindex.Header.Version
//		indexNew.Header.MerkleRoot = diskindex.Header.MerkleRoot
//		indexNew.Header.Time = diskindex.Header.Time
//		indexNew.Header.Bits = diskindex.Header.Bits
//		indexNew.Header.Nonce = diskindex.Header.Nonce
//		indexNew.Status = diskindex.Status
//		indexNew.TxCount = diskindex.TxCount
//
//		var pow Pow
//		if pow.CheckProofOfWork(indexNew.GetBlockHash(), indexNew.Header.Bits, msg.ActiveNetParams) {
//			logs.Error("LoadBlockIndex(): CheckProofOfWork failed: %s", indexNew.ToString())
//			return false
//		}
//
//		cursor.Next()
//	}
//
//	return true
//}
//


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

