package service

import (
	"github.com/btcboost/copernicus/model/block"
	lchain "github.com/btcboost/copernicus/logic/chain"
	lblock "github.com/btcboost/copernicus/logic/block"
	"github.com/btcboost/copernicus/model/chain"
)


func ProcessBlockHeader(bl * block.BlockHeader) error {
	return nil
}

func ProcessBlock(b *block.Block) (bool, error) {
	gChain := chain.GetInstance()
	isNewBlock := false
	var err error

	bIndex := gChain.FindBlockIndex(b.Header.GetHash())
	if bIndex != nil {
		if bIndex.Accepted() {
			return isNewBlock,nil
		}
	}

	err = lblock.Check(b)
	if err != nil {
		return isNewBlock, err
	}

	bIndex,err = lchain.AcceptBlock(b)
	if err != nil {
		return isNewBlock, err
	}

	isNewBlock = true
	err = gChain.ActiveBest(bIndex)
	if err != nil {
		return isNewBlock, err
	}

	return isNewBlock, err
}
