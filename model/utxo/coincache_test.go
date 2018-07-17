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

	coin1 := necm.cacheCoins[outpoint1]
	coin1 = &Coin{
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

	coin2 := necm.cacheCoins[outpoint2]
	coin2 = &Coin{
		txOut:         *txout1,
		height:        100012,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

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

	necm.AddCoin(&outpoint1, coin1)
	necm.AddCoin(&outpoint2, coin2)

	c1 := necm.GetCoin(&outpoint1)
	if !reflect.DeepEqual(c1, coin1) {
		t.Error("the coin1 should equal get coin value.")
	}

	c2 := necm.GetCoin(&outpoint2)
	if !reflect.DeepEqual(c2, coin2) {
		t.Error("the coin1 should equal get coin value.")
	}

	necm.Flush(*hash1)

	//err1 := utxoTip.UpdateCoins(necm, hash1)
	//if err1 != nil {
	//	t.Errorf("update coins failed,the error is :%v", err1)
	//}

	if !reflect.DeepEqual(utxoTip.GetCoin(&outpoint1), coin1) {
		t.Error("the lru cache shouldn't equal cache coins of coinmap, the cache coin of map is equal nil")
	}

	if !reflect.DeepEqual(utxoTip.GetCoin(&outpoint2), coin2) {
		t.Error("the lru cache shouldn't equal cache coins of coinmap, the cache coin of map is equal nil")
	}

	if utxoTip.HaveCoin(&outpoint2) == false {
		t.Error("the cache should have coin, please check")
	}

	//flush
	necm.SetBestBlock(*hash1)
	//necm.Flush(*hash1)
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

	hash3 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d010")
	outpoint3 := outpoint.OutPoint{Hash: *hash3, Index: 0}

	script3 := script.NewScriptRaw([]byte{opcodes.OP_13, opcodes.OP_EQUAL})
	txout3 := txout.NewTxOut(3, script3)

	coin3 := necm.cacheCoins[outpoint3]
	coin3 = &Coin{
		txOut:         *txout3,
		height:        10000,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

	necm.AddCoin(&outpoint3, coin3)

	//no flush, get best block hash is hash1 ,when use flush,get best block hash is hash2.
	necm.SetBestBlock(*hash3)
	//necm.Flush(*hash3)
	hash4, err4 := cvt.GetBestBlock()
	if err4 != nil {
		panic("get best block failed..")
	}

	if hash4 == *hash3 {
		t.Error("get best block failed...")
	}

	if cvt.GetCacheSize() != 2 {
		t.Error("the cache size is 2, please check...")
	}
}
