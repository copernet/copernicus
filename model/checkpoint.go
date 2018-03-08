package model

import (
	"github.com/btcboost/copernicus/utils"
)

type Checkpoint struct {
	Height int32
	Hash   *utils.Hash
}

// GetLastCheckpoint returns last CBlockIndex* in mapBlockIndex that is a checkpoint
func GetLastCheckpoint(data []*Checkpoint) *BlockIndex {
	//for _, v := range data {
	//	hash := v.Hash
	//	t, ok := blockchain.MapBlockIndex.Data[*hash]
	//	if ok {
	//		return t
	//	}
	//}
	return nil
}
