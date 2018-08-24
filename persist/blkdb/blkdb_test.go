package blkdb

import (
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"reflect"
	"testing"
)

func initBlockDB() {
	bc := &BlockTreeDBConfig{
		Do: &db.DBOption{
			FilePath:  "/tmp/dbtest",
			CacheSize: 1 << 20,
		},
	}

	InitBlockTreeDB(bc)
}

func TestBlockDB(t *testing.T) {
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
