package persist

import (
	"encoding/json"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func createBlkIdx() *blockindex.BlockIndex {
	blkHeader := block.NewBlockHeader()
	blkHeader.Time = uint32(1534822771)
	blkHeader.Version = 536870912
	blkHeader.Bits = 486604799
	preHash := util.HashFromString("00000000000001bcd6b635a1249dfbe76c0d001592a7219a36cd9bbd002c7238")
	merkleRoot := util.HashFromString("7e814211a7de289a490380c0c20353e0fd4e62bf55a05b38e1628e0ea0b4fd3d")
	blkHeader.HashPrevBlock = *preHash
	blkHeader.Nonce = 1391785674
	blkHeader.MerkleRoot = *merkleRoot
	//init block index
	blkidx := blockindex.NewBlockIndex(blkHeader)
	return blkidx
}

func TestGetInstance(t *testing.T) {
	testDir, err := initTestEnv(t, []string{"--regtest"})
	if err != nil {
		t.Error("initTestEnv err")
	}
	defer os.RemoveAll(testDir)

	GetInstance()
	prstGloal := new(PersistGlobal)
	prstGloal.GlobalBlockFileInfo = make([]*block.BlockFileInfo, 0, 1000)
	prstGloal.GlobalDirtyFileInfo = make(map[int32]bool)
	prstGloal.GlobalDirtyBlockIndex = make(map[util.Hash]*blockindex.BlockIndex)
	prstGloal.GlobalMapBlocksUnlinked = make(map[*blockindex.BlockIndex][]*blockindex.BlockIndex)
	if !reflect.DeepEqual(prstGloal, persistGlobal) {
		t.Error("the global variable should eaual.")
	}

	for i := 0; i < 10; i++ {
		prstGloal.AddBlockSequenceID()
		if prstGloal.GlobalBlockSequenceID != int32(i+1) {
			t.Errorf("the GlobalBlockSequenceID:%d should equal i", prstGloal.GlobalBlockSequenceID)
		}
	}

	blkidx := createBlkIdx()
	prstGloal.AddDirtyBlockIndex(blkidx)
	mapDirtyBlkIdx := prstGloal.GlobalDirtyBlockIndex[*blkidx.GetBlockHash()]
	if !reflect.DeepEqual(blkidx, mapDirtyBlkIdx) {
		t.Errorf("the GlobalDirtyBlockIndex value should equal.")
	}
}

func TestInitPruneState(t *testing.T) {
	initps := InitPruneState()
	ps := &PruneState{
		PruneMode:       false,
		HavePruned:      false,
		CheckForPruning: false,
		PruneTarget:     0,
	}
	if !reflect.DeepEqual(initps, ps) {
		t.Errorf("the prune state should equal")
	}
}

func initTestEnv(t *testing.T, args []string) (dirpath string, err error) {
	conf.Cfg = conf.InitConfig(args)

	conf.Cfg.Chain.UtxoHashStartHeight = 0
	conf.Cfg.Chain.UtxoHashEndHeight = 1000

	unitTestDataDirPath, err := conf.SetUnitTestDataDir(conf.Cfg)

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

	InitPersistGlobal(blkdb.GetInstance())

	return unitTestDataDirPath, nil
}
