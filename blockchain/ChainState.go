package blockchain

import (
	"github.com/btcboost/copernicus/utils"
)

type BlockMap struct {
	Data map[utils.Hash]*BlockIndex
}

// ChainState store the blockchain global state
type ChainState struct {
	ChainAcTive       Chain
	MapBlockIndex     BlockMap
	MapBlocksUnlinked map[*BlockIndex][]*BlockIndex
	PindexBestInvalid *BlockIndex
}

// Global status for blockchain

//GChainState Global unique variables
var GChainState ChainState

var GCheckpointsEnabled = DEFAULT_CHECKPOINTS_ENABLED

// GindexBestHeader Best header we've seen so far (used for getheaders queries' starting points)
var GindexBestHeader *BlockIndex

func init() {
	GChainState.MapBlockIndex.Data = make(map[utils.Hash]*BlockIndex)
	GChainState.MapBlocksUnlinked = make(map[*BlockIndex][]*BlockIndex)
}
