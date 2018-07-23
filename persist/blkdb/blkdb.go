package blkdb

import (
	"bytes"
	"fmt"

	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/persist/db"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/astaxie/beego/logs"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chainparams"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/util"
)

type BlockTreeDB struct {
	dbw *db.DBWrapper
}

var blockTreeDb *BlockTreeDB

type BlockTreeDBConfig struct {
	Do *db.DBOption
}

func InitBlockTreDB(uc *BlockTreeDBConfig) {
	// fmt.Printf("InitBlockTreDB processing ....%v",uc)
	blockTreeDb = newBlockTreeDB(uc.Do)

}

func GetInstance() *BlockTreeDB {
	if blockTreeDb == nil {
		panic("blockTreeDb has not init !!!")
	}
	return blockTreeDb
}
func newBlockTreeDB(do *db.DBOption) *BlockTreeDB {
	if do == nil {
		return nil
	}
	dbw, err := db.NewDBWrapper(do)
	if err != nil {
		panic("init DBWrapper failed...")
	}
	return &BlockTreeDB{
		dbw: dbw,
	}
}

func (blockTreeDB *BlockTreeDB) ReadBlockFileInfo(file int32) (*block.BlockFileInfo, error) {
	log.Debug("file======%#v", file)
	keyBuf := bytes.NewBuffer(nil)
	keyBuf.Write([]byte{db.DbBlockFiles})
	util.WriteElements(keyBuf, uint64(file))
	vbytes, err := blockTreeDB.dbw.Read(keyBuf.Bytes())
	if err == leveldb.ErrNotFound {
		return nil, nil
	}
	log.Debug("file======%#v", file)

	if err != nil {
		log.Error("ReadBlockFileInfo err: %#v", err.Error())
		panic("read failed ....")
	}
	bufs := bytes.NewBuffer(vbytes)
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

func (blockTreeDB *BlockTreeDB) ReadLastBlockFile() (int32, error) {
	data, err := blockTreeDB.dbw.Read([]byte{db.DbLastBlock})
	if err != nil {
		return 0, err
	}
	buf := bytes.NewBuffer(data)
	var lastFile int32
	err = util.ReadElements(buf, &lastFile)
	return lastFile, err
}

func (blockTreeDB *BlockTreeDB) WriteMaxBlockFile(file int) error {
	vbuf := bytes.NewBuffer(nil)
	util.WriteElements(vbuf, uint64(file))
	return blockTreeDB.dbw.Write([]byte{db.DbMaxBlock}, vbuf.Bytes(), false)
}

func (blockTreeDB *BlockTreeDB) ReadMaxBlockFile() (int, error) {
	data, err := blockTreeDB.dbw.Read([]byte{db.DbMaxBlock})
	if err != nil {
		return -2, err
	}
	buf := bytes.NewBuffer(data)
	var lastFile = -2
	err = util.ReadElements(buf, &lastFile)
	return lastFile, err
}

func (blockTreeDB *BlockTreeDB) WriteBatchSync(fileInfoList []*block.BlockFileInfo, lastFile int, blockIndexes []*blockindex.BlockIndex) error {
	batch := db.NewBatchWrapper(blockTreeDB.dbw)
	keytmp := make([]byte, 0, 100)
	valuetmp := make([]byte, 0, 100)
	keyBuf := bytes.NewBuffer(keytmp)
	valueBuf := bytes.NewBuffer(valuetmp)

	for _, v := range fileInfoList {
		keyBuf.Reset()
		valueBuf.Reset()
		keyBuf.Write([]byte{db.DbBlockFiles})
		util.WriteElements(keyBuf, uint64(v.GetIndex()))
		if err := v.Serialize(valueBuf); err != nil {
			return err
		}
		batch.Write(keyBuf.Bytes(), valueBuf.Bytes())

	}
	valueBuf.Reset()
	util.WriteElements(valueBuf, uint64(lastFile))
	batch.Write([]byte{db.DbLastBlock}, valueBuf.Bytes())

	for _, v := range blockIndexes {
		keyBuf.Reset()
		valueBuf.Reset()
		keyBuf.Write([]byte{db.DbBlockIndex})
		v.GetBlockHash().Serialize(keyBuf)
		if err := v.Serialize(valueBuf); err != nil {
			return err
		}
		batch.Write(keyBuf.Bytes(), valueBuf.Bytes())
	}

	err := blockTreeDB.dbw.WriteBatch(batch, true)
	if true && err != nil {
		lastFile, e := blockTreeDB.ReadLastBlockFile()
		fmt.Println(lastFile, e)

	}
	return err
}

func (blockTreeDB *BlockTreeDB) ReadTxIndex(txid *util.Hash) (*block.DiskTxPos, error) {
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, db.DbTxIndex)
	tmp = append(tmp, txid[:]...)
	vdata, err := blockTreeDB.dbw.Read(tmp)
	if err != nil {
		log.Error("Error: ReadTxIndex======%#v", err)
		panic("Error: ReadTxIndex======")
	}
	if vdata == nil {
		return nil, nil
	}
	dtp := block.NewDiskTxPos(nil, 0)
	err = dtp.Unserialize(bytes.NewBuffer(vdata))
	return dtp, err

}

