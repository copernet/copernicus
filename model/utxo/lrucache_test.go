package utxo

import (
	"testing"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/txout"
)

func TestLRUCache(t *testing.T) {
	uc := &UtxoConfig{Do: &db.DBOption{
		FilePath:  "/tmp/dbtest",
		CacheSize: 1 << 20,
	},}

	InitUtxoLruTip(uc)

	necm := NewEmptyCoinsMap()
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
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

	necm.AddCoin(&outpoint1, necm.cacheCoins[outpoint1])

	ok := necm.Flush(*hash1)
	if !ok {
		t.Error("flush failed....", )
	}

	b := utxoTip.Flush()
	if !b {
		t.Error("flush error, the coin not flush to db..")
	}

	c := utxoTip.GetCoin(&outpoint1)
	if c == nil  {
		t.Error("get coin faild...")
	}

	hv := utxoTip.HaveCoin(&outpoint1)
	if !hv {
		t.Error("the cache not have coin, please check...")
	}
}
