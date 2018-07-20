package utxo

import (
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"io/ioutil"
	"testing"
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

	uc := &UtxoConfig{
		&dbo,
	}
	InitUtxoLruTip(uc)
	chain.InitGlobalChain(nil)
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}

	if utxoTip.HaveCoin(&outpoint1) == true {
		t.Error("the db not have coin")
	}

	if utxoTip.DynamicMemoryUsage() < 0 {
		t.Error("memory can not less than zero,please check..")
	}

	if utxoTip.GetCoin(&outpoint1) != nil {
		t.Error("the db not have coin, so the coin is nil.")
	}

	utxoTip.SetBestBlock(*hash1)
	utxoTip.Flush()

	hash2, err2 := utxoTip.GetBestBlock()
	if err2 != nil {
		panic("get best block failed")
	}

	if hash2 != *hash1 {
		t.Error("the best block hash should equal hash1")
	}

}
