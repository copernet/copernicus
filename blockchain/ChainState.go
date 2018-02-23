package blockchain

import (
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/utils"
)

type BlockMap struct {
	Data map[utils.Hash]*BlockIndex
}

// ChainState store the blockchain global state
type ChainState struct {
	ChainAcTive       Chain
	MapBlockIndex     BlockMap
	PindexBestInvalid *BlockIndex

	//* The set of all CBlockIndex entries with BLOCK_VALID_TRANSACTIONS (for itself
	//* and all ancestors) and as good as our current tip or better. Entries may be
	//* failed, though, and pruning nodes may be missing the data for the block.
	setBlockIndexCandidates *algorithm.CustomSet

	// All pairs A->B, where A (or one of its ancestors) misses transactions, but B
	// has transactions. Pruned nodes may have entries where B is missing data.
	MapBlocksUnlinked map[*BlockIndex][]*BlockIndex
}

// Global status for blockchain

//GChainState Global unique variables
var GChainState ChainState
var GfCheckpointsEnabled = DEFAULT_CHECKPOINTS_ENABLED
var GfCheckBlockIndex = false
var GfRequireStandard = true
var GfIsBareMultisigStd = DEFAULT_PERMIT_BAREMULTISIG

// GfHavePruned Pruning-related variables and constants, True if any block files have ever been pruned.
var GfHavePruned = false
var GfPruneMode = false
var GfTxIndex = false
var GfReIndex = false

// GindexBestHeader Best header we've seen so far (used for getheaders queries' starting points)
var GindexBestHeader *BlockIndex

func init() {
	GChainState.MapBlockIndex.Data = make(map[utils.Hash]*BlockIndex)
	GChainState.MapBlocksUnlinked = make(map[*BlockIndex][]*BlockIndex)
	GChainState.setBlockIndexCandidates = algorithm.NewCustomSet(BlockIndexWorkComparator)
}
