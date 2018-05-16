package chain

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/model/chain"
	lblock "github.com/btcboost/copernicus/logic/block"
)

func AcceptBlock(b * block.Block) (*blockindex.BlockIndex,error) {
	var err error
	var bIndex *blockindex.BlockIndex
	var c = chain.GetInstance()

	bIndex,err = AcceptBlockHeader(&b.Header)
	if err != nil {
		return nil,err
	}


	return nil,nil
}

func  AcceptBlockHeader(bh * block.BlockHeader) (*blockindex.BlockIndex,error) {
	var c = chain.GetInstance()

	bIndex := c.FindBlockIndex(bh.GetHash())
	if bIndex != nil {
		if bIndex.HeaderValid() == false {
			return nil,errcode.New(errcode.ErrorBlockHeaderNoValid)
		}
	} else {
		err := lblock.CheckBlockHeader(&bIndex.Header)
		if err != nil {
			return nil,err
		}

		bIndex = blockindex.NewBlockIndex(bh)
		bIndex.Prev = c.FindBlockIndex(bh.HashPrevBlock)
		if bIndex.Prev == nil {
			return nil,errcode.New(errcode.ErrorBlockHeaderNoParent)
		}
	}


	return nil,nil
}
