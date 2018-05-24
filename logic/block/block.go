package block

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/blockindex"
)

func Check(b * block.Block) error {
	return nil
}

func GetBlock(hash *util.Hash) (* block.Block, error) {
	return nil,nil
}

func Write(bi *blockindex.BlockIndex, b *block.Block) error {
	return nil
}