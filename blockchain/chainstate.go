package blockchain

import (
	"sync/atomic"

	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

type BlockMap struct {
	Data map[utils.Hash]*core.BlockIndex
}

// ChainState store the blockChain global state
type ChainState struct {
	ChainActive      core.Chain
	MapBlockIndex    BlockMap
	IndexBestInvalid *core.BlockIndex

	// The set of all CBlockIndex entries with BLOCK_VALID_TRANSACTIONS (for itself
	// and all ancestors) and as good as our current tip or better. Entries may be
	// failed, though, and pruning nodes may be missing the data for the block.
	setBlockIndexCandidates *container.CustomSet

	// All pairs A->B, where A (or one of its ancestors) misses transactions, but B
	// has transactions. Pruned nodes may have entries where B is missing data.
	MapBlocksUnlinked map[*core.BlockIndex][]*core.BlockIndex
}

// Global status for blockChain
var (
	// GChainState Global unique variables
	GChainState       ChainState
	GImporting        atomic.Value
	GMaxTipAge        int64
	GMemPool          *mempool.Mempool
	GCoinsTip         *utxo.CoinsViewCache
	GBlockTree        *BlockTreeDB
	GMinRelayTxFee    utils.FeeRate
	Pool              *mempool.TxMempool
	GfReindex         = false
	GnCoinCacheUsage  = 5000 * 300
	GWarningCache     []ThresholdConditionCache
	GVersionBitsCache *VersionBitsCache
)

var (
	// GHavePruned pruning-related variables and constants, True if any block files have ever been pruned.
	GHavePruned = false
	GPruneMode  = false
	GTxIndex    = false

	//GIndexBestHeader Best header we've seen so far (used for getHeaders queries' starting points)
	GIndexBestHeader *core.BlockIndex
	//GChainActive currently-connected chain of blocks (protected by cs_main).
	GChainActive core.Chain
	GPruneTarget uint64

	//GCheckForPruning Global flag to indicate we should check to see if there are block/undo files
	//* that should be deleted. Set on startup or if we allocate more file space when
	//* we're in prune mode.
	GCheckForPruning    = false
	GCheckpointsEnabled = consensus.DefaultCheckPointsEnabled
	GCheckBlockIndex    = false
	GRequireStandard    = true
	GIsBareMultiSigStd  = consensus.DefaultPermitBareMultiSig
)

const (
	// MaxBlockFileSize the maximum size of a blk?????.dat file (since 0.8)  // 128 MiB
	MaxBlockFileSize = 0x8000000
	// BlockFileChunkSize the pre-allocation chunk size for blk?????.dat files (since 0.8)  // 16 MiB
	BlockFileChunkSize = 0x1000000
	// UndoFileChunkSize the pre-allocation chunk size for rev?????.dat files (since 0.8) // 1 MiB
	UndoFileChunkSize                 = 0x100000
	DefaultMinRelayTxFee utils.Amount = 1000

	// DBPeakUsageFactor compensate for extra memory peak (x1.5-x1.9) at flush time.
	DBPeakUsageFactor = 2
	// MaxBlockCoinsDBUsage no need to periodic flush if at least this much space still available.
	MaxBlockCoinsDBUsage = 200 * DBPeakUsageFactor
	// MinBlockCoinsDBUsage always periodic flush if less than this much space still available.
	MinBlockCoinsDBUsage = 50 * DBPeakUsageFactor
	// DataBaseWriteInterval time to wait (in seconds) between writing blocks/block index to disk.
	DataBaseWriteInterval = 60 * 60
	// DataBaseFlushInterval time to wait (in seconds) between flushing chainState to disk.
	DataBaseFlushInterval = 24 * 60 * 60
)

func init() {
	GChainState.MapBlockIndex.Data = make(map[utils.Hash]*core.BlockIndex)
	GChainState.MapBlocksUnlinked = make(map[*core.BlockIndex][]*core.BlockIndex)
	GChainState.setBlockIndexCandidates = container.NewCustomSet(BlockIndexWorkComparator)
	GImporting.Store(false)
	GMaxTipAge = consensus.DefaultMaxTipAge
	GMinRelayTxFee.SataoshisPerK = int64(DefaultMinRelayTxFee)
	GMemPool = mempool.NewMemPool(GMinRelayTxFee)
	GWarningCache = NewWarnBitsCache(VersionBitsNumBits)
	GVersionBitsCache = NewVersionBitsCache()
}
