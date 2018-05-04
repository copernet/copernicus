package msghandle

import (
	"copernicus/core"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/mempool"
)

func ProcessNewblock(b *core.Block, isNewBlock * bool) bool  {

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