package main

import (
	"github.com/btcboost/copernicus/logic/blockindex"
	lchain "github.com/btcboost/copernicus/logic/chain"
	
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/persist/blkdb"
	"github.com/btcboost/copernicus/persist/db"
	"github.com/btcboost/copernicus/persist/global"
)

func appInitMain() {
	config := utxo.UtxoConfig{Do: &db.DBOption{CacheSize: 48}}
	utxo.InitUtxoLruTip(&config)
	chain.InitGlobalChain(nil)
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: &db.DBOption{CacheSize: 48}}
	blkdb.InitBlockTreDB(&blkdbCfg)
	global.InitPersistGlobal()
	blockindex.LoadBlockIndexDB()
	// gChain := chain.GetInstance()
	// mostWorkChain := gChain.FindMostWorkChain()
	// if mostWorkChain != gChain.Tip(){
	// 	lchain.ActivateBestChainStep()
	// }
	lchain.InitGenesisChain()
}
