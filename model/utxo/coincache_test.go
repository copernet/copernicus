package utxo

import (
	"testing"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/txout"
	"io/ioutil"
	"github.com/copernet/copernicus/persist/db"
	"reflect"
)

func TestCoinCache(t *testing.T) {
	necm := NewEmptyCoinsMap()

	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d011")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}

	script2 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout2 := txout.NewTxOut(3, script2)

	necm.cacheCoins[outpoint1] = &Coin{
		txOut:         *txout2,
		height:        10000,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

	hash2 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d012")
	outpoint2 := outpoint.OutPoint{Hash: *hash2, Index: 0}

	script1 := script.NewEmptyScript()
	txout1 := txout.NewTxOut(2, script1)

	necm.cacheCoins[outpoint2] = &Coin{
		txOut:         *txout1,
		height:        100012,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

	necm.AddCoin(&outpoint1, necm.cacheCoins[outpoint1])
	necm.AddCoin(&outpoint2, necm.cacheCoins[outpoint2])

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

	necm.Flush(*hash1)

	err1 := utxoTip.UpdateCoins(necm, hash1)
	if err1 != nil {
		t.Errorf("update coins failed,the error is :%v", err1)
	}

	if reflect.DeepEqual(utxoTip.GetCoin(&outpoint1), necm.cacheCoins[outpoint1]) {
		t.Error("the lru cache shouldn't equal cache coins of coinmap, the cache coin of map is equal nil")
	}

	if utxoTip.HaveCoin(&outpoint2) == false {
		t.Error("the cache should have coin, please check")
	}

	//flush
	necm.SetBestBlock(*hash1)
	necm.Flush(*hash1)
	cvt := GetUtxoCacheInstance()
	hash0, err0 := cvt.GetBestBlock()
	if err0 != nil {
		panic("get best block failed...")
	}
	if hash0 != *hash1 {
		t.Error("get best block failed...")
	}
	if cvt.GetCacheSize() != 2 {
		t.Error("the cache size is 2, please check...")
	}

	//no flush, get best block hash is hash1 ,when use flush,get best block hash is hash2.
	necm.SetBestBlock(*hash2)
	//necm.Flush(*hash2)
	gulci := GetUtxoCacheInstance()
	hash3, err3 := gulci.GetBestBlock()
	if err3 != nil {
		panic("get best block failed..")
	}
	if hash3 != *hash1 {
		t.Error("get best block failed...")
	}
	if gulci.GetCacheSize() != 2 {
		t.Error("the cache size is 2, please check...")
	}

}
