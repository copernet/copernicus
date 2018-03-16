package blockchain

import (
	"sync/atomic"

	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

type BlockMap struct {
	Data map[utils.Hash]*core.BlockIndex
}

// ChainState store the blockchain global state
type ChainState struct {
	ChainAcTive       core.Chain
	MapBlockIndex     BlockMap
	PindexBestInvalid *core.BlockIndex

	//* The set of all CBlockIndex entries with BLOCK_VALID_TRANSACTIONS (for itself
	//* and all ancestors) and as good as our current tip or better. Entries may be
	//* failed, though, and pruning nodes may be missing the data for the block.
	setBlockIndexCandidates *container.CustomSet

	// All pairs A->B, where A (or one of its ancestors) misses transactions, but B
	// has transactions. Pruned nodes may have entries where B is missing data.
	MapBlocksUnlinked map[*core.BlockIndex][]*core.BlockIndex
}

// Global status for blockchain
var (
	//GChainState Global unique variables
	GChainState       ChainState
	GfImporting       atomic.Value
	GMaxTipAge        int64
	Gmempool          *mempool.Mempool
	Pool              *mempool.TxMempool
	GpcoinsTip        *utxo.CoinsViewCache
	Gpblocktree       *BlockTreeDB
	GminRelayTxFee    utils.FeeRate
	GfReindex         = false
	GnCoinCacheUsage  = 5000 * 300
	Gwarningcache     []ThresholdConditionCache
	Gversionbitscache *VersionBitsCache
)

var (
	// GfHavePruned Pruning-related variables and constants, True if any block files have ever been pruned.
	GfHavePruned = false
	GfPruneMode  = false
	GfTxIndex    = false

	//GindexBestHeader Best header we've seen so far (used for getheaders queries' starting points)
	GindexBestHeader *core.BlockIndex
	//GChainActive currently-connected chain of blocks (protected by cs_main).
	GChainActive core.Chain
	GPruneTarget uint64

	//GfCheckForPruning Global flag to indicate we should check to see if there are block/undo files
	//* that should be deleted. Set on startup or if we allocate more file space when
	//* we're in prune mode.
	GfCheckForPruning    = false
	GfCheckpointsEnabled = DEFAULT_CHECKPOINTS_ENABLED
	GfCheckBlockIndex    = false
	GfRequireStandard    = true
	GfIsBareMultisigStd  = DEFAULT_PERMIT_BAREMULTISIG
)

const (
	// MAX_BLOCKFILE_SIZE The maximum size of a blk?????.dat file (since 0.8)  // 128 MiB
	MAX_BLOCKFILE_SIZE = 0x8000000
	// BLOCKFILE_CHUNK_SIZE The pre-allocation chunk size for blk?????.dat files (since 0.8)  // 16 MiB
	BLOCKFILE_CHUNK_SIZE = 0x1000000
	// UNDOFILE_CHUNK_SIZE The pre-allocation chunk size for rev?????.dat files (since 0.8) // 1 MiB

	UNDOFILE_CHUNK_SIZE                   = 0x100000
	DEFAULT_MIN_RELAY_TX_FEE utils.Amount = 1000

	// DB_PEAK_USAGE_FACTOR compensate for extra memory peak (x1.5-x1.9) at flush time.
	DB_PEAK_USAGE_FACTOR = 2
	// MAX_BLOCK_COINSDB_USAGE no need to periodic flush if at least this much space still available.
	MAX_BLOCK_COINSDB_USAGE = 200 * DB_PEAK_USAGE_FACTOR
	// MIN_BLOCK_COINSDB_USAGE always periodic flush if less than this much space still available.
	MIN_BLOCK_COINSDB_USAGE = 50 * DB_PEAK_USAGE_FACTOR
	// DATABASE_WRITE_INTERVAL time to wait (in seconds) between writing blocks/block index to disk.
	DATABASE_WRITE_INTERVAL = 60 * 60
	// DATABASE_FLUSH_INTERVAL time to wait (in seconds) between flushing chainstate to disk.
	DATABASE_FLUSH_INTERVAL = 24 * 60 * 60
)

func init() {
	GChainState.MapBlockIndex.Data = make(map[utils.Hash]*core.BlockIndex)
	GChainState.MapBlocksUnlinked = make(map[*core.BlockIndex][]*core.BlockIndex)
	GChainState.setBlockIndexCandidates = container.NewCustomSet(BlockIndexWorkComparator)
	GfImporting.Store(false)
	GMaxTipAge = DEFAULT_MAX_TIP_AGE
	GminRelayTxFee.SataoshisPerK = int64(DEFAULT_MIN_RELAY_TX_FEE)
	Gmempool = mempool.NewMemPool(GminRelayTxFee)
	Gwarningcache = NewWarnBitsCache(VERSIONBITS_NUM_BITS)
	Gversionbitscache = NewVersionBitsCache()
}
