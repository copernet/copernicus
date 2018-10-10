package main

import (
	"fmt"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/logic/lreindex"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/persist/disk"
)

func appInitMain(args []string) {
	conf.Cfg = conf.InitConfig(args)

	if conf.Cfg.P2PNet.TestNet {
		model.SetTestNetParams()
	} else if conf.Cfg.P2PNet.RegTest {
		model.SetRegTestParams()
	}

	fmt.Println("Current data dir:\033[0;32m", conf.DataDir, "\033[0m")

	//init log
	log.Init()

	// Init UTXO DB
	utxoDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/chainstate",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
	utxo.InitUtxoLruTip(&utxoConfig)

	chain.InitGlobalChain()

	// Init blocktree DB
	blkDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	persist.InitPersistGlobal()

	// Load blockindex DB
	lblockindex.LoadBlockIndexDB()

	// when reindexing, we reuse the genesis block already on the disk
	if !conf.Cfg.Reindex {
		lchain.InitGenesisChain()
	}

	mempool.InitMempool()
	crypto.InitSecp256()

	if conf.Cfg.Reindex {
		disk.CleanupBlockRevFiles()
		err := lreindex.Reindex()
		if err != nil {
			log.Error("fatal error occurred when reindex: %s, will shutdown!", err)
			shutdownRequestChannel <- struct{}{}
		}

		gChain := chain.GetInstance()
		log.Info(`
----After reindex----
    chain height: <%d>
    chain's index map count: %d
    tip block index: %s
---------------------`, gChain.Height(), gChain.IndexMapSize(), gChain.Tip().String())

		if chain.GetInstance().Genesis() == nil {
			log.Warn("after reindex, genesis block is not init, reindex may not worked, init genesis block now")
			lchain.InitGenesisChain()
		}
		fmt.Println("reindex finish")
	}
}
