package main

import (
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/blockindex"
	lchain "github.com/copernet/copernicus/logic/chain"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/persist/global"
)

func appInitMain() {
	log.Init()
	config := utxo.UtxoConfig{Do: &db.DBOption{FilePath: conf.GetDataPath() + "/chainstate", CacheSize: (1 << 20) * 8}}
	utxo.InitUtxoLruTip(&config)
	chain.InitGlobalChain()
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: &db.DBOption{FilePath: conf.GetDataPath() + "/blocks/index", CacheSize: (1 << 20) * 8}}
	blkdb.InitBlockTreDB(&blkdbCfg)
	global.InitPersistGlobal()
	blockindex.LoadBlockIndexDB()
	// gChain := chain.GetInstance()
	// mostWorkChain := gChain.FindMostWorkChain()
	// if mostWorkChain != gChain.Tip(){
	// 	lchain.ActivateBestChainStep()
	// }
	lchain.InitGenesisChain()
	mempool.InitMempool()
	crypto.InitSecp256()
}
