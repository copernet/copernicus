package lchain

import (
	"encoding/json"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckBlockIndex_NoCheck(t *testing.T) {
	conf.Cfg = &conf.Configuration{}
	conf.Cfg.BlockIndex.CheckBlockIndex = false
	if err := CheckBlockIndex(); err != nil {
		t.Errorf("NoCheck should do nothing and return nil:%v", err)
	}
}

var timePerBlock = int64(model.ActiveNetParams.TargetTimePerBlock)
var initBits = model.ActiveNetParams.PowLimitBits

func getBlockIndex(indexPrev *blockindex.BlockIndex, timeInterval int64, bits uint32) *blockindex.BlockIndex {
	blockIdx := new(blockindex.BlockIndex)
	blockIdx.Prev = indexPrev
	blockIdx.Height = indexPrev.Height + 1
	blockIdx.Header.Time = indexPrev.Header.Time + uint32(timeInterval)
	blockIdx.Header.Bits = bits
	blockIdx.ChainWork = *big.NewInt(0).Add(&indexPrev.ChainWork, pow.GetBlockProof(blockIdx))

	seed := rand.NewSource(time.Now().Unix())
	random := rand.New(seed)
	blockIdx.TxCount = int32(random.Intn(1000) + 1)
	blockIdx.ChainTxCount = indexPrev.ChainTxCount + blockIdx.TxCount
	blockIdx.AddStatus(blockindex.BlockHaveData)
	blockIdx.RaiseValidity(blockindex.BlockValidTransactions)
	return blockIdx
}

//
//func makeTestBlockTreeDB() {
//	var args []string
//	conf.Cfg = conf.InitConfig(args)
//	if conf.Cfg == nil {
//		fmt.Println("please run `./copernicus -h` for usage.")
//		os.Exit(0)
//	}
//
//	if conf.Cfg.P2PNet.TestNet {
//		model.SetTestNetParams()
//	} else if conf.Cfg.P2PNet.RegTest {
//		model.SetRegTestParams()
//	}
//
//	pow.UpdateMinimumChainWork()
//	conf.DataDir = conf.DataDir + "/test"
//	fmt.Println("Current data dir:\033[0;32m", conf.DataDir, "\033[0m")
//
//	//init log
//	logDir := filepath.Join(conf.DataDir, log.DefaultLogDirname)
//	if !conf.FileExists(logDir) {
//		err := os.MkdirAll(logDir, os.ModePerm)
//		if err != nil {
//			panic("logdir create failed: " + err.Error())
//		}
//	}
//
//	logConf := struct {
//		FileName string `json:"filename"`
//		Level    int    `json:"level"`
//	}{
//		FileName: logDir + "/" + conf.Cfg.Log.FileName + ".log",
//		Level:    log.GetLevel(conf.Cfg.Log.Level),
//	}
//
//	configuration, err := json.Marshal(logConf)
//	if err != nil {
//		panic(err)
//	}
//	log.Init(string(configuration))
//
//	// Init UTXO DB
//	utxoDbCfg := &db.DBOption{
//		FilePath:  conf.DataDir + "/chainstate",
//		CacheSize: (1 << 20) * 8,
//		Wipe:      conf.Cfg.Reindex,
//	}
//	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
//	utxo.InitUtxoLruTip(&utxoConfig)
//
//	blkDbCfg := &db.DBOption{
//		FilePath:  conf.DataDir + "/blocks/index",
//		CacheSize: (1 << 20) * 8,
//		Wipe:      conf.Cfg.Reindex,
//	}
//	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
//	blkdb.InitBlockTreeDB(&blkdbCfg)
//
//}

func initTestEnv(t *testing.T, args []string) (dirpath string, err error) {
	conf.Cfg = conf.InitConfig(args)

	conf.Cfg.Chain.UtxoHashStartHeight = 0
	conf.Cfg.Chain.UtxoHashEndHeight = 1000

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
	}{
		FileName: logDir + "/" + conf.Cfg.Log.FileName + ".log",
		Level:    log.GetLevel(conf.Cfg.Log.Level),
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}
	log.Init(string(configuration))

	// Init UTXO DB
	utxoDbCfg := &db.DBOption{
		FilePath:  conf.DataDir + "/chainstate",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
	utxo.InitUtxoLruTip(&utxoConfig)

	// Init blocktree DB
	blkDbCfg := &db.DBOption{
		FilePath:  conf.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	mempool.InitMempool()

	crypto.InitSecp256()

	ltx.ScriptVerifyInit()

	chain.InitGlobalChain(blkdb.GetInstance())
	persist.InitPersistGlobal(blkdb.GetInstance())

	err = InitGenesisChain()
	assert.Nil(t, err)

	gChain := chain.GetInstance()
	gChain.SetTip(nil)

	GlobalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)
	gChain.InitLoad(GlobalBlockIndexMap, branch)

	bl := gChain.GetParams().GenesisBlock
	bIndex := blockindex.NewBlockIndex(&bl.Header)
	bIndex.Height = 0
	bIndex.TxCount = 1
	bIndex.ChainTxCount = 1
	bIndex.AddStatus(blockindex.BlockHaveData)
	bIndex.RaiseValidity(blockindex.BlockValidTransactions)
	err = gChain.AddToIndexMap(bIndex)
	if err != nil {
		panic("AddToIndexMap fail")
	}

	return unitTestDataDirPath, nil
}

//func initEnv() {
//
//	// set params, don't modify!
//	model.SetRegTestParams()
//	// clear chain data of last test case
//	testDir, err := initTestEnv(t, []string{"--regtest"})
//	assert.Nil(t, err)
//	defer os.RemoveAll(testDir)()
//
//	conf.Cfg = &conf.Configuration{}
//	conf.Cfg.BlockIndex.CheckBlockIndex = true
//
//	chain.InitGlobalChain(blkdb.GetInstance())
//	persist.InitPersistGlobal(blkdb.GetInstance())
//
//	gChain := chain.GetInstance()
//	gChain.SetTip(nil)
//
//	GlobalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
//	branch := make([]*blockindex.BlockIndex, 0, 20)
//	gChain.InitLoad(GlobalBlockIndexMap, branch)
//
//	bl := gChain.GetParams().GenesisBlock
//	bIndex := blockindex.NewBlockIndex(&bl.Header)
//	bIndex.Height = 0
//	bIndex.TxCount = 1
//	bIndex.ChainTxCount = 1
//	bIndex.AddStatus(blockindex.BlockHaveData)
//	bIndex.RaiseValidity(blockindex.BlockValidTransactions)
//	err := gChain.AddToIndexMap(bIndex)
//	if err != nil {
//		panic("AddToIndexMap fail")
//	}
//}

func TestCheckBlockIndex_OnlyGenesis(t *testing.T) {
	initTestEnv(t, []string{"--regtest"})
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)

	defer os.RemoveAll(testDir)

	gChain := chain.GetInstance()
	genesisIndex := gChain.FindBlockIndex(*gChain.GetParams().GenesisHash)
	if genesisIndex == nil {
		t.Errorf("genesis index find fail")
	}
	gChain.SetTip(genesisIndex)

	if err := CheckBlockIndex(); err != nil {
		t.Errorf("should return nil:%v", err)
	}
}

