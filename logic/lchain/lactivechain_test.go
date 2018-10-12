package lchain

import (
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/util"
	"testing"
)

func TestCheckBlockIndex_NoCheck(t *testing.T) {
	conf.Cfg = &conf.Configuration{}
	conf.Cfg.BlockIndex.CheckBlockIndex = false
	if err := CheckBlockIndex(); err != nil {
		t.Errorf("NoCheck should do nothing and return nil:%v", err)
	}
}

func TestCheckBlockIndex_OnlyGenesis(t *testing.T) {
	conf.Cfg = &conf.Configuration{}
	conf.Cfg.BlockIndex.CheckBlockIndex = true

	chain.InitGlobalChain()
	gChain := chain.GetInstance()

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
		t.Errorf("AddToIndexMap fail:%v", err)
	}

	gChain.SetTip(bIndex)

	if err := CheckBlockIndex(); err != nil {
		t.Errorf("should return nil:%v", err)
	}
}
