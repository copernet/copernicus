package utxo

import (
	"testing"
	"github.com/copernet/copernicus/persist/db"
	"io/ioutil"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/outpoint"
	"fmt"
	"github.com/davecgh/go-spew/spew"
)

func TestCoinsDB(t *testing.T) {
	path, err := ioutil.TempDir("", "coindbtest")
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}

	dbo := db.DBOption{
		FilePath:       path,
		CacheSize:      1 << 20,
		Wipe:           false,
		DontObfuscate:  false,
		ForceCompactdb: false,
	}
	cdb := NewCoinsDB(&dbo)

	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}
	hc := cdb.HaveCoin(&outpoint1)
	fmt.Printf("wtether the db have coin : %v\n", hc)

	es := cdb.EstimateSize()
	fmt.Printf("Estimate size value is :%v\n ", es)

	c, err1 := cdb.GetCoin(&outpoint1)
	if err1 != nil {
		spew.Dump("panic")
	}
	fmt.Printf("get coin value is : %v\n", c)
	//hash, err := cdb.GetBestBlock()
	//if err != nil {
	//	t.Error("get best block error")
	//}
	//
	//t.Logf("get best block hash is: %v \n", hash)
}
