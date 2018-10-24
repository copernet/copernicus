package lchain

import (
	"encoding/json"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func initTestEnv(t *testing.T, args []string, initScriptVerify bool) (dirpath string, err error) {
	conf.Cfg = conf.InitConfig(args)

	unitTestDataDirPath, err := conf.SetUnitTestDataDir(conf.Cfg)
	t.Logf("test in temp dir: %s", unitTestDataDirPath)
	if err != nil {
		return "", err
	}

	if conf.Cfg.P2PNet.TestNet {
		model.SetTestNetParams()
	} else if conf.Cfg.P2PNet.RegTest {
		model.SetRegTestParams()
	}

	//init log
	logDir := filepath.Join(conf.DataDir, log.DefaultLogDirname)
	if !conf.FileExists(logDir) {
		err := os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	logConf := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
		Daily    bool   `json:"daily"`
	}{
		FileName: logDir + "/" + conf.Cfg.Log.FileName + ".log",
		Level:    log.GetLevel(conf.Cfg.Log.Level),
		Daily:    false,
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}
	log.Init(string(configuration))

	persist.InitPersistGlobal()

	// Init UTXO DB
	utxoDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/chainstate",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
	utxo.InitUtxoLruTip(&utxoConfig)

	// Init blocktree DB
	blkDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	chain.InitGlobalChain()
	tchain := chain.GetInstance()
	*tchain = *chain.NewChain()

	// Load blockindex DB
	lblockindex.LoadBlockIndexDB()

	err = InitGenesisChain()
	assert.Nil(t, err)

	mempool.InitMempool()

	crypto.InitSecp256()

	if initScriptVerify {
		ltx.ScriptVerifyInit()
	}

	return unitTestDataDirPath, nil
}

func TestStat(t *testing.T) {
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"}, false)
	assert.Nil(t, err)
	defer os.RemoveAll(testDir)

	done := make(chan struct{}, 1)
	cdb := utxo.GetUtxoCacheInstance().(*utxo.CoinsLruCache).GetCoinsDB()
	besthash, _ := cdb.GetBestBlock()

	var stat stat
	stat.bestblock = *besthash
	stat.height = int(chain.GetInstance().FindBlockIndex(*besthash).Height)
	iter := cdb.GetDBW().Iterator(nil)
	iter.Seek([]byte{db.DbCoin})
	taskControl.StartLogTask()
	taskControl.StartUtxoTask()
	taskControl.PushUtxoTask(utxoTaskArg{iter, &stat})
	done <- struct{}{}

	select {
	case <-done:

	case <-time.After(time.Second * 10):
		assert.Fail(t, "taskControl timeout")
	}

}
