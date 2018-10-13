package lchain

import (
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/util"
	"math/big"
	"math/rand"
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

func initEnv() {
	conf.Cfg = &conf.Configuration{}
	conf.Cfg.BlockIndex.CheckBlockIndex = true

	chain.InitGlobalChain()
	gChain := chain.GetInstance()
	gChain.SetTip(nil)

	GlobalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)
	gChain.InitLoad(GlobalBlockIndexMap, branch)

	persist.InitPersistGlobal()

	bl := gChain.GetParams().GenesisBlock
	bIndex := blockindex.NewBlockIndex(&bl.Header)
	bIndex.Height = 0
	bIndex.TxCount = 1
	bIndex.ChainTxCount = 1
	bIndex.AddStatus(blockindex.BlockHaveData)
	bIndex.RaiseValidity(blockindex.BlockValidTransactions)
	err := gChain.AddToIndexMap(bIndex)
	if err != nil {
		panic("AddToIndexMap fail")
	}
}

func TestCheckBlockIndex_OnlyGenesis(t *testing.T) {
	initEnv()
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
	initEnv()
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
			panic("AddToIndexMap fail")
		}
	}

	gChain.SetTip(blockIdx[9])
	print(gChain.Height())
	print(gChain.IndexMapSize())
	if err := CheckBlockIndex(); err != nil {
		t.Errorf("should return nil:%v", err)
	}
}
