package mining

import (
	"fmt"
	"sort"
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
	"gopkg.in/fatih/set.v0"
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
	inBlock               set.Set // element type: *mempool.TxMempoolEntry
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
	ba.inBlock.Clear()
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

// This transaction selection algorithm orders the mempool based on feerate of a
// transaction including all unconfirmed ancestors. Since we don't remove
// transactions from the mempool as we select them for block inclusion, we need
// an alternate method of updating the feerate of a transaction with its
// not-yet-selected ancestors as we go. This is accomplished by walking the
// in-mempool descendants of selected transactions and storing a temporary
// modified state in mapModifiedTxs. Each time through the loop, we compare the
// best transaction in mapModifiedTxs with the next transaction in the mempool
// to decide what transaction package to work on next.
func (ba *BlockAssembler) addPackageTxs(packagesSelected, descendantsUpdated int) {

}

func (ba *BlockAssembler) isStillDependent(te *mempool.TxMempoolEntry) bool {
	setParent := blockchain.GMemPool.GetMemPoolParents(te)
	plist := setParent.List()
	for _, item := range plist {
		parent := item.(*mempool.TxMempoolEntry)
		if !ba.inBlock.Has(parent) {
			return true
		}
	}
	return false
}

func (ba *BlockAssembler) onlyUnconfirmed(testSet *set.Set) {
	testSet.Each(func(item interface{}) bool {
		iit := item.(*mempool.TxMempoolEntry)
		// Only test txs not already in the block.
		if ba.inBlock.Has(iit) {
			testSet.Remove(iit)
		}
		return true
	})
}

func (ba *BlockAssembler) TestPackage(packageSize uint64, packageSigops int64) bool {
	blockSizeWithPackage := ba.blockSize + packageSize
	if blockSizeWithPackage >= ba.maxGeneratedBlockSize {
		return false
	}
	if ba.blockSigOps+uint64(packageSigops) >= consensus.GetMaxBlockSigOpsCount(blockSizeWithPackage) {
		return false
	}
	return true
}

// Perform transaction-level checks before adding to block:
// - transaction finality (locktime)
// - serialized size (in case -blockmaxsize is in use)
func (ba *BlockAssembler) testPackageTransactions(pkg *set.Set) bool {
	potentialBlockSize := ba.blockSize
	pkg.Each(func(item interface{}) bool {
		state := core.ValidationState{}
		it := item.(*mempool.TxMempoolEntry)
		if !blockchain.ContextualCheckTransaction(ba.chainParams, it.TxRef, &state, ba.height, ba.lockTimeCutoff) {
			return false
		}

		txSize := it.TxRef.SerializeSize()
		if potentialBlockSize+uint64(txSize) >= ba.maxGeneratedBlockSize {
			return false
		}

		potentialBlockSize += uint64(txSize)
		return true
	})
	return true
}

func (ba *BlockAssembler) testForBlock(te *mempool.TxMempoolEntry) bool {
	blockSizeWithTx := ba.blockSize + uint64(te.TxRef.SerializeSize())
	if blockSizeWithTx >= ba.maxGeneratedBlockSize {
		if ba.blockSize > ba.maxGeneratedBlockSize-100 || ba.lastFewTxs > 50 {
			ba.blockFinished = true
			return false
		}
		if ba.blockSize > ba.maxGeneratedBlockSize-1000 {
			ba.lastFewTxs++
		}
		return false
	}

	maxBlockSigOps := consensus.GetMaxBlockSigOpsCount(blockSizeWithTx)
	if ba.blockSigOps+uint64(te.SigOpCount) >= maxBlockSigOps {
		// If the block has room for no more sig ops then flag that the block is
		// finished.
		// TODO: We should consider adding another transaction that isn't very
		// dense in sigops instead of bailing out so easily.
		if ba.blockSigOps > maxBlockSigOps-2 {
			ba.blockFinished = true
			return false
		}
		// Otherwise attempt to find another tx with fewer sigops to put in the
		// block.
		return false
	}

	// Must check that lock times are still valid. This can be removed once MTP
	// is always enforced as long as reorgs keep the mempool consistent.
	state := core.ValidationState{}
	return blockchain.ContextualCheckTransaction(ba.chainParams, te.TxRef, &state, ba.height, ba.lockTimeCutoff)
}

func (ba *BlockAssembler) addToBlock(te *mempool.TxMempoolEntry) {
	ba.block.Txs = append(ba.block.Txs, te.TxRef)
	ba.bt.txFees = append(ba.bt.txFees, te.Fee)
	ba.bt.txSigOpsCount = append(ba.bt.txSigOpsCount, te.SigOpCount)
	ba.blockSize += uint64(te.TxSize)
	ba.blockTx++
	ba.blockSigOps += uint64(te.SigOpCount)
	ba.fees += te.Fee
	ba.inBlock.Add(te)
}

func (ba *BlockAssembler) UpdatePackagesForAdded(alreadyAdded *set.Set, mapModifiedTx *set.Set) int {
	descendantsUpdated := 0
	alreadyAdded.Each(func(item interface{}) bool {
		it := item.(*mempool.TxMempoolEntry)
		descendants := set.New() // element type: *mempool.TxMempoolEntry
		blockchain.GMemPool.CalculateDescendants(it, descendants)
		// Insert all descendants (not yet in block) into the modified set.
		descendants.Each(func(ele interface{}) bool {
			desc := ele.(*mempool.TxMempoolEntry)
			if alreadyAdded.Has(desc) {
				// do nothing
			} else {
				descendantsUpdated++
				// mit := mapModifiedTx
				// todo complete
			}

			return true
		})
		return true
	})

	return 0
}

// mapModifiedTx (which implies that the mapTx ancestor state is stale due to
// ancestor inclusion in the block). Also skip transactions that we've already
// failed to add. This can happen if we consider a transaction in mapModifiedTx
// and it fails: we can then potentially consider it again while walking mapTx.
// It's currently guaranteed to fail again, but as a belt-and-suspenders check
// we put it in failedTx and avoid re-evaluation, since the re-evaluation would
// be using cached size/sigops/fee values that are not actually correct.
func (ba *BlockAssembler) skipMapTxEntry(it *mempool.TxMempoolEntry, mapModifiedTx *set.Set, failedTx *set.Set) bool {
	return true
}

func (ba *BlockAssembler) sortForBlock(pkg *set.Set, entry *mempool.TxMempoolEntry) []*mempool.TxMempoolEntry {
	// Sort package by ancestor count. If a transaction A depends on transaction
	// B, then A's ancestor count must be greater than B's. So this is
	// sufficient to validly order the transactions for block inclusion.
	sortedEntries := make([]*mempool.TxMempoolEntry, pkg.Size())
	i := 0
	pkg.Each(func(item interface{}) bool {
		it := item.(*mempool.TxMempoolEntry)
		sortedEntries[i] = it
		return true
	})

	sort.SliceStable(sortedEntries, func(i, j int) bool {
		if sortedEntries[i].CountWithAncestors != sortedEntries[j].CountWithAncestors {
			return sortedEntries[i].CountWithAncestors < sortedEntries[j].CountWithAncestors
		}
		return !utils.CompareByHash(sortedEntries[i].TxRef.Hash, sortedEntries[j].TxRef.Hash)
	})
	return sortedEntries
}
