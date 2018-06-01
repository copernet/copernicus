package chain

import (
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/util"
)

func NewFakeChain() *Chain {
	c := Chain{
		active:      make([]*blockindex.BlockIndex, 0),
		branch:      make([]*blockindex.BlockIndex, 0),
		waitForTx:   make(map[util.Hash]*blockindex.BlockIndex),
		orphan:      make(map[util.Hash][]*blockindex.BlockIndex, 0),
		indexMap:    make(map[util.Hash]*blockindex.BlockIndex),
		newestBlock: nil,
		receiveID:   0,
	}
	c.params = chainparams.ActiveNetParams
	
	genbi := blockindex.NewBlockIndex(&c.params.GenesisBlock.Header)
	c.active = append(c.active, genbi)
	c.branch = append(c.branch, genbi)

	return &c
}
