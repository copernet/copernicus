package chain

import (
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/persist"
	"gopkg.in/fatih/set.v0"
	"os"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"math/big"
	"testing"
)

func TestMain(m *testing.M) {
	persist.InitPersistGlobal()
	conf.Cfg = conf.InitConfig([]string{})
	os.Exit(m.Run())
}

func getBlockIndex(indexPrev *blockindex.BlockIndex, timeInterval int64, bits uint32) *blockindex.BlockIndex {
	blockIdx := new(blockindex.BlockIndex)
	blockIdx.Prev = indexPrev
	blockIdx.Height = indexPrev.Height + 1
	blockIdx.Header.Time = indexPrev.Header.Time + uint32(timeInterval)
	blockIdx.Header.Bits = bits
	blockIdx.Header.HashPrevBlock = indexPrev.Header.GetHash()
	blockIdx.ChainWork = *big.NewInt(0).Add(&indexPrev.ChainWork, pow.GetBlockProof(blockIdx))
	return blockIdx
}

func TestChain_Simple(t *testing.T) {
	InitGlobalChain()
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
	tChain.active = append(tChain.active, bIndex[0])

	for height = 1; height < 11; height++ {
		bIndex[height] = getBlockIndex(bIndex[height-1], timePerBlock, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
		tChain.active = append(tChain.active, bIndex[height])
	}
	for height = 11; height < 16; height++ {
		bIndex[height] = getBlockIndex(bIndex[height-1], timePerBlock, initBits)
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
	if tChain.IsCurrent() {
		t.Errorf("IsCurrent Error")
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
	InitGlobalChain()
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
	tChain.active = append(tChain.active, bIndex[0])

	for height = 1; height < 11; height++ {
		bIndex[height] = getBlockIndex(bIndex[height-1], timePerBlock, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
		tChain.active = append(tChain.active, bIndex[height])
	}
	for height = 5; height < 15; height++ {
		bIndex[height] = getBlockIndex(bIndex[height-1], timePerBlock-1, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
		tChain.active = append(tChain.active, bIndex[height])
	}

	if tChain.FindFork(bIndex[9]) != bIndex[4] {
		t.Errorf("FindFork Error")
	}

	setTips := set.New()
	setTips.Add(tChain.Tip())

	if !tChain.GetChainTips().IsEqual(setTips) {
		t.Errorf("GetChainTips Error")
	}

}

func TestChain_InitLoad(t *testing.T) {
	InitGlobalChain()
	tChain := GetInstance()

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
	tChain.active = append(tChain.active, bIndex[0])

	for height = 1; height < 11; height++ {
		bIndex[height] = getBlockIndex(bIndex[height-1], timePerBlock, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
		tChain.active = append(tChain.active, bIndex[height])
	}
	if tChain.SetTip(nil); tChain.Tip() != nil {
		t.Errorf("SetTip Error")
	}
}

func TestChain_GetBlockScriptFlags(t *testing.T) {
	InitGlobalChain()
	testChain := GetInstance()
	timePerBlock := int64(model.ActiveNetParams.TargetTimePerBlock)
	initBits := model.ActiveNetParams.PowLimitBits

	blockIdx := make([]*blockindex.BlockIndex, 100)
	blockheader := block.NewBlockHeader()
	blockheader.Time = 1332234914
	blockIdx[0] = blockindex.NewBlockIndex(blockheader)
	blockIdx[0].Height = 172011
	for i := 1; i < 20; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
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
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
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
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
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
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
	}
	expect |= script.ScriptVerifyCheckLockTimeVerify
	if flag := testChain.GetBlockScriptFlags(blockIdx[19]); flag != uint32(expect) {
		t.Errorf("GetBlockScriptFlags wrong: %d", flag)
	}
}

func TestBuildForwardTree(t *testing.T) {
	globalChain = nil
	InitGlobalChain()
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
		blockIdx[height] = getBlockIndex(blockIdx[height-1], timePerBlock, pow.BigToCompact(dummyPow))
		testChain.AddToIndexMap(blockIdx[height])
	}
	for height = 11; height < 21; height++ {
		blockIdx[height] = getBlockIndex(blockIdx[height-11], timePerBlock, initBits)
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
