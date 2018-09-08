package chain

import (
	"github.com/copernet/copernicus/model/blockindex"

	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chainparams"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/util"
	"math/big"
	"testing"
)

var testChain *Chain

func getBlockIndex(indexPrev *blockindex.BlockIndex, timeInterval int64, bits uint32) *blockindex.BlockIndex {
	blockIdx := new(blockindex.BlockIndex)
	blockIdx.Prev = indexPrev
	blockIdx.Height = indexPrev.Height + 1
	blockIdx.Header.Time = indexPrev.Header.Time + uint32(timeInterval)
	blockIdx.Header.Bits = bits
	blockIdx.ChainWork = *big.NewInt(0).Add(&indexPrev.ChainWork, pow.GetBlockProof(blockIdx))
	return blockIdx
}

func TestChain(t *testing.T) {
	InitGlobalChain()
	testChain = GetInstance()
	testChain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	blockIdx := make([]*blockindex.BlockIndex, 50)
	initBits := chainparams.ActiveNetParams.PowLimitBits
	timePerBlock := int64(chainparams.ActiveNetParams.TargetTimePerBlock)
	height := 0

	// Pile up some blocks.
	blockIdx[height] = blockindex.NewBlockIndex(&chainparams.ActiveNetParams.GenesisBlock.Header)
	testChain.AddToBranch(blockIdx[0])
	testChain.AddToIndexMap(blockIdx[0])
	testChain.active = append(testChain.active, blockIdx[0])

	for height = 1; height < 11; height++ {
		blockIdx[height] = getBlockIndex(blockIdx[height-1], timePerBlock, initBits)
		testChain.AddToBranch(blockIdx[height])
		testChain.AddToIndexMap(blockIdx[height])
		testChain.active = append(testChain.active, blockIdx[height])
	}
	for height = 11; height < 16; height++ {
		blockIdx[height] = getBlockIndex(blockIdx[height-1], timePerBlock, initBits)
		testChain.AddToBranch(blockIdx[height])
		testChain.AddToIndexMap(blockIdx[height])
	}

	if testChain.GetParams() != chainparams.ActiveNetParams {
		t.Errorf("GetParams "+
			"expect: %s, actual: %s", chainparams.ActiveNetParams.Name, testChain.GetParams().Name)
	}
	if testChain.Genesis() != blockIdx[0] {
		t.Errorf("Genesis "+
			"expect: %s, actual: %s", blockIdx[0].GetBlockHash(), testChain.Genesis().GetBlockHash())
	}
	for i := 0; i < 11; i++ {
		hash := blockIdx[i].GetBlockHash()
		actualBlockIdx := testChain.FindHashInActive(*hash)
		if actualBlockIdx != blockIdx[i] {
			t.Errorf("FindHashInActive "+
				"expect: %s, actual: %s", blockIdx[i].GetBlockHash(), actualBlockIdx.GetBlockHash())
		}
	}
	for i := 11; i < 16; i++ {
		hash := blockIdx[i].GetBlockHash()
		actualBlockIdx := testChain.FindHashInActive(*hash)
		if actualBlockIdx != nil {
			t.Errorf("active chain should not have this blockindex, pls check")
		}
	}
	for i := 0; i < 16; i++ {
		hash := blockIdx[i].GetBlockHash()
		actualBlockIdx := testChain.FindBlockIndex(*hash)
		if actualBlockIdx != blockIdx[i] {
			t.Errorf("FindBlockIndex "+
				"expect: %s, actual: %s", blockIdx[i].GetBlockHash(), actualBlockIdx.GetBlockHash())
		}
	}
	if testChain.Tip() != blockIdx[10] {
		t.Errorf("Tip "+
			"expect: %s, actual: %s", blockIdx[10].GetBlockHash(), testChain.Tip().GetBlockHash())
	}
	if testChain.TipHeight() != 10 {
		t.Errorf("TipHeight "+
			"expect: 10, actual: %d", testChain.TipHeight())
	}
	for i := 0; i < 16; i++ {
		hash := blockIdx[i].GetBlockHash()
		actualHeight := testChain.GetSpendHeight(hash)
		if actualHeight != -1 && actualHeight != blockIdx[i].Height+1 {
			t.Errorf("GetSpendHeight "+
				"expect: %d, actual: %d", blockIdx[i].Height+1, actualHeight)
		}
	}
	for i := 0; i < 11; i++ {
		actualBlockIdx := testChain.GetIndex(int32(i))
		if actualBlockIdx != blockIdx[i] {
			t.Errorf("GetIndex "+
				"expect: %s, actual: %s", blockIdx[i].GetBlockHash(), actualBlockIdx.GetBlockHash())
		}
	}
	for i := 11; i < 20; i++ {
		actualBlockIdx := testChain.GetIndex(int32(i))
		if actualBlockIdx != nil {
			t.Errorf("active chain should not have this blockindex, pls check")
		}
	}
	for i := 0; i < 11; i++ {
		if !testChain.Contains(blockIdx[i]) {
			t.Errorf("active chain should contains this blockindex, pls check")
		}
	}
	for i := 11; i < 20; i++ {
		if testChain.Contains(blockIdx[i]) {
			t.Errorf("active chain should not contains this blockindex, pls check")
		}
	}
	if testChain.FindFork(blockIdx[15]) != blockIdx[10] {
		t.Errorf("FindFork "+
			"expect: %s, actual: %s", blockIdx[10].GetBlockHash(), testChain.FindFork(blockIdx[15]).GetBlockHash())
	}
	for i := 0; i < 16; i++ {
		if !testChain.InBranch(blockIdx[i]) {
			t.Errorf("branch should contains this blockindex, pls check")
		}
	}
	if testChain.FindMostWorkChain() != blockIdx[15] {
		t.Errorf("FindMostWorkChain "+
			"expect: %s, actual: %s", blockIdx[15].GetBlockHash(), testChain.FindMostWorkChain().GetBlockHash())
	}
	if testChain.SetTip(blockIdx[14]); testChain.Tip() != blockIdx[14] {
		t.Errorf("SetTip "+
			"expect: %s, actual: %s", blockIdx[14].GetBlockHash(), testChain.Tip().GetBlockHash())
	}

}

