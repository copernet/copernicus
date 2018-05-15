package service

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/chain"
	lblock "github.com/btcboost/copernicus/logic/block"
)

func ProcessBlock(b *block.Block) (bool,error) {
	gChain := chain.GetInstance()
	isNewBlock := false
	accepted := false
	var err error

	bIndex := gChain.FindBlockIndex(b.Header.GetHash())
	if bIndex != nil {
		accepted = bIndex.Accepted()
		if accepted {
			return isNewBlock,nil
		}
	}

	err = lblock.Check(b)
	if err != nil {
		return isNewBlock,err
	}

	bIndex,err = gChain.AcceptBlock(b)
	if err != nil {
		return isNewBlock,err
	}

	isNewBlock = true
	err = gChain.ActiveBest(bIndex)
	if err != nil {
		return isNewBlock,err
	}

	return isNewBlock,err
}
