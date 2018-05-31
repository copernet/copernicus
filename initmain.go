package main

import (
	"github.com/btcboost/copernicus/logic/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/persist/blkdb"
	"github.com/btcboost/copernicus/persist/db"
	"github.com/btcboost/copernicus/persist/global"
)

func appInitMain() {
	config := utxo.UtxoConfig{Do: &db.DBOption{CacheSize: 10000}}
	utxo.InitUtxoLruTip(&config)
	chain.InitGlobalChain(nil)
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: &db.DBOption{CacheSize: 10000}}
	blkdb.InitBlockTreDB(&blkdbCfg)
	global.InitPersistGlobal()
	blockindex.LoadBlockIndexDB()
}
