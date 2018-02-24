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
var (
	//GChainState Global unique variables
	GChainState          ChainState
	GfCheckpointsEnabled = DEFAULT_CHECKPOINTS_ENABLED
	GfCheckBlockIndex    = false
	GfRequireStandard    = true
	GfIsBareMultisigStd  = DEFAULT_PERMIT_BAREMULTISIG
)

var (
	// GfHavePruned Pruning-related variables and constants, True if any block files have ever been pruned.
	GfHavePruned = false
	GfPruneMode  = false
	GfTxIndex    = false
	GfReIndex    = false
	//GindexBestHeader Best header we've seen so far (used for getheaders queries' starting points)
	GindexBestHeader *BlockIndex
)

func init() {
	GChainState.MapBlockIndex.Data = make(map[utils.Hash]*BlockIndex)
	GChainState.MapBlocksUnlinked = make(map[*BlockIndex][]*BlockIndex)
	GChainState.setBlockIndexCandidates = algorithm.NewCustomSet(BlockIndexWorkComparator)
}
