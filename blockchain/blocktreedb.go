package blockchain

import (
	"bytes"
	"strconv"

	"encoding/binary"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/logger"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

type BlockTreeDB struct {
	dbw *database.DBWrapper
}

func (blockTreeDB *BlockTreeDB) ReadBlockFileInfo(file int) *BlockFileInfo {
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, utxo.DbBlockFiles)
	tmp = strconv.AppendInt(tmp, int64(file), 10)
	buf, err := blockTreeDB.dbw.Read(tmp)
	if err != nil {
		panic("read failed ....")
	}
	bufs := bytes.NewBuffer(buf)

	blockFileInfo, err := DeserializeBlockFileInfo(bufs)
	if err != nil {
		return nil
	}
	return blockFileInfo
}

type writeTxIndex struct {
	hash utils.Hash
	pos  core.DiskTxPos
}

func (blockTreeDB *BlockTreeDB) WriteTxIndex(ect []*writeTxIndex) error {
	var batch *database.BatchWrapper
	for _, v := range ect {
		key := make([]byte, 0, 100)
		key = append(key, utxo.DbTxIndex)
		key = append(key, v.hash[:]...)

		buf := bytes.NewBuffer(nil)
		if err := v.pos.SerializeDiskTxPos(buf); err != nil {
			return err
		}
		batch.Write(key, buf.Bytes())
	}
	return blockTreeDB.dbw.WriteBatch(batch, false)
}

func (blockTreeDB *BlockTreeDB) ReadTxIndex(txid *utils.Hash) ([]byte, error) {
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, utxo.DbTxIndex)
	tmp = append(tmp, txid[:]...)
	return blockTreeDB.dbw.Read(tmp)
}

func (blockTreeDB *BlockTreeDB) WriteReindexing(reindexing bool) error {
	if reindexing {
		return blockTreeDB.dbw.Write([]byte{utxo.DbReindexFlag}, []byte{1}, false)
	}
	return blockTreeDB.dbw.Erase([]byte{utxo.DbReindexFlag}, false)
}

func (blockTreeDB *BlockTreeDB) ReadReindexing(reindexing bool) bool {
	reindexing = blockTreeDB.dbw.Exists([]byte{utxo.DbReindexFlag})
	return true
}

type bkFileInfo struct {
	i       int
	bkfInfo *BlockFileInfo
}

func IntToBytes(i int) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func (blockTreeDB *BlockTreeDB) WriteBatchSync(fileInfo []*bkFileInfo, latFile int, blockIndexes []*core.BlockIndex) error {
	var batch *database.BatchWrapper
	for _, v := range fileInfo {
		tmp := make([]byte, 0, 100)
		tmp = append(tmp, utxo.DbLastBlock)
		tmp = strconv.AppendInt(tmp, int64(v.i), 10)
		buf := bytes.NewBuffer(nil)
		if err := v.bkfInfo.Serialize(buf); err != nil {
			return err
		}
		batch.Write(tmp, buf.Bytes())
	}
	file := IntToBytes(latFile)
	batch.Write([]byte{utxo.DbLastBlock}, file)

	for _, v := range blockIndexes {
		tmp := make([]byte, 0, 100)
		tmp = append(tmp, utxo.DbBlockIndex)
		buf := bytes.NewBuffer(nil)
		v.GetBlockHash().Serialize(buf)
		tmp = append(tmp, buf.Bytes()...)
		if err := NewDiskBlockIndex(v).Serialize(buf); err != nil {
			return err
		}
		batch.Write(tmp, buf.Bytes())
	}

	return blockTreeDB.dbw.WriteBatch(batch, true)
}

func (blockTreeDB *BlockTreeDB) LoadBlockIndexGuts(f func(hash *utils.Hash) *core.BlockIndex) bool {
	var cursor database.IterWrapper

	hash := utils.Hash{}
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, utxo.DbBlockIndex)
	tmp = append(tmp, hash[:]...)
	cursor.Seek(tmp)

	// Load mapBlockIndex
	for cursor.Valid() {
		//todo:boost::this_thread::interruption_point();
		type key struct {
			b    byte
			hash utils.Hash
		}
		k := cursor.GetKey()
		kk := key{}
		if k == nil || kk.b != utxo.DbBlockIndex {
			break
		}

		var diskindex DiskBlockIndex
		val := cursor.GetVal()
		if val == nil {
			return logger.ErrorLog("LoadBlockIndex() : failed to read value")
		}
		// Construct block index object
		indexNew := InsertBlockIndex(diskindex.GetBlockHash())
		indexNew.Prev = InsertBlockIndex(&diskindex.hashPrev)
		indexNew.Height = diskindex.Height
		indexNew.File = diskindex.File
		indexNew.DataPos = diskindex.DataPos
		indexNew.UndoPos = diskindex.UndoPos
		indexNew.Version = diskindex.Version
		indexNew.MerkleRoot = diskindex.MerkleRoot
		indexNew.Time = diskindex.Time
		indexNew.Bits = diskindex.Bits
		indexNew.Nonce = diskindex.Nonce
		indexNew.Status = diskindex.Status
		indexNew.TxCount = diskindex.TxCount

		var pow Pow
		if pow.CheckProofOfWork(indexNew.GetBlockHash(), indexNew.Bits, msg.ActiveNetParams) {
			return logger.ErrorLog("LoadBlockIndex(): CheckProofOfWork failed: %s", indexNew.ToString())
		}

		cursor.Next()
	}

	return true
}

func (blockTreeDB *BlockTreeDB) ReadLastBlockFile() ([]byte, error) {
	return blockTreeDB.dbw.Read([]byte{utxo.DbLastBlock})
}

func (blockTreeDB *BlockTreeDB) WriteFlag(name string, value bool) error {
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, utxo.DbFlag)
	tmp = append(tmp, name...)
	if !value {
		return blockTreeDB.dbw.Write(tmp, []byte{'1'}, value)
	}
	return blockTreeDB.dbw.Write(tmp, []byte{'0'}, value)
}

func (blockTreeDB *BlockTreeDB) ReadFlag(name string) bool {
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, utxo.DbFlag)
	tmp = append(tmp, name...)
	b, err := blockTreeDB.dbw.Read(tmp)

	if b[0] == '1' && err == nil {
		return true
	}
	return false
}

func NewBlockTreeDB(do *database.DBOption) *BlockTreeDB {
	if do == nil {
		return nil
	}

	dbw, err := database.NewDBWrapper(&database.DBOption{
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