func TestChain_AddToBranch(t *testing.T) {
	InitGlobalChain()
	testChain = GetInstance()
	testChain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	blockIdx := make([]*blockindex.BlockIndex, 50)
	initBits := chainparams.ActiveNetParams.PowLimitBits
	timePerBlock := int64(chainparams.ActiveNetParams.TargetTimePerBlock)
	dummyPow := big.NewInt(0).Rsh(chainparams.ActiveNetParams.PowLimit, uint(0))
	height := 0

	//Pile up some blocks
	blockIdx[height] = blockindex.NewBlockIndex(&chainparams.ActiveNetParams.GenesisBlock.Header)
	testChain.AddToBranch(blockIdx[0])
	for height = 1; height < 11; height++ {
		i := height
		dummyPow = big.NewInt(0).Rsh(chainparams.ActiveNetParams.PowLimit, uint(i))
		blockIdx[height] = getBlockIndex(blockIdx[height-1], timePerBlock, pow.BigToCompact(dummyPow))
		testChain.AddToBranch(blockIdx[height])
	}
	for height = 11; height < 21; height++ {
		blockIdx[height] = getBlockIndex(blockIdx[height-11], timePerBlock, initBits)
		testChain.AddToBranch(blockIdx[height])
	}

	if len(testChain.branch) != 26 {
		t.Errorf("some block does not addtobranch, pls check")
	}
	if testChain.FindMostWorkChain() != blockIdx[10] {
		t.Errorf("some block does not sort by work, pls check")
	}
	if testChain.RemoveFromBranch(blockIdx[0]); testChain.InBranch(blockIdx[0]) {
		t.Errorf("block should not in branch")
	}
}

func TestChain_GetBlockScriptFlags(t *testing.T) {
	InitGlobalChain()
	testChain = GetInstance()
	timePerBlock := int64(chainparams.ActiveNetParams.TargetTimePerBlock)
	initBits := chainparams.ActiveNetParams.PowLimitBits

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
	blockIdx[0].Height = chainparams.ActiveNetParams.BIP66Height
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
	blockIdx[0].Height = chainparams.ActiveNetParams.BIP65Height
	for i := 1; i < 20; i++ {
		blockIdx[i] = getBlockIndex(blockIdx[i-1], timePerBlock, initBits)
	}
	expect |= script.ScriptVerifyCheckLockTimeVerify
	if flag := testChain.GetBlockScriptFlags(blockIdx[19]); flag != uint32(expect) {
		t.Errorf("GetBlockScriptFlags wrong: %d", flag)
	}
}
