package mining

import (
	"strconv"

	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/policy"
	"github.com/btcboost/copernicus/utils"
)

const DEFAULT_PRINTPRIORITY = false

type BlockTemplate struct {
	block         *core.Block
	txFees        []utils.Amount
	txSigOpsCount []int64
}

// Container for tracking updates to ancestor feerate as we include (parent)
// transactions in a block
type txMemPoolModifiedEntry struct {
	iter                    *mempool.TxMempoolEntry
	sizeWithAncestors       uint64
	modFeesWithAncestors    utils.Amount
	sigOpCountWithAncestors int64
}

func newTxMemPoolModifiedEntry(entry *mempool.TxMempoolEntry) {
	mEntry := new(txMemPoolModifiedEntry)
	mEntry.iter = entry
	mEntry.sizeWithAncestors = entry.GetsizeWithAncestors()
	mEntry.modFeesWithAncestors = entry.ModFeesWithAncestors
	mEntry.sigOpCountWithAncestors = entry.SigOpCountWithAncestors
}

// This matches the calculation in CompareTxMemPoolEntryByAncestorFee,
// except operating on CTxMemPoolModifiedEntry.
// TODO: refactor to avoid duplication of this logic.
func compareModifiedEntry(a, b *txMemPoolModifiedEntry) bool {
	f1 := b.sizeWithAncestors * uint64(a.modFeesWithAncestors)
	f2 := a.sizeWithAncestors * uint64(b.modFeesWithAncestors)

	if f1 == f2 {
		return a.iter.TxRef.Hash.ToBigInt().Cmp(b.iter.TxRef.Hash.ToBigInt()) < 0
	}
	return f1 > f2
}

// CompareTxIterByAncestorCount A comparator that sorts transactions based on number of ancestors.
// This is sufficient to sort an ancestor package in an order that is valid
// to appear in a block.
func CompareTxIterByAncestorCount(a, b mempool.TxMempoolEntry) bool {
	if a.SigOpCountWithAncestors != b.SigOpCountWithAncestors {
		return a.SigOpCountWithAncestors < b.SigOpCountWithAncestors
	}
	return a.TxRef.Hash.ToBigInt().Cmp(b.TxRef.Hash.ToBigInt()) < 0
}

// BlockAssembler Generate a new block, without valid proof-of-work
type BlockAssembler struct {
	bt                    *BlockTemplate
	block                 *core.Block
	maxGeneratedBlockSize uint64
	blockMinFeeRate       utils.FeeRate
	blockSize             uint64
	blockTx               uint64
	blockSigOps           uint64
	fees                  utils.Amount
	inBlock               []*mempool.TxMempoolEntry
	height                int
	lockTimeCutoff        int64
	chainParams           *msg.BitcoinParams
	lastFewTxs            int
	blockFinished         bool
}

func ScoreCompare(a, b *mempool.TxMempoolEntry) bool {
	return mempool.CompareTxMempoolEntryByScore(b, a)
}

func UpdateTime(bh *core.BlockHeader, params *msg.BitcoinParams, indexPrev *core.BlockIndex) int64 {
	oldTime := int64(bh.Time)
	var newTime int64
	mt := indexPrev.GetMedianTimePast() + 1
	at := utils.GetAdjustedTime()
	if mt > at {
		newTime = mt
	} else {
		newTime = at
	}
	if oldTime < newTime {
		bh.Time = uint32(newTime)
	}

	// Updating time can change work required on testnet:
	if params.FPowAllowMinDifficultyBlocks {
		pow := blockchain.Pow{}
		bh.Bits = pow.GetNextWorkRequired(indexPrev, bh, params)
	}
	return newTime - oldTime
}

func ComputeMaxGeneratedBlockSize(indexPrev *core.BlockIndex) uint64 {
	// Block resource limits
	// If -blockmaxsize is not given, limit to DEFAULT_MAX_GENERATED_BLOCK_SIZE
	// If only one is given, only restrict the specified resource.
	// If both are given, restrict both.
	maxGeneratedBlockSize := uint64(utils.GetArg("-blockmaxsize", int64(policy.DefaultMaxGeneratedBlockSize)))

	// Limit size to between 1K and MaxBlockSize-1K for sanity:
	csize := policy.DefaultMaxBlockSize - 1000
	if csize < maxGeneratedBlockSize {
		maxGeneratedBlockSize = csize
	}
	if 1000 > maxGeneratedBlockSize {
		maxGeneratedBlockSize = 1000
	}
	return maxGeneratedBlockSize
}

func NewBlockAssembler(params *msg.BitcoinParams) *BlockAssembler {
	ba := new(BlockAssembler)
	ba.chainParams = params
	v := utils.GetArg("-blockmintxfee", int64(policy.DefaultBlockMinTxFee))
	ba.blockMinFeeRate = utils.NewFeeRate(v) // todo confirm
	ba.maxGeneratedBlockSize = ComputeMaxGeneratedBlockSize(core.ActiveChain.Tip())
	return ba
}

func (ba *BlockAssembler) resetBlock() {
	ba.inBlock = nil
	// Reserve space for coinbase tx.
	ba.blockSize = 1000
	ba.blockSigOps = 1000

	// These counters do not include coinbase tx.
	ba.blockTx = 0
	ba.fees = 0
	ba.lastFewTxs = 0
	ba.blockFinished = false
}

func getExcessiveBlockSizeSig() []byte {
	cbmsg := "/EB" + getSubVersionEB(consensus.DefaultMaxBlockSize) + "/"
	return []byte(cbmsg)
}

// This function convert MaxBlockSize from byte to
// MB with a decimal precision one digit rounded down
// E.g.
// 1660000 -> 1.6
// 2010000 -> 2.0
// 1000000 -> 1.0
// 230000  -> 0.2
// 50000   -> 0.0
// NB behavior for EB<1MB not standardized yet still
// the function applies the same algo used for
// EB greater or equal to 1MB
func getSubVersionEB(maxBlockSize uint64) string {
	// Prepare EB string we are going to add to SubVer:
	// 1) translate from byte to MB and convert to string
	// 2) limit the EB string to the first decimal digit (floored)
	v := int(maxBlockSize / (consensus.OneMegabyte))
	toStr := strconv.Itoa(v)
	ret := v / 10
	if ret <= 0 {
		return "0." + toStr
	}
	length := len(toStr)
	return toStr[:length-1] + "." + toStr[length-1:]
}

func (ba *BlockAssembler) CreateNewBlock(script core.Script) *BlockTemplate {
	// timeStart := utils.GetMockTimeInMicros()
	//
	// ba.resetBlock()
	return nil
}
