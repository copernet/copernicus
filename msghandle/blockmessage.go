package msghandle

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/mempool"
)

func AcceptBlockHeader(bh * BlockHeader, bi ** BlockIndex) bool {



	return true
}

func AcceptBlock(b *Block, isNewBlock * bool) bool {

	if isNewBlock != nil {
		*isNewBlock = false
	}



	return true
}

func ProcessNewblock(b *Block, isNewBlock * bool) bool  {

	if isNewBlock != nil {
		*isNewBlock = false
	}

	ok := b.checkblock(b,CHECK_PARTIAL)
	if !ok {
		return false
	}

	ok := AcceptBlock(b,isNewBlock)
	if !ok {
		return false
	}

	return true
}