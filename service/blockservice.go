package service

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/chain"
	lblock "github.com/btcboost/copernicus/logic/block"
)

func ProcessBlock(b *block.Block) (bool,error) {
	gChain := chain.GetInstance()
	isNewBlock := false
	haveData := false
	var err error

	bIndex := gChain.FindBlockIndex(b.Header.GetHash())
	if bIndex != nil {
		haveData = bIndex.HaveData()
		if haveData {
			return isNewBlock,nil
		}
	}

	err = lblock.Check(b)
	if err != nil {
		return isNewBlock,err
	}

	replaceIndexFlag := false
	if bIndex != nil && haveData == false {
		replaceIndexFlag = true
	}
	bIndex,err = gChain.AcceptBlock(b,replaceIndexFlag)
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
