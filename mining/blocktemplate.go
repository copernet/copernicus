package mining

import (
	"fmt"
	"strconv"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/policy"
	"github.com/btcboost/copernicus/utils"
)

var (
	lastBlockTx   = uint64(0)
	lastBlockSize = uint64(0)
)

const DEFAULT_PRINTPRIORITY = false

type BlockTemplate struct {
	block         *core.Block
	txFees        []utils.Amount
	txSigOpsCount []int64
}

func newBlockTemplate() *BlockTemplate {
	return &BlockTemplate{
		block:         core.NewBlock(),
		txFees:        make([]utils.Amount, 0),
		txSigOpsCount: make([]int64, 0),
	}
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

func UpdateTime(bl *core.Block, params *msg.BitcoinParams, indexPrev *core.BlockIndex) int64 {
	oldTime := int64(bl.BlockHeader.Time)
	var newTime int64
	mt := indexPrev.GetMedianTimePast() + 1
	at := utils.GetAdjustedTime()
	if mt > at {
		newTime = mt
	} else {
		newTime = at
	}
	if oldTime < newTime {
		bl.BlockHeader.Time = uint32(newTime)
	}

	// Updating time can change work required on testnet:
	if params.FPowAllowMinDifficultyBlocks {
		pow := blockchain.Pow{}
		bl.BlockHeader.Bits = pow.GetNextWorkRequired(indexPrev, &bl.BlockHeader, params)
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
	timeStart := utils.GetMockTimeInMicros()

	ba.resetBlock()
	ba.bt = newBlockTemplate()

	// Pointer for convenience.
	ba.block = ba.bt.block

	// add dummy coinbase tx as first transaction
	ba.block.Txs = make([]*core.Tx, 1)
	ba.block.Txs[0] = core.NewTx()

	// updated at end
	ba.bt.txFees = append(ba.bt.txFees, -1)
	ba.bt.txSigOpsCount = append(ba.bt.txSigOpsCount, -1)

	// todo LOCK2(cs_main, mempool.cs);
	indexPrev := core.ActiveChain.Tip()
	ba.height = indexPrev.Height + 1

	ba.block.BlockHeader.Version = int32(blockchain.ComputeBlockVersion(indexPrev, msg.ActiveNetParams, nil))
	// -regtest only: allow overriding block.nVersion with
	// -blockversion=N to test forking scenarios
	if ba.chainParams.MineBlocksOnDemands {
		ba.block.BlockHeader.Version = int32(utils.GetArg("-blockversion", int64(ba.block.BlockHeader.Version)))
	}

	ba.block.BlockHeader.Time = uint32(utils.GetAdjustedTime())

	ba.maxGeneratedBlockSize = ComputeMaxGeneratedBlockSize(indexPrev)

	if consensus.StandardLocktimeVerifyFlags&consensus.LocktimeMedianTimePast != 0 {
		ba.lockTimeCutoff = indexPrev.GetMedianTimePast()
	} else {
		ba.lockTimeCutoff = int64(ba.block.BlockHeader.Time)
	}
	addPriorityTxs()
	packagesSelected := 0
	descendantsUpdated := 0
	addPackageTxs(packagesSelected, descendantsUpdated)

	time1 := utils.GetMockTimeInMicros()
	lastBlockTx = ba.blockTx
	lastBlockSize = ba.blockSize

	// Create coinbase transaction
	coinbaseTx := core.NewTx()
	coinbaseTx.Ins = make([]*core.TxIn, 1)
	// outpoint := core.OutPoint{
	// 	Hash:  utils.HashZero,
	// 	Index: -1,
	// }
	// coinbaseTx.Ins[0] = core.NewTxIn(&outpoint, nil)
	s := core.NewScriptRaw([]byte{})
	s.PushScriptNum(core.NewCScriptNum(int64(ba.height)))
	s.PushOpCode(core.OP_0)
	coinbaseTx.Ins[0].Script = s

	coinbaseTx.Outs = make([]*core.TxOut, 1)
	value := ba.fees + blockchain.GetBlockSubsidy(ba.height, ba.chainParams)
	coinbaseTx.Outs[0] = core.NewTxOut(int64(value), script.GetScriptByte())

	ba.block.Txs = make([]*core.Tx, 1)
	ba.block.Txs[0] = coinbaseTx

	ba.bt.txFees = append(ba.bt.txFees, -1*ba.fees)
	serializeSize := ba.block.SerializeSize()
	logs.Info("CreateNewBlock(): total size: %d txs: %d fees: %d sigops %d\n",
		serializeSize, ba.blockTx, ba.fees, ba.blockSigOps)

	// Fill in header.
	ba.block.BlockHeader.HashPrevBlock = *indexPrev.GetBlockHash()
	UpdateTime(ba.block, ba.chainParams, indexPrev)

	pow := blockchain.Pow{}
	ba.block.BlockHeader.Bits = pow.GetNextWorkRequired(indexPrev, &ba.block.BlockHeader, ba.chainParams)
	ba.block.BlockHeader.Nonce = 0
	ba.bt.txSigOpsCount[0] = int64(ba.block.Txs[0].GetSigOpCountWithoutP2SH())

	state := core.ValidationState{}
	if !blockchain.TestBlockValidity(ba.chainParams, &state, ba.block, indexPrev, false, false) {
		panic(fmt.Sprintf("CreateNewBlock(): TestBlockValidity failed: %s", state.FormatStateMessage()))
	}

	time2 := utils.GetMockTimeInMicros()
	log.Print("bench", "debug", "CreateNewBlock() packages: %.2fms (%d packages, %d "+
		"updated descendants), validity: %.2fms (total %.2fms)\n", 0.001*float64(time1-timeStart),
		packagesSelected, descendantsUpdated, 0.001*float64(time2-time1), 0.001*float64(time2-timeStart))

	return ba.bt
}

func addPriorityTxs() {

}

func addPackageTxs(packagesSelected, descendantsUpdated int) {

}
