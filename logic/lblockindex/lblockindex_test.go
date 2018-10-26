package lblockindex

import (
	"bytes"
	"fmt"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/util"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"
)

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
	conf.Cfg.DataDir = "/tmp"
	chain.InitGlobalChain()
	//gChain := chain.GetInstance()
	//gChain.SetTip(nil)
	initBlockDB()
	initUtxoDB()
	persist.InitPersistGlobal()
}

func initGenesis() {
	GlobalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)
	gChain := chain.GetInstance()
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

func initBlkFile(number int) {
	for i := 0; i <= number; i++ {
		pos := &block.DiskBlockPos{File: int32(i), Pos: 0}
		disk.OpenBlockFile(pos, true)
	}
}

func cleanEnv() {
	os.RemoveAll(pathblockdb)
	os.RemoveAll(pathutxodb)
}

func cleanBlkFile(number int) {
	for i := 0; i <= number; i++ {
		pos := &block.DiskBlockPos{File: int32(i), Pos: 0}
		os.Remove(disk.GetBlockPosFilename(*pos, "blk"))
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

func TestLoadBlockIndexDB_NoDB(t *testing.T) {
	initEnv()
	defer cleanEnv()

	ret := LoadBlockIndexDB()
	if !ret {
		t.Errorf("load fail")
	}

	gChain := chain.GetInstance()
	size := gChain.IndexMapSize()
	if size != 0 {
		t.Errorf("should have no index")
	}
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

	gChain := chain.GetInstance()
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

	blocktreedb := blkdb.GetInstance()
	err := blocktreedb.WriteBatchSync(fileInfoList, lastFile, blockIdx)
	if err != nil {
		t.Errorf("write blockindex fail")
	}

	ret := LoadBlockIndexDB()
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
	if int32(lastFile) != gPersist.GlobalLastBlockFile {
		t.Errorf("lastfile not equal with origin value")
	}

	if len(fileInfoList) != len(gPersist.GlobalBlockFileInfo) {
		t.Errorf("file number not equal with origin")
	}
}

func TestLoadBlockIndexDB_HasIndexNoForkMoreFile(t *testing.T) {
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

	gChain := chain.GetInstance()
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

	blocktreedb := blkdb.GetInstance()
	err := blocktreedb.WriteBatchSync(fileInfoList, lastFile, blockIdx)
	if err != nil {
		t.Errorf("write blockindex fail")
	}

	ret := LoadBlockIndexDB()
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

	gChain := chain.GetInstance()
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

	blocktreedb := blkdb.GetInstance()
	err := blocktreedb.WriteBatchSync(fileInfoList, lastFile, blockIdx)
	if err != nil {
		t.Errorf("write blockindex fail")
	}

	ret := LoadBlockIndexDB()
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
	if int32(lastFile) != gPersist.GlobalLastBlockFile {
		t.Errorf("lastfile+1 not equal with origin value")
	}

	if len(fileInfoList) != len(gPersist.GlobalBlockFileInfo) {
		t.Errorf("file number not equal with origin")
	}
}

func TestCheckIndexAgainstCheckpoint_OnlyGenesis(t *testing.T) {
	initEnv()
	defer cleanEnv()
	initGenesis()

	gChain := chain.GetInstance()
	genesisIndex := gChain.FindBlockIndex(*gChain.GetParams().GenesisHash)
	if genesisIndex == nil {
		t.Errorf("genesis index find fail")
	}

	ret := CheckIndexAgainstCheckpoint(genesisIndex)
	if !ret {
		t.Errorf("genesis index should ret true")
	}
}

func TestCheckIndexAgainstCheckpoint_HaveCheckpoint(t *testing.T) {
	initEnv()
	defer cleanEnv()
	initGenesis()

	gChain := chain.GetInstance()
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

	ret := CheckIndexAgainstCheckpoint(blockIdx[4])
	if ret {
		t.Errorf("idx[4] should ret false")
	}
	ret = CheckIndexAgainstCheckpoint(blockIdx[8])
	if !ret {
		t.Errorf("idx[8] should ret true")
	}
}
