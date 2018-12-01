package utxo

import (
	"fmt"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGetBestBlock(t *testing.T) {
	conf.Cfg = conf.InitConfig([]string{})
	testDataDir, err := conf.SetUnitTestDataDir(conf.Cfg)
	if err != nil {
		fmt.Print("init test directory failed")
		os.Exit(1)
	}
	defer os.RemoveAll(testDataDir)
	uc := &UtxoConfig{Do: &db.DBOption{
		FilePath:  conf.Cfg.DataDir,
		CacheSize: 1 << 20,
	}}
	InitUtxoLruTip(uc)

	rhash, err := GetUtxoCacheInstance().GetBestBlock()
	assert.EqualError(t, err, "leveldb: not found")
	assert.Equal(t, util.Hash{}, rhash)

	necm := NewEmptyCoinsMap()
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}

	script1 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout1 := txout.NewTxOut(3, script1)

	coin1 := &Coin{
		txOut:         *txout1,
		height:        10000,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

	necm.AddCoin(&outpoint1, coin1, true)

	err = GetUtxoCacheInstance().UpdateCoins(necm, hash1)
	assert.Nil(t, err, "update coins failed")
	ok := GetUtxoCacheInstance().Flush()
	assert.True(t, ok)

	rhash, err = GetUtxoCacheInstance().GetBestBlock()
	t.Logf("best block: %s", rhash.String())
	assert.Nil(t, err)
	assert.NotEqual(t, util.Hash{}, rhash)
}

func TestLRUCache(t *testing.T) {

	conf.Cfg = conf.InitConfig([]string{})
	testDataDir, err := conf.SetUnitTestDataDir(conf.Cfg)
	if err != nil {
		fmt.Print("init test directory failed")
		os.Exit(1)
	}
	defer os.RemoveAll(testDataDir)
	uc := &UtxoConfig{Do: &db.DBOption{
		FilePath:  conf.Cfg.DataDir,
		CacheSize: 1 << 20,
	}}
	InitUtxoLruTip(uc)

	necm := NewEmptyCoinsMap()
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}

	script1 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout1 := txout.NewTxOut(3, script1)

	coin1 := &Coin{
		txOut:         *txout1,
		height:        10000,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

	necm.AddCoin(&outpoint1, coin1, true)

	err = GetUtxoCacheInstance().UpdateCoins(necm, hash1)
	assert.Nil(t, err, "update coins failed")

	b := utxoTip.Flush()
	assert.True(t, b, "flush error, the coin not flush to db..")

	c := utxoTip.GetCoin(&outpoint1)
	if c == nil {
		t.Error("get coin faild...")
	}

	hv := utxoTip.HaveCoin(&outpoint1)
	if !hv {
		t.Error("the cache not have coin, please check...")
	}
}

func TestUpdateCoins(t *testing.T) {

	conf.Cfg = conf.InitConfig([]string{})
	testDataDir, err := conf.SetUnitTestDataDir(conf.Cfg)
	if err != nil {
		fmt.Print("init test directory failed")
		os.Exit(1)
	}
	defer os.RemoveAll(testDataDir)
	uc := &UtxoConfig{Do: &db.DBOption{
		FilePath:  conf.Cfg.DataDir,
		CacheSize: 1 << 20,
	}}
	InitUtxoLruTip(uc)

	necm := NewEmptyCoinsMap()
	hash := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0b7")
	outPoint := outpoint.OutPoint{Hash: *hash, Index: 0}

	scriptRaw := script.NewScriptRaw([]byte{opcodes.OP_12, opcodes.OP_EQUAL})
	txouts := txout.NewTxOut(3, scriptRaw)

	ncoin := &Coin{
		txOut:         *txouts,
		height:        1 << 20,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

	necm.AddCoin(&outPoint, ncoin, true)

	err = GetUtxoCacheInstance().UpdateCoins(necm, hash)
	assert.Nil(t, err, "flush failed!")

	c2 := utxoTip.GetCoin(&outPoint)
	assert.NotNil(t, c2, "get coin failed!")

	coinCopy := ncoin.DeepCopy()
	coinCopy.dirty = true
	necm.AddCoin(&outPoint, coinCopy, true)
	err = GetUtxoCacheInstance().UpdateCoins(necm, hash)
	assert.Nil(t, err, "update copy coin failed")

	coinCopy2 := coinCopy.DeepCopy()
	necm.AddCoin(&outPoint, coinCopy2, true)
	coinCopy2.Clear()
	coinCopy2.fresh = true
	err = GetUtxoCacheInstance().UpdateCoins(necm, hash)
	assert.Nil(t, err, "update copy coin failed")

	//------
	necm.UnCache(&outPoint)
	coinCopy3 := ncoin.DeepCopy()
	necm.AddCoin(&outPoint, coinCopy3, true)
	coinCopy3.fresh = true
	coinCopy3.dirty = true
	err = GetUtxoCacheInstance().UpdateCoins(necm, hash)
	assert.Nil(t, err, "update copy coin failed")

	usage := GetUtxoCacheInstance().DynamicMemoryUsage()
	assert.NotEmpty(t, usage)
}

func TestUpdateCoins2(t *testing.T) {

	conf.Cfg = conf.InitConfig([]string{})
	testDataDir, err := conf.SetUnitTestDataDir(conf.Cfg)
	if err != nil {
		fmt.Print("init test directory failed")
		os.Exit(1)
	}
	defer os.RemoveAll(testDataDir)
	uc := &UtxoConfig{Do: &db.DBOption{
		FilePath:  conf.Cfg.DataDir,
		CacheSize: 1 << 20,
	}}
	InitUtxoLruTip(uc)

	necm := NewEmptyCoinsMap()
	hash := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0b7")
	outPoint := outpoint.OutPoint{Hash: *hash, Index: 0}

	scriptRaw := script.NewScriptRaw([]byte{opcodes.OP_12, opcodes.OP_EQUAL})
	txouts := txout.NewTxOut(3, scriptRaw)

	ncoin := &Coin{
		txOut:         *txouts,
		height:        1 << 20,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         false,
	}

	necm.AddCoin(&outPoint, ncoin, true)
	err = GetUtxoCacheInstance().UpdateCoins(necm, hash)
	assert.Nil(t, err, "flush failed!")

	newCoin := ncoin.DeepCopy()
	necm.AddCoin(&outPoint, newCoin, true)
	newCoin.dirty = true
	newCoin.fresh = true
	err = GetUtxoCacheInstance().UpdateCoins(necm, hash)
	assert.Nil(t, err, "flush failed!")
}

func TestCoinsLruCache_GetCoinsDB(t *testing.T) {
	cdb := GetUtxoCacheInstance().(*CoinsLruCache).GetCoinsDB()
	assert.NotNil(t, cdb)
}

func TestAccessByTxid(t *testing.T) {
	conf.Cfg = conf.InitConfig([]string{})
	testDataDir, err := conf.SetUnitTestDataDir(conf.Cfg)
	if err != nil {
		fmt.Print("init test directory failed")
		os.Exit(1)
	}
	defer os.RemoveAll(testDataDir)
	uc := &UtxoConfig{Do: &db.DBOption{
		FilePath:  conf.Cfg.DataDir,
		CacheSize: 1 << 20,
	}}
	InitUtxoLruTip(uc)

	necm := NewEmptyCoinsMap()
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}
	outpoint2 := outpoint.OutPoint{Hash: *hash1, Index: 1}

	script1 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout1 := txout.NewTxOut(3, script1)

	coin1 := NewFreshCoin(txout1, 10000, false)

	necm.AddCoin(&outpoint1, coin1, true)
	necm.AddCoin(&outpoint2, coin1, true)

	err = GetUtxoCacheInstance().UpdateCoins(necm, hash1)
	assert.Nil(t, err, "update coins failed")

	b := GetUtxoCacheInstance().Flush()
	assert.True(t, b, "flush error, the coin not flush to db..")

	rCoin := GetUtxoCacheInstance().AccessByTxID(hash1)

	unspendCoinFromDB := NewFreshCoin(txout1, 10000, false)
	assert.Equal(t, unspendCoinFromDB.GetTxOut(), rCoin.GetTxOut())

	hashUnKnown := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a0")
	rCoin = GetUtxoCacheInstance().AccessByTxID(hashUnKnown)
	assert.Nil(t, rCoin)
}
