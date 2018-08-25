package blkdb

import (
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"reflect"
	"testing"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chainparams"
)

func initBlockDB() {
	bc := &BlockTreeDBConfig{
		Do: &db.DBOption{
			FilePath:  "/Users/wolf4j/Library/Application Support/Coper/blocks/index",
			CacheSize: 1 << 20,
		},
	}

	InitBlockTreeDB(bc)
}

func TestWRTxIndex(t *testing.T) {
	initBlockDB()

	// test TxIndex && init block pos
	dbpos := block.NewDiskBlockPos(12, 12)
	//init tx pos
	dtp := block.NewDiskTxPos(dbpos, 1)
	wantVal := &block.DiskTxPos{
		BlockIn:    dbpos,
		TxOffsetIn: 1,
	}
	txindexs := make(map[util.Hash]block.DiskTxPos)
	h := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d011")
	txindexs[*h] = *dtp

	//test Write and Read TxIndex
	err := GetInstance().WriteTxIndex(txindexs)
	if err != nil {
		t.Error("read failed")
	}
	txpos, err := GetInstance().ReadTxIndex(h)
	if err != nil {
		t.Error("read failed")
	}

	if !reflect.DeepEqual(wantVal, txpos) {
		t.Errorf("the wantVal not equal except value: %v:%v", wantVal, txpos)
	}
}

func TestWriteFlag(t *testing.T) {
	initBlockDB()
	//test flag: value is false
	err := GetInstance().WriteFlag("b", false)
	if err != nil {
		t.Errorf("write flag failed:%v", err)
	}
	res := GetInstance().ReadFlag("b")
	if res != true {
		t.Errorf("the flag should is true:%v", res)
	}

	//test flag: value is true
	err = GetInstance().WriteFlag("b", true)
	if err != nil {
		t.Errorf("write flag failed:%v", err)
	}
	res2 := GetInstance().ReadFlag("b")
	if res2 != false {
		t.Errorf("the flag should is false:%v", res2)
	}
}

func TestWriteReindexing(t *testing.T) {
	initBlockDB()
	//test write reindex: reindexing value is true
	err := GetInstance().WriteReindexing(true)
	if err != nil {
		t.Errorf("write the index failed:%v", err)
	}
	rr := GetInstance().ReadReindexing()
	if rr != true {
		t.Errorf("the reindexing should is true:%v", rr)
	}

	//test write reindex: reindexing value is false
	err = GetInstance().WriteReindexing(false)
	if err != nil {
		t.Errorf("write the index failed:%v", err)
	}
	rr1 := GetInstance().ReadReindexing()
	if rr1 != false {
		t.Errorf("the reindexing should is false:%v", rr1)
	}
}

func TestReadBlockFileInfo(t *testing.T) {
	initBlockDB()
	//the block file info exist
	bfi, err := GetInstance().ReadBlockFileInfo(0)
	if err != nil {
		t.Error("read block file info<000> failed.")
	}
	log.Info("blockFileInfo value is:%v", bfi)
	if bfi == nil {
		t.Errorf("the block fileInfo not equal nil:%v", bfi)
	}

	//the block file info not exist
	bfi, err = GetInstance().ReadBlockFileInfo(1111)
	if err == nil {
		t.Error("read block file info<1111> failed.")
	}
	if bfi != nil {
		t.Error("error")
	}
}

func TestReadLastBlockFile(t *testing.T) {
	initBlockDB()
	lastFile, err := GetInstance().ReadLastBlockFile()
	if err != nil {
		t.Error("read last block file failed")
	}

	bfi, err := GetInstance().ReadBlockFileInfo(lastFile)
	if err != nil {
		t.Error("read last block fileInfo failed")
	}
	log.Info("last blockFileInfo value is:%v", bfi)
	if bfi == nil {
		t.Errorf("the last blockFileInfo not equal nil, the value is:%v", bfi)
	}
}

func TestLoadBlockIndexGuts(t *testing.T) {
	initBlockDB()
	blkidxMap := make(map[util.Hash]*blockindex.BlockIndex)

	ret := GetInstance().LoadBlockIndexGuts(blkidxMap, chainparams.ActiveNetParams)
	log.Info("the blockIndexMap value is:%v", blkidxMap)
	if !ret {
		t.Error("load block index guts failed, please check.")
	}
}

func TestWriteBatchSync(t *testing.T) {
	initBlockDB()
	blkfi := make(map[int32]*block.BlockFileInfo)
	blkHeader := block.NewBlockHeader()
	idx := blockindex.NewBlockIndex(blkHeader)
	idxs := make([]*blockindex.BlockIndex, 0, 10)
	idxs = append(idxs, idx)
	err := GetInstance().WriteBatchSync(blkfi, 0, idxs)
	if err != nil {
		t.Errorf("write blockFileInfo failed.")
	}

	bfi1 := make(map[int32]*block.BlockFileInfo)
	fi := block.NewBlockFileInfo()
	fi.UndoSize = 0
	fi.Size = 1
	fi.Blocks = 1
	fi.HeightFirst = 1
	fi.HeightLast = 2

	bfi1[1] = fi

	//init block header
	blkHeader1 := block.NewBlockHeader()
	blkHeader.Time = uint32(1534822771)
	blkHeader.Version = 536870912
	blkHeader.Bits = 486604799
	preHash := util.HashFromString("00000000000001bcd6b635a1249dfbe76c0d001592a7219a36cd9bbd002c7238")
	merkleRoot := util.HashFromString("7e814211a7de289a490380c0c20353e0fd4e62bf55a05b38e1628e0ea0b4fd3d")
	blkHeader.HashPrevBlock = *preHash
	blkHeader.Nonce = 1391785674
	blkHeader.MerkleRoot = *merkleRoot
	//init block index
	blkidx := blockindex.NewBlockIndex(blkHeader1)
	blkidxs := make([]*blockindex.BlockIndex, 0, 10)
	blkidxs = append(blkidxs, blkidx)
	err = GetInstance().WriteBatchSync(bfi1, 0, blkidxs)
	if err != nil {
		t.Errorf("write blockFileInfo failed.")
	}
}
