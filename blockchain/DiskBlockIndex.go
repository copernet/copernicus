package blockchain

import (
	"github.com/btcboost/copernicus/utils"
	"io"
)

type DiskBlockIndex struct {
	BlockIndex
	hashPrev utils.Hash
}

func (diskBlcokIndex *DiskBlockIndex) Serialize(wirter io.Writer) error {
	return nil
}

func NewDiskBlockIndex(pindex *BlockIndex) *DiskBlockIndex {
	return nil
}
