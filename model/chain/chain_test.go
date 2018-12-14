package chain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/persist/disk"
	"gopkg.in/fatih/set.v0"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"math/big"
	"testing"
)

func makeTestBlockTreeDB() {
	var args []string
	conf.Cfg = conf.InitConfig(args)
	if conf.Cfg == nil {
		fmt.Println("please run `./copernicus -h` for usage.")
		os.Exit(0)
	}

	if conf.Cfg.P2PNet.TestNet {
		model.SetTestNetParams()
	} else if conf.Cfg.P2PNet.RegTest {
		model.SetRegTestParams()
	}

	pow.UpdateMinimumChainWork()
	conf.DataDir = conf.DataDir + "/test"
	fmt.Println("Current data dir:\033[0;32m", conf.DataDir, "\033[0m")

	//init log
	logDir := filepath.Join(conf.DataDir, log.DefaultLogDirname)
	if !conf.FileExists(logDir) {
		err := os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			panic("logdir create failed: " + err.Error())
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

	blkDbCfg := &db.DBOption{
		FilePath:  conf.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

}

func TestMain(m *testing.M) {
	makeTestBlockTreeDB()
	persist.InitPersistGlobal(blkdb.GetInstance())
	conf.Cfg = conf.InitConfig([]string{})
	os.Exit(m.Run())
}

func getBlockIndexSimple(indexPrev *blockindex.BlockIndex, timeInterval int64, bits uint32) *blockindex.BlockIndex {
	blockIdx := new(blockindex.BlockIndex)
	blockIdx.Prev = indexPrev
	blockIdx.BuildSkip()
	blockIdx.Height = indexPrev.Height + 1
	blockIdx.Header.Time = indexPrev.Header.Time + uint32(timeInterval)
	blockIdx.Header.Bits = bits
	blockIdx.Header.HashPrevBlock = indexPrev.Header.GetHash()
	blockIdx.ChainWork = *big.NewInt(0).Add(&indexPrev.ChainWork, pow.GetBlockProof(blockIdx))
	return blockIdx
}

func TestChain_Simple(t *testing.T) {
	InitGlobalChain(blkdb.GetInstance())
	tChain := GetInstance()

	tChain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	bIndex := make([]*blockindex.BlockIndex, 50)
	tChain.active = make([]*blockindex.BlockIndex, 0)
	tChain.branch = make([]*blockindex.BlockIndex, 0)
	initBits := model.ActiveNetParams.PowLimitBits
	timePerBlock := int64(model.ActiveNetParams.TargetTimePerBlock)
	height := 0

	//Pile up some blocks
	bIndex[0] = blockindex.NewBlockIndex(&model.ActiveNetParams.GenesisBlock.Header)
	tChain.AddToBranch(bIndex[0])
	tChain.AddToIndexMap(bIndex[0])
	tChain.SetTip(bIndex[0])

	for height = 1; height < 11; height++ {
		bIndex[height] = getBlockIndexSimple(bIndex[height-1], timePerBlock, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
		tChain.SetTip(bIndex[height])
	}
	for height = 11; height < 16; height++ {
		bIndex[height] = getBlockIndexSimple(bIndex[height-1], timePerBlock, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
	}

	if tChain.GetParams() != model.ActiveNetParams {
		t.Errorf("GetParams expect: %s, actual: %s", model.ActiveNetParams.Name, tChain.GetParams().Name)
	}
	if tChain.Genesis() != bIndex[0] {
		t.Errorf("Genesis expect: %s, actual: %s", bIndex[0].GetBlockHash(), tChain.Genesis().GetBlockHash())
	}
	if tChain.FindHashInActive(model.GenesisBlockHash) != bIndex[0] {
		t.Errorf("FindHashInActive Error")
	}
	if tChain.FindBlockIndex(model.GenesisBlockHash) != bIndex[0] {
		t.Errorf("FindHashInActive Error")
	}
	if tChain.Tip() != bIndex[10] {
		t.Errorf("Tip Error")
	}
	if tChain.TipHeight() != 10 {
		t.Errorf("TipHeight Error")
	}

	if tChain.GetSpendHeight(bIndex[15].GetBlockHash()) != 16 {
		t.Errorf("GetSpendHeight Error")
	}
	if tChain.GetIndex(10) != bIndex[10] {
		t.Errorf("GetIndex Error")
	}
	if !tChain.Equal(tChain) {
		t.Errorf("Equal Error")
	}
	if tChain.Contains(bIndex[15]) {
		t.Errorf("Contains Error")
	}
	if tChain.Next(bIndex[9]) != bIndex[10] {
		t.Errorf("Next Error")
	}
	if tChain.Height() != 10 {
		t.Errorf("Height Error")
	}
	if tChain.GetAncestor(10) != bIndex[10] {
		t.Errorf("GetAncestor Error")
	}
	if tChain.SetTip(bIndex[6]); tChain.Tip() != bIndex[6] {
		t.Errorf("SetTip Error")
	}
	if !tChain.ParentInBranch(bIndex[10]) {
		t.Errorf("ParentInBranch Error")
	}
	if tChain.RemoveFromBranch(bIndex[15]); tChain.InBranch(bIndex[15]) {
		t.Errorf("InBranch Error")
	}
	if tChain.FindMostWorkChain() != bIndex[14] {
		t.Errorf("FindMostWorkChain Error")
	}
	if tChain.AddToOrphan(bIndex[15]); tChain.ChainOrphanLen() != 1 {
		t.Errorf("AddToOrphan Error")
	}
	if tChain.IndexMapSize() != 16 {
		t.Errorf("IndexMapSize Error")
	}
	if tChain.ClearActive(); tChain.Tip() != nil {
		t.Errorf("ClearActive Error")
	}
}

func TestChain_Fork(t *testing.T) {
	//makeTestBlockTreeDB()
	InitGlobalChain(blkdb.GetInstance())
	tChain := GetInstance()

	tChain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	tChain.active = make([]*blockindex.BlockIndex, 0)
	tChain.branch = make([]*blockindex.BlockIndex, 0)
	bIndex := make([]*blockindex.BlockIndex, 50)
	initBits := model.ActiveNetParams.PowLimitBits
	timePerBlock := int64(model.ActiveNetParams.TargetTimePerBlock)
	height := 0

	//Pile up some blocks
	bIndex[0] = blockindex.NewBlockIndex(&model.ActiveNetParams.GenesisBlock.Header)
	tChain.AddToIndexMap(bIndex[0])
	tChain.AddToBranch(bIndex[0])
	tChain.SetTip(bIndex[0])

	for height = 1; height < 11; height++ {
		bIndex[height] = getBlockIndexSimple(bIndex[height-1], timePerBlock, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
		tChain.SetTip(bIndex[height])
	}
	for height = 5; height < 15; height++ {
		bIndex[height] = getBlockIndexSimple(bIndex[height-1], timePerBlock-1, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
	}

	if tChain.FindFork(bIndex[9]) != bIndex[4] {
		t.Errorf("FindFork Error")
	}

	setTips := set.New()
	setTips.Add(tChain.Tip())
	setTips.Add(bIndex[14])

	if !tChain.GetChainTips().IsEqual(setTips) {
		t.Errorf("GetChainTips Error")
	}

}

func TestChain_InitLoad(t *testing.T) {
	//makeTestBlockTreeDB()
	InitGlobalChain(blkdb.GetInstance())
	tChain := GetInstance()
	tChain.ClearActive()

	tChain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	bIndex := make([]*blockindex.BlockIndex, 50)
	tChain.active = make([]*blockindex.BlockIndex, 0)
	tChain.branch = make([]*blockindex.BlockIndex, 0)
	initBits := model.ActiveNetParams.PowLimitBits
	timePerBlock := int64(model.ActiveNetParams.TargetTimePerBlock)
	height := 0

	tChain.InitLoad(tChain.indexMap, tChain.branch)
	if tChain.Genesis() != nil {
		t.Errorf("Genesis Error")
	}
	if tChain.FindHashInActive(model.GenesisBlockHash) != nil {
		t.Errorf("FindHashInActive Error")
	}
	if tChain.FindBlockIndex(model.GenesisBlockHash) != nil {
		t.Errorf("FindBlockIndex Error")
	}
	if tChain.Tip() != nil {
		t.Errorf("Tip Error")
	}
	if tChain.TipHeight() != 0 {
		t.Errorf("TipHeight Error")
	}
	if tChain.GetSpendHeight(&model.GenesisBlockHash) != -1 {
		t.Errorf("GetSpendHeight Error")
	}
	if tChain.Contains(nil) {
		t.Errorf("Contains Error")
	}
	if tChain.Next(nil) != nil {
		t.Errorf("Next Error")
	}
	if tChain.FindFork(nil) != nil {
		t.Errorf("FindFork Error")
	}
	if tChain.ParentInBranch(nil) {
		t.Errorf("ParentInBranch Error")
	}
	if tChain.InBranch(nil) {
		t.Errorf("InBranch Error")
	}
	if tChain.AddToBranch(nil) == nil {
		t.Errorf("AddToBranch Error")
	}
	if tChain.RemoveFromBranch(nil) == nil {
		t.Errorf("RemoveFromBranch Error")
	}
	if tChain.FindMostWorkChain() != nil {
		t.Errorf("FindMostWorkChain Error")
	}
	if tChain.AddToIndexMap(nil) == nil {
		t.Errorf("AddToIndexMap Error")
	}

	//Pile up some blocks
	bIndex[0] = blockindex.NewBlockIndex(&model.ActiveNetParams.GenesisBlock.Header)
	tChain.AddToBranch(bIndex[0])
	tChain.AddToIndexMap(bIndex[0])
	tChain.SetTip(bIndex[0])

	for height = 1; height < 11; height++ {
		bIndex[height] = getBlockIndexSimple(bIndex[height-1], timePerBlock, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
		tChain.SetTip(bIndex[height])
	}
	if tChain.SetTip(nil); tChain.Tip() != nil {
		t.Errorf("SetTip Error")
	}
}

func TestChain_GetBlockScriptFlags(t *testing.T) {
	//makeTestBlockTreeDB()
	InitGlobalChain(blkdb.GetInstance())
	testChain := GetInstance()
	timePerBlock := int64(model.ActiveNetParams.TargetTimePerBlock)
	initBits := model.ActiveNetParams.PowLimitBits

	blockIdx := make([]*blockindex.BlockIndex, 100)
	blockheader := block.NewBlockHeader()
	blockheader.Time = 1332234914
	blockIdx[0] = blockindex.NewBlockIndex(blockheader)
	blockIdx[0].Height = 172011
	for i := 1; i < 20; i++ {
		blockIdx[i] = getBlockIndexSimple(blockIdx[i-1], timePerBlock, initBits)
	}
	expect := script.ScriptVerifyNone
	if flag := testChain.GetBlockScriptFlags(blockIdx[19]); flag != uint32(expect) {
		t.Errorf("GetBlockScriptFlags wrong: %d", flag)
	}

	blockIdx = make([]*blockindex.BlockIndex, 100)
	blockheader = block.NewBlockHeader()
	blockheader.Time = 1335916577
	blockIdx[0] = blockindex.NewBlockIndex(blockheader)
	blockIdx[0].Height = 178184
	for i := 1; i < 20; i++ {
		blockIdx[i] = getBlockIndexSimple(blockIdx[i-1], timePerBlock, initBits)
	}
	expect |= script.ScriptVerifyP2SH
	if flag := testChain.GetBlockScriptFlags(blockIdx[19]); flag != uint32(expect) {
		t.Errorf("GetBlockScriptFlags wrong: %d", flag)
	}

	blockIdx = make([]*blockindex.BlockIndex, 100)
	blockheader = block.NewBlockHeader()
	blockheader.Time = 1435974872
	blockIdx[0] = blockindex.NewBlockIndex(blockheader)
	blockIdx[0].Height = model.ActiveNetParams.BIP66Height
	for i := 1; i < 20; i++ {
		blockIdx[i] = getBlockIndexSimple(blockIdx[i-1], timePerBlock, initBits)
	}
	expect |= script.ScriptVerifyDersig
	if flag := testChain.GetBlockScriptFlags(blockIdx[19]); flag != uint32(expect) {
		t.Errorf("GetBlockScriptFlags wrong: %d", flag)
	}

	blockIdx = make([]*blockindex.BlockIndex, 100)
	blockheader = block.NewBlockHeader()
	blockheader.Time = 1450113884
	blockIdx[0] = blockindex.NewBlockIndex(blockheader)
	blockIdx[0].Height = model.ActiveNetParams.BIP65Height
	for i := 1; i < 20; i++ {
		blockIdx[i] = getBlockIndexSimple(blockIdx[i-1], timePerBlock, initBits)
	}
	expect |= script.ScriptVerifyCheckLockTimeVerify
	if flag := testChain.GetBlockScriptFlags(blockIdx[19]); flag != uint32(expect) {
		t.Errorf("GetBlockScriptFlags wrong: %d", flag)
	}
}

func TestBuildForwardTree(t *testing.T) {
	globalChain = nil
	//makeTestBlockTreeDB()
	InitGlobalChain(blkdb.GetInstance())
	testChain := GetInstance()
	testChain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	blockIdx := make([]*blockindex.BlockIndex, 50)
	initBits := model.ActiveNetParams.PowLimitBits
	timePerBlock := int64(model.ActiveNetParams.TargetTimePerBlock)
	height := 0

	//Pile up some blocks
	blockIdx[height] = blockindex.NewBlockIndex(&model.ActiveNetParams.GenesisBlock.Header)
	testChain.AddToIndexMap(blockIdx[height])
	for height = 1; height < 11; height++ {
		i := height
		dummyPow := big.NewInt(0).Rsh(model.ActiveNetParams.PowLimit, uint(i))
		blockIdx[height] = getBlockIndexSimple(blockIdx[height-1], timePerBlock, pow.BigToCompact(dummyPow))
		testChain.AddToIndexMap(blockIdx[height])
	}
	for height = 11; height < 21; height++ {
		blockIdx[height] = getBlockIndexSimple(blockIdx[height-11], timePerBlock, initBits)
		testChain.AddToIndexMap(blockIdx[height])
	}

	forward := testChain.BuildForwardTree()
	forwardCount := 0
	for _, v := range forward {
		forwardCount += len(v)
	}
	indexCount := testChain.IndexMapSize()
	if forwardCount != indexCount {
		t.Errorf("forward tree node count wrong, expect:%d, actual:%d", indexCount, forwardCount)
	}

	genesisSlice, ok := forward[nil]
	if ok {
		if len(genesisSlice) != 1 {
			t.Errorf("genesis block number wrong, expect only 1, actual:%d, info:%v", len(genesisSlice), genesisSlice)
		}
		if genesisSlice[0] != blockIdx[0] {
			t.Errorf("genesis block wrong, expect:%v, actual:%v", blockIdx[0], genesisSlice[0])
		}
	} else {
		t.Errorf("no any genesis block, expect 1")
	}

	height1Slice, ok := forward[genesisSlice[0]]
	if ok {
		if len(height1Slice) != 2 {
			t.Errorf("height1 block number wrong, expect 2, actual:%d, info:%v", len(height1Slice), height1Slice)
		}
		if (height1Slice[0] != blockIdx[1] && height1Slice[0] != blockIdx[11]) || (height1Slice[1] != blockIdx[1] && height1Slice[1] != blockIdx[11]) {
			t.Errorf("height1 block wrong, expect1:%v, expect2:%v, actual:%v", blockIdx[1], blockIdx[11], height1Slice)
		}
	} else {
		t.Errorf("no any height1 block, expect 2")
	}

	height11Slice, ok := forward[blockIdx[10]]
	if ok {
		t.Errorf("height 10 should not have any son, but now have:%v", height11Slice)
	}
}

func initBlkFile(number int) {
	for i := 0; i <= number; i++ {
		pos := &block.DiskBlockPos{File: int32(i), Pos: 0}
		disk.OpenBlockFile(pos, true)
	}
}

func cleanBlkFile(number int) {
	for i := 0; i <= number; i++ {
		pos := &block.DiskBlockPos{File: int32(i), Pos: 0}
		os.Remove(disk.GetBlockPosFilename(*pos, "blk"))
	}
}

var pathblockdb string
var pathutxodb string

func initBlockDB() {
	var err error
	pathblockdb, err = ioutil.TempDir("/tmp", "blockindextest")
	if err != nil {
		panic(fmt.Sprintf("generate temp db path failed: %s\n", err))
	}
	bc := &blkdb.BlockTreeDBConfig{
		Do: &db.DBOption{
			FilePath:  pathblockdb,
			CacheSize: 1 << 20,
		},
	}

	blkdb.InitBlockTreeDB(bc)
}

func initUtxoDB() {
	var err error
	pathutxodb, err = ioutil.TempDir("/tmp", "utxotest")
	if err != nil {
		panic(fmt.Sprintf("generate temp db path failed: %s\n", err))
	}

	dbo := db.DBOption{
		FilePath:       pathutxodb,
		CacheSize:      1 << 20,
		Wipe:           false,
		DontObfuscate:  false,
		ForceCompactdb: false,
	}

	uc := &utxo.UtxoConfig{
		Do: &dbo,
	}
	utxo.InitUtxoLruTip(uc)
}

func initEnv() {
	model.SetRegTestParams()
	conf.Cfg = &conf.Configuration{}
	conf.DataDir = "/tmp"
	initBlockDB()
	InitGlobalChain(blkdb.GetInstance())
	//gChain := chain.GetInstance()
	//gChain.SetTip(nil)
	initUtxoDB()
	persist.InitPersistGlobal(blkdb.GetInstance())
}

func cleanEnv() {
	os.RemoveAll(pathblockdb)
	os.RemoveAll(pathutxodb)
}

func initGenesis() {
	GlobalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)
	gChain := GetInstance()
	gChain.InitLoad(GlobalBlockIndexMap, branch)

	bl := gChain.GetParams().GenesisBlock
	bIndex := blockindex.NewBlockIndex(&bl.Header)
	bIndex.Height = 0
	bIndex.TxCount = 1
	bIndex.ChainTxCount = 1
	bIndex.File = 0
	bIndex.AddStatus(blockindex.BlockHaveData)
	bIndex.RaiseValidity(blockindex.BlockValidTransactions)
	err := gChain.AddToIndexMap(bIndex)
	if err != nil {
		panic("AddToIndexMap fail")
	}
}

func TestLoadBlockIndexDB_EmptyDB(t *testing.T) {
	initEnv()
	defer cleanEnv()

	c := GetInstance()
	ret := c.loadBlockIndex(blkdb.GetInstance())
	if !ret {
		t.Errorf("load fail")
	}

	gChain := GetInstance()
	size := gChain.IndexMapSize()
	if size != 0 {
		t.Errorf("should have no index")
	}
}

var timePerBlock = int64(model.ActiveNetParams.TargetTimePerBlock)
var initBits = uint32(0x207FFFFF)

func getBlockIndex(indexPrev *blockindex.BlockIndex, timeInterval int64, bits uint32) *blockindex.BlockIndex {
	blockIdx := new(blockindex.BlockIndex)
	blockIdx.Prev = indexPrev
	blockIdx.Header.HashPrevBlock = indexPrev.Header.GetHash()
	blockIdx.Height = indexPrev.Height + 1
	blockIdx.File = blockIdx.Height / 3
	blockIdx.Header.Time = indexPrev.Header.Time + uint32(timeInterval)
	if blockIdx.Height%3 == 0 {
		blockIdx.Header.Time = indexPrev.Header.Time - uint32(timeInterval)
	}
	blockIdx.Header.Bits = bits
	blockIdx.ChainWork = *big.NewInt(0).Add(&indexPrev.ChainWork, pow.GetBlockProof(blockIdx))

	powCheck := pow.Pow{}
	params := model.ActiveNetParams
	for {
		blockIdx.Header.Nonce++
		hash := blockIdx.Header.GetHash()
		log.Debug("mining height %d, hash: %s", blockIdx.Height, hash)
		if powCheck.CheckProofOfWork(&hash, bits, params) {
			break
		}
		blockIdx.Header.Hash = util.HashZero
	}

	seed := rand.NewSource(time.Now().Unix())
	random := rand.New(seed)
	blockIdx.TxCount = int32(random.Intn(1000) + 1)
	blockIdx.ChainTxCount = indexPrev.ChainTxCount + blockIdx.TxCount
	if blockIdx.Height%4 == 0 {
		blockIdx.ChainTxCount = 0
	}
	blockIdx.Header.Bits = bits
	blockIdx.AddStatus(blockindex.BlockHaveData)
	blockIdx.RaiseValidity(blockindex.BlockValidTransactions)
	return blockIdx
}

func TestLoadBlockIndexDB_HasIndexNoFork(t *testing.T) {
	initEnv()
	defer cleanEnv()
	initGenesis()

	blocknumber := 10
	initBlkFile((blocknumber - 1) / 3)
	defer cleanBlkFile((blocknumber - 1) / 3)

	fileInfoList := map[int32]*block.BlockFileInfo{}
	for i := 0; i <= (blocknumber-1)/3; i++ {
		fileInfoList[int32(i)] = &block.BlockFileInfo{}
	}
	var lastFile = (blocknumber - 1) / 3

	gChain := GetInstance()
	genesisIndex := gChain.FindBlockIndex(*gChain.GetParams().GenesisHash)
	if genesisIndex == nil {
		t.Errorf("genesis index find fail")
	}

	blockIdx := make([]*blockindex.BlockIndex, blocknumber)
	blockIdx[0] = genesisIndex
	for i := 1; i < blocknumber; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
		err := gChain.AddToIndexMap(blockIdx[i])
		if err != nil {
			t.Errorf("AddToIndexMap fail")
		}
	}

	btd := blkdb.GetInstance()
	err := btd.WriteBatchSync(fileInfoList, lastFile, blockIdx)
	if err != nil {
		t.Errorf("write blockindex fail")
	}
	ret := gChain.loadBlockIndex(blkdb.GetInstance())
	if !ret {
		t.Errorf("load fail")
	}
	size := gChain.IndexMapSize()
	if size != blocknumber {
		t.Errorf("block number not equal")
	}

	for i := 0; i < blocknumber; i++ {
		chainIndex := gChain.FindBlockIndex(blockIdx[i].Header.GetHash())
		if chainIndex == nil {
			t.Errorf("index in gChain find fail")
		}
		origin := make([]byte, 0, 100)
		current := make([]byte, 0, 100)
		oriBuf := bytes.NewBuffer(origin)
		curBuf := bytes.NewBuffer(current)
		if err := blockIdx[i].Serialize(oriBuf); err != nil {
			t.Error("serialize fail")
		}
		if err := chainIndex.Serialize(curBuf); err != nil {
			t.Error("serialize fail")
		}
		if !reflect.DeepEqual(oriBuf.Bytes(), curBuf.Bytes()) {
			t.Errorf("index in gChain do not deep equal after reload")
		}
	}

	gPersist := persist.GetInstance()
	gPersist.LoadBlockFileInfo(btd)
	if int32(lastFile) != gPersist.GlobalLastBlockFile {
		t.Errorf("lastfile not equal with origin value")
	}

	if len(fileInfoList) != len(gPersist.GlobalBlockFileInfo) {
		t.Errorf("file number not equal with origin")
	}
}

func TestLoadBlockIndexDB_HasIndexNoForkMoreFile(t *testing.T) {
	//makeTestBlockTreeDB()
	initEnv()
	defer cleanEnv()
	initGenesis()

	blocknumber := 10
	initBlkFile((blocknumber - 1) / 3)
	defer cleanBlkFile((blocknumber - 1) / 3)

	fileInfoList := map[int32]*block.BlockFileInfo{}
	for i := 0; i <= (blocknumber-1)/3+1; i++ {
		fileInfoList[int32(i)] = &block.BlockFileInfo{}
	}
	var lastFile = (blocknumber - 1) / 3

	gChain := GetInstance()
	genesisIndex := gChain.FindBlockIndex(*gChain.GetParams().GenesisHash)
	if genesisIndex == nil {
		t.Errorf("genesis index find fail")
	}

	blockIdx := make([]*blockindex.BlockIndex, blocknumber)
	blockIdx[0] = genesisIndex
	for i := 1; i < blocknumber; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
		err := gChain.AddToIndexMap(blockIdx[i])
		if err != nil {
			t.Errorf("AddToIndexMap fail")
		}
	}

	btd := blkdb.GetInstance()
	err := btd.WriteBatchSync(fileInfoList, lastFile, blockIdx)
	if err != nil {
		t.Errorf("write blockindex fail")
	}

	ret := gChain.loadBlockIndex(btd)
	if !ret {
		t.Errorf("load fail")
	}

	size := gChain.IndexMapSize()
	if size != blocknumber {
		t.Errorf("block number not equal")
	}

	for i := 0; i < blocknumber; i++ {
		chainIndex := gChain.FindBlockIndex(blockIdx[i].Header.GetHash())
		if chainIndex == nil {
			t.Errorf("index in gChain find fail")
		}
		origin := make([]byte, 0, 100)
		current := make([]byte, 0, 100)
		oriBuf := bytes.NewBuffer(origin)
		curBuf := bytes.NewBuffer(current)
		if err := blockIdx[i].Serialize(oriBuf); err != nil {
			t.Error("serialize fail")
		}
		if err := chainIndex.Serialize(curBuf); err != nil {
			t.Error("serialize fail")
		}
		if !reflect.DeepEqual(oriBuf.Bytes(), curBuf.Bytes()) {
			t.Errorf("index in gChain do not deep equal after reload")
		}
	}

	gPersist := persist.GetInstance()
	gPersist.LoadBlockFileInfo(btd)
	if int32(lastFile)+1 != gPersist.GlobalLastBlockFile {
		t.Errorf("lastfile+1 not equal with origin value")
	}

	if len(fileInfoList) != len(gPersist.GlobalBlockFileInfo) {
		t.Errorf("file number not equal with origin")
	}
}

func TestLoadBlockIndexDB_HasFork(t *testing.T) {
	initEnv()
	defer cleanEnv()
	initGenesis()

	blocknumber := 10
	initBlkFile((blocknumber - 1) / 3)
	defer cleanBlkFile((blocknumber - 1) / 3)

	fileInfoList := map[int32]*block.BlockFileInfo{}
	for i := 0; i <= (blocknumber-1)/3; i++ {
		fileInfoList[int32(i)] = &block.BlockFileInfo{}
	}
	var lastFile = (blocknumber - 1) / 3

	gChain := GetInstance()
	genesisIndex := gChain.FindBlockIndex(*gChain.GetParams().GenesisHash)
	if genesisIndex == nil {
		t.Errorf("genesis index find fail")
	}

	blockIdx := make([]*blockindex.BlockIndex, blocknumber*2-1)
	blockIdx[0] = genesisIndex
	for i := 1; i < blocknumber; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
		err := gChain.AddToIndexMap(blockIdx[i])
		if err != nil {
			t.Errorf("AddToIndexMap fail")
		}
	}
	for i := blocknumber; i < blocknumber*2-1; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-blocknumber], timePerBlock+1, initBits)
		err := gChain.AddToIndexMap(blockIdx[i])
		if err != nil {
			t.Errorf("AddToIndexMap fail")
		}
	}

	btd := blkdb.GetInstance()
	err := btd.WriteBatchSync(fileInfoList, lastFile, blockIdx)
	if err != nil {
		t.Errorf("write blockindex fail")
	}

	ret := gChain.loadBlockIndex(btd)
	if !ret {
		t.Errorf("load fail")
	}

	size := gChain.IndexMapSize()
	if size != blocknumber*2-1 {
		t.Errorf("block number not equal")
	}

	for i := 0; i < blocknumber*2-1; i++ {
		chainIndex := gChain.FindBlockIndex(blockIdx[i].Header.GetHash())
		if chainIndex == nil {
			t.Errorf("index in gChain find fail")
		}
		origin := make([]byte, 0, 100)
		current := make([]byte, 0, 100)
		oriBuf := bytes.NewBuffer(origin)
		curBuf := bytes.NewBuffer(current)
		if err := blockIdx[i].Serialize(oriBuf); err != nil {
			t.Error("serialize fail")
		}
		if err := chainIndex.Serialize(curBuf); err != nil {
			t.Error("serialize fail")
		}
		if !reflect.DeepEqual(oriBuf.Bytes(), curBuf.Bytes()) {
			t.Errorf("index in gChain do not deep equal after reload")
		}
	}

	gPersist := persist.GetInstance()
	gPersist.LoadBlockFileInfo(btd)
	if int32(lastFile) != gPersist.GlobalLastBlockFile {
		t.Errorf("lastfile+1 not equal with origin value")
	}

	if len(fileInfoList) != len(gPersist.GlobalBlockFileInfo) {
		t.Errorf("file number not equal with origin")
	}
}

func TestCheckIndexAgainstCheckpoint_HaveCheckpoint(t *testing.T) {
	initEnv()
	defer cleanEnv()
	initGenesis()

	gChain := GetInstance()
	genesisIndex := gChain.FindBlockIndex(*gChain.GetParams().GenesisHash)
	if genesisIndex == nil {
		t.Errorf("genesis index find fail")
	}

	blocknumber := 10

	blockIdx := make([]*blockindex.BlockIndex, blocknumber*2-1)
	blockIdx[0] = genesisIndex
	for i := 1; i < blocknumber; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
		err := gChain.AddToIndexMap(blockIdx[i])
		if err != nil {
			t.Errorf("AddToIndexMap fail")
		}
	}

	params := gChain.GetParams()
	params.Checkpoints = append(params.Checkpoints, &model.Checkpoint{Height: 3, Hash: blockIdx[3].GetBlockHash()})
	params.Checkpoints = append(params.Checkpoints, &model.Checkpoint{Height: 6, Hash: blockIdx[6].GetBlockHash()})

	err := gChain.CheckIndexAgainstCheckpoint(blockIdx[4])
	if err == nil {
		t.Errorf("idx[4] should ret error")
	}

	err = gChain.CheckIndexAgainstCheckpoint(blockIdx[8])
	if err != nil {
		t.Errorf("idx[8] should pass")
	}
}
