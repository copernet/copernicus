package lutxo

import (
	"fmt"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestAccessByTxid(t *testing.T) {
	conf.Cfg = conf.InitConfig([]string{})
	testDataDir, err := conf.SetUnitTestDataDir(conf.Cfg)
	if err != nil {
		fmt.Print("init test directory failed")
		os.Exit(1)
	}
	defer os.RemoveAll(testDataDir)
	uc := &utxo.UtxoConfig{Do: &db.DBOption{
		FilePath:  conf.Cfg.DataDir,
		CacheSize: 1 << 20,
	}}
	utxo.InitUtxoLruTip(uc)

	necm := utxo.NewEmptyCoinsMap()
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}
	outpoint2 := outpoint.OutPoint{Hash: *hash1, Index: 1}

	script1 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout1 := txout.NewTxOut(3, script1)

	coin1 := utxo.NewFreshCoin(txout1, 10000, false)

	necm.AddCoin(&outpoint1, coin1, true)
	necm.AddCoin(&outpoint2, coin1, true)

	err = utxo.GetUtxoCacheInstance().UpdateCoins(necm, hash1)
	assert.Nil(t, err, "update coins failed")

	b := utxo.GetUtxoCacheInstance().Flush()
	assert.True(t, b, "flush error, the coin not flush to db..")

	rCoin := AccessByTxid(utxo.GetUtxoCacheInstance(), hash1)

	unspendCoinFromDB := utxo.NewFreshCoin(txout1, 10000, false)
	assert.Equal(t, unspendCoinFromDB.GetTxOut(), rCoin.GetTxOut())
}
