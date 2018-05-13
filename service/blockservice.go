package service

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/chain"
)

func ProcessNewBlock(b *block.Block) (bool,error) {

	gChain := chain.GetInstance()
	isNewBlock := false

	bindex := gChain.FindBlockIndex(b.Header.GetHash())
	missData := bindex.Status & MISS_DATA
	if bindex != nil && missData == false {
		return false,nil
	}

	return isNewBlock,nil
}
