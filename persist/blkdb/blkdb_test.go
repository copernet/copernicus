package blkdb

import (
	"testing"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/util"
	"reflect"
)

func initBlockDB() {
	bc := &BlockTreeDBConfig{Do: &db.DBOption{
		FilePath:  "/tmp/dbtest",
		CacheSize: 1 << 20,
	}}

	InitBlockTreDB(bc)
}

func TestBlockDB(t *testing.T) {
	initBlockDB()

	// test TxIndex
	//init block pos
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

	//test block file info
	bfi, err := GetInstance().ReadBlockFileInfo(12)
	if err != nil {
		t.Error("read failed")
	}
	if bfi != nil {
		t.Errorf("the ReadBlockFileInfo faild: %v", bfi)
	}

	//test flag: value is false
	err1 := GetInstance().WriteFlag("b", false)
	if err1 != nil {
		t.Errorf("write flag failed:%v", err1)
	}
	res := GetInstance().ReadFlag("b")
	if res != true {
		t.Errorf("the flag should is true:%v", res)
	}

	//test flag: value is true
	err2 := GetInstance().WriteFlag("b", true)
	if err1 != nil {
		t.Errorf("write flag failed:%v", err2)
	}
	res2 := GetInstance().ReadFlag("b")
	if res2 != false {
		t.Errorf("the flag should is false:%v", res2)
	}

	//test write reindex: reindexing value is true
	err3 := GetInstance().WriteReindexing(true)
	if err3 != nil {
		t.Errorf("write the index failed:%v", err3)
	}
	rr := GetInstance().ReadReindexing()
	if rr != true {
		t.Errorf("the reindexing should is true:%v", rr)
	}

	//test write reindex: reindexing value is false
	err4 := GetInstance().WriteReindexing(false)
	if err4 != nil {
		t.Errorf("write the index failed:%v", err4)
	}
	rr1 := GetInstance().ReadReindexing()
	if rr1 != false {
		t.Errorf("the reindexing should is false:%v", rr1)
	}

}
