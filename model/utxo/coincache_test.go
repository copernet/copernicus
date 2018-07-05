package utxo

import (
	"testing"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/txout"
	"fmt"
	"io/ioutil"
	"github.com/copernet/copernicus/persist/db"
	"github.com/davecgh/go-spew/spew"
)

func TestCoinCache(t *testing.T) {
	fmt.Println("========create coinmap cache=========")
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
		fresh:         false,
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
		dirty:         true,
		fresh:         false,
	}

	necm.AddCoin(&outpoint1, necm.cacheCoins[outpoint1])
	necm.AddCoin(&outpoint2, necm.cacheCoins[outpoint2])

	//flush
	necm.SetBestBlock(*hash1)
	necm.Flush(*hash1)
	cvt := GetUtxoLruCacheInstance()
	h := cvt.GetBestBlock()
	s := cvt.GetCacheSize()//2
	spew.Dump("get best block hash is :%v, get cache size value is :%v \n", h, s)

	//no flush, get best block hash is hash1 ,when use flush,get best block hash is hash2.
	necm.SetBestBlock(*hash2)
	//necm.Flush(*hash2)
	gulci := GetUtxoLruCacheInstance()
	h1 := gulci.GetBestBlock()
	s1 := gulci.GetCacheSize()//2

	spew.Dump("get best block hash is :%v, get cache size value is :%v \n", h1, s1)

	fmt.Println("=============create lru cache============")

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
}