func (blockTreeDB *BlockTreeDB) WriteTxIndex(txIndexes map[util.Hash]block.DiskTxPos) error {
	var batch = db.NewBatchWrapper(blockTreeDB.dbw)
	keytmp := make([]byte, 0, 100)
	valuetmp := make([]byte, 0, 100)
	keyBuf := bytes.NewBuffer(keytmp)
	valueBuf := bytes.NewBuffer(valuetmp)
	for k, v := range txIndexes {
		keyBuf.Reset()
		valueBuf.Reset()
		keyBuf.Write([]byte{db.DbTxIndex})
		keyBuf.Write(k[:])
		if err := v.Serialize(valueBuf); err != nil {
			return err
		}
		batch.Write(keyBuf.Bytes(), valueBuf.Bytes())
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

func (blockTreeDB *BlockTreeDB) LoadBlockIndexGuts(GlobalBlockIndexMap map[util.Hash]*blockindex.BlockIndex, params *chainparams.BitcoinParams) bool {
	// todo for iter and check key、 pow
	cursor := blockTreeDB.dbw.Iterator()
	defer cursor.Close()
	hash := util.Hash{}
	tmp := make([]byte, 0, 100)
	tmp = append(tmp, db.DbBlockIndex)
	tmp = append(tmp, hash[:]...)
	cursor.Seek(tmp)

	// Load mapBlockIndex
	for cursor.Valid() {
		k := cursor.GetKey()
		if k == nil || k[0] != db.DbBlockIndex {
			break
		}

		var bi = blockindex.NewBlockIndex(block.NewBlockHeader())
		val := cursor.GetVal()
		if val == nil {
			logs.Error("LoadBlockIndex() : failed to read value")
			return false
		}
		bi.Unserialize(bytes.NewBuffer(val))
		if bi.TxCount == 0 {
			fmt.Println("err")
			blockTreeDB.dbw.Erase(k, true)
			cursor.Next()
			continue
		}
		newIndex := InsertBlockIndex(*bi.GetBlockHash(), GlobalBlockIndexMap)
		if newIndex == nil {
			cursor.Next()
			continue
		}
		pre := InsertBlockIndex(bi.Header.HashPrevBlock, GlobalBlockIndexMap)
		newIndex.Prev = pre
		newIndex.SetBlockHash(*bi.GetBlockHash())
		newIndex.Height = bi.Height
		newIndex.File = bi.File
		newIndex.DataPos = bi.DataPos
		newIndex.UndoPos = bi.UndoPos
		newIndex.Header.Version = bi.Header.Version
		newIndex.Header.HashPrevBlock = bi.Header.HashPrevBlock
		newIndex.Header.MerkleRoot = bi.Header.MerkleRoot
		newIndex.Header.Time = bi.Header.Time
		newIndex.Header.Bits = bi.Header.Bits
		newIndex.Header.Nonce = bi.Header.Nonce
		newIndex.Status = bi.Status
		newIndex.TxCount = bi.TxCount

		if !new(pow.Pow).CheckProofOfWork(bi.GetBlockHash(), bi.Header.Bits, params) {
			logs.Error("LoadBlockIndex(): CheckProofOfWork failed: %s", bi.String())
			return false
		}
		cursor.Next()
	}
	return true
}

func InsertBlockIndex(hash util.Hash, GlobalBlockIndexMap map[util.Hash]*blockindex.BlockIndex) *blockindex.BlockIndex {

	if i, ok := GlobalBlockIndexMap[hash]; ok {
		return i
	}
	if hash.IsNull() {
		return nil
	}
	var bi = blockindex.NewBlockIndex(block.NewBlockHeader())

	GlobalBlockIndexMap[hash] = bi

	return bi

}