func TestCheckBlockIndex_NoFork(t *testing.T) {
	//initEnv()
	initTestEnv(t, []string{"--regtest"})
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)

	defer os.RemoveAll(testDir)
	gChain := chain.GetInstance()
	genesisIndex := gChain.FindBlockIndex(*gChain.GetParams().GenesisHash)
	if genesisIndex == nil {
		t.Errorf("genesis index find fail")
	}

	blockIdx := make([]*blockindex.BlockIndex, 10)
	blockIdx[0] = genesisIndex
	for i := 1; i < 10; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
		err := gChain.AddToIndexMap(blockIdx[i])
		if err != nil {
			t.Errorf("AddToIndexMap fail")
		}
	}

	gChain.SetTip(blockIdx[9])
	if err := CheckBlockIndex(); err != nil {
		t.Errorf("should return nil:%v", err)
	}
}

func TestCheckBlockIndex_TwoBranch(t *testing.T) {
	//initEnv()
	initTestEnv(t, []string{"--regtest"})
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)

	defer os.RemoveAll(testDir)
	gChain := chain.GetInstance()
	genesisIndex := gChain.FindBlockIndex(*gChain.GetParams().GenesisHash)
	if genesisIndex == nil {
		t.Errorf("genesis index find fail")
	}

	blockIdx := make([]*blockindex.BlockIndex, 20)
	blockIdx[0] = genesisIndex
	for i := 1; i < 10; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
		err := gChain.AddToIndexMap(blockIdx[i])
		if err != nil {
			t.Errorf("AddToIndexMap fail")
		}
	}

	blockIdx[10] = getBlockIndex(blockIdx[3], timePerBlock+1, initBits)
	err = gChain.AddToIndexMap(blockIdx[10])
	if err != nil {
		t.Errorf("AddToIndexMap fail")
	}

	for i := 11; i < 20; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock+1, initBits)
		err := gChain.AddToIndexMap(blockIdx[i])
		if err != nil {
			t.Errorf("AddToIndexMap fail")
		}
	}

	gChain.SetTip(blockIdx[19])
	if err := CheckBlockIndex(); err != nil {
		t.Errorf("should return nil:%v", err)
	}
}

func TestCheckBlockIndex_Invalid(t *testing.T) {
	//initEnv()
	initTestEnv(t, []string{"--regtest"})
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)

	defer os.RemoveAll(testDir)
	conf.Cfg = &conf.Configuration{}
	conf.Cfg.BlockIndex.CheckBlockIndex = true

	chain.InitGlobalChain(blkdb.GetInstance())
	gChain := chain.GetInstance()
	gChain.SetTip(nil)

	GlobalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)
	gChain.InitLoad(GlobalBlockIndexMap, branch)

	persist.InitPersistGlobal(blkdb.GetInstance())

	err = CheckBlockIndex()
	assert.NotNil(t, err)

	bl := gChain.GetParams().GenesisBlock
	bIndex := blockindex.NewBlockIndex(&bl.Header)
	bIndex.Height = 0
	bIndex.TxCount = 0
	bIndex.ChainTxCount = 0
	bIndex.AddStatus(blockindex.BlockValidUnknown)
	bIndex.RaiseValidity(blockindex.BlockFailed)
	err = gChain.AddToIndexMap(bIndex)
	if err != nil {
		panic("AddToIndexMap fail")
	}

	err = CheckBlockIndex()
	assert.NotNil(t, err)
}

func TestActivateBestChainStep_Invalid(t *testing.T) {
	//initEnv()
	initTestEnv(t, []string{"--regtest"})
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)

	defer os.RemoveAll(testDir)
	connTrace := make(connectTrace)
	invalid := false
	err = ActivateBestChainStep(blockindex.NewBlockIndex(block.NewBlockHeader()),
		block.NewBlock(), &invalid, connTrace)
	assert.NotNil(t, err)
}
