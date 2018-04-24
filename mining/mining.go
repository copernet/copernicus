package mining

import (
	"fmt"
	"log"
	"math"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/policy"
	"github.com/btcboost/copernicus/utils"
	"github.com/google/btree"
)

const (
	// Limit the number of attempts to add transactions to the block when it is
	// close to full; this is just a simple heuristic to finish quickly if the
	// mempool has a lot of entries.
	maxConsecutiveFailures = 1000
)

// global value for getmininginfo rpc use
var (
	lastBlockTx   uint64
	lastBlockSize uint64
)

func getLastBlockTx() uint64 {
	return lastBlockTx
}

func getLastBlockSize() uint64 {
	return lastBlockSize
}

type BlockTemplate struct {
	*core.Block
	TxFees        []utils.Amount
	TxSigOpsCount []int
}

func newBlockTemplate() *BlockTemplate {
	return &BlockTemplate{
		Block:         core.NewBlock(),
		TxFees:        make([]utils.Amount, 0),
		TxSigOpsCount: make([]int, 0),
	}
}

// BlockAssembler Generate a new block, without valid proof-of-work
type BlockAssembler struct {
	bt                    *BlockTemplate
	maxGeneratedBlockSize uint64
	blockMinFeeRate       utils.FeeRate
	blockSize             uint64
	blockTx               uint64
	blockSigOps           uint64
	fees                  utils.Amount
	inBlock               map[*mempool.TxEntry]struct{} // todo modify key to value pattern instead of pointer pattern
	height                int
	lockTimeCutoff        int64
	chainParams           *msg.BitcoinParams
}

func NewBlockAssembler(params *msg.BitcoinParams) *BlockAssembler {
	ba := new(BlockAssembler)
	ba.bt = newBlockTemplate()
	ba.chainParams = params
	v := utils.GetArg("-blockmintxfee", int64(policy.DefaultBlockMinTxFee))
	ba.blockMinFeeRate = *utils.NewFeeRate(v) // todo confirm
	ba.maxGeneratedBlockSize = computeMaxGeneratedBlockSize(core.ActiveChain.Tip())
	return ba
}

func (ba *BlockAssembler) resetBlockAssembler() {
	ba.inBlock = make(map[*mempool.TxEntry]struct{})
	// Reserve space for coinbase tx.
	ba.blockSize = 1000
	ba.blockSigOps = 100

	// These counters do not include coinbase tx.
	ba.blockTx = 0
	ba.fees = 0
}

func (ba *BlockAssembler) testPackage(packageSize uint64, packageSigops int64, add *core.Tx) bool {
	blockSizeWithPackage := ba.blockSize + packageSize
	if blockSizeWithPackage >= ba.maxGeneratedBlockSize {
		return false
	}
	if ba.blockSigOps+uint64(packageSigops) >= consensus.GetMaxBlockSigOpsCount(blockSizeWithPackage) {
		return false
	}
	return true
}

func (ba *BlockAssembler) addToBlock(te *mempool.TxEntry) {
	ba.bt.Block.Txs = append(ba.bt.Block.Txs, te.Tx)
	ba.bt.TxFees = append(ba.bt.TxFees, utils.Amount(te.TxFee))
	ba.bt.TxSigOpsCount = append(ba.bt.TxSigOpsCount, te.SigOpCount)
	ba.blockSize += uint64(te.TxSize)
	ba.blockTx++
	ba.blockSigOps += uint64(te.SigOpCount)
	ba.fees += utils.Amount(te.TxFee)
	ba.inBlock[te] = struct{}{}
}

func computeMaxGeneratedBlockSize(indexPrev *core.BlockIndex) uint64 {
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

// This transaction selection algorithm orders the mempool based on feerate of a
// transaction including all unconfirmed ancestors. Since we don't remove
// transactions from the mempool as we select them for block inclusion, we need
// an alternate method of updating the feerate of a transaction with its
// not-yet-selected ancestors as we go.
func (ba *BlockAssembler) addPackageTxs() int {
	descendantsUpdated := 0
	pool := blockchain.GMemPool
	pool.RLock()
	defer pool.RUnlock()

	consecutiveFailed := 0

	txSet := btree.New(32) // todo confirm 32
	switch strategy {
	case sortByFee:
		for _, entry := range pool.PoolData {
			txSet.ReplaceOrInsert(EntryFeeSort(*entry))
		}
	case sortByFeeRate:
		for _, entry := range pool.PoolData {
			txSet.ReplaceOrInsert(EntryAncestorFeeRateSort(*entry))
		}
	}

	//pendingTx := make(map[utils.Hash]mempool.TxEntry)
	failedTx := make(map[utils.Hash]mempool.TxEntry)
	ba.updatePackagesForAdded(txSet, ba.inBlock)

	for {
		// select the max value item, and delete it. select strategy is descent.
		entry := mempool.TxEntry(txSet.DeleteMax().(EntryAncestorFeeRateSort))
		// if inBlock has the item, continue next loop
		if _, ok := ba.inBlock[&entry]; ok {
			continue
		}
		// if the item has failed in packing into the block, continue next loop
		if _, ok := failedTx[entry.Tx.Hash]; ok {
			continue
		}

		packageSize := entry.SumSizeWitAncestors
		packageFee := entry.SumFeeWithAncestors
		packageSigOps := entry.SumSigOpCountWithAncestors

		// deal with several different mining strategies
		isEnd := false
		switch strategy {
		case sortByFee:
			// if the current fee lower than the specified min fee rate, stop loop directly.
			// because the following after this item must be lower than this
			if packageFee < ba.blockMinFeeRate.GetFee(int(packageSize)) {
				isEnd = true
			}
		case sortByFeeRate:
			currentFeeRate := utils.NewFeeRateWithSize(packageFee, packageSize)
			if currentFeeRate.Less(ba.blockMinFeeRate) {
				isEnd = true
			}
		}
		if isEnd {
			break
		}

		if !ba.testPackage(uint64(packageSize), packageSigOps, nil) {
			consecutiveFailed++
			if consecutiveFailed > maxConsecutiveFailures &&
				ba.blockSize > ba.maxGeneratedBlockSize-1000 {
				// Give up if we're close to full and haven't succeeded in a while.
				break
			}
			continue
		}

		// add the ancestors of the current item to block
		noLimit := uint64(math.MaxUint64)
		ancestors, _ := pool.CalculateMemPoolAncestors(entry.Tx, noLimit, noLimit, noLimit, noLimit, false)
		ba.onlyUnconfirmed(ancestors)
		ancestors[&entry] = struct{}{} // add current item
		if !ba.testPackageTransactions(ancestors) {
			continue
		}

		// This transaction will make it in; reset the failed counter.
		consecutiveFailed = 0
		addset := make(map[utils.Hash]mempool.TxEntry)
		for add := range ancestors {
			ba.addToBlock(add)
			addset[add.Tx.Hash] = *add
		}
		descendantsUpdated += ba.updatePackagesForAdded(txSet, ancestors)
	}
	return descendantsUpdated
}

func (ba *BlockAssembler) CreateNewBlock() *BlockTemplate {
	timeStart := utils.GetMockTimeInMicros()

	ba.resetBlockAssembler()

	// add dummy coinbase tx as first transaction
	ba.bt.Block.Txs = make([]*core.Tx, 0, 100000)
	ba.bt.Block.Txs = append(ba.bt.Block.Txs, core.NewTx())
	ba.bt.TxFees = make([]utils.Amount, 0, 100000)
	ba.bt.TxFees = append(ba.bt.TxFees, -1)
	ba.bt.TxSigOpsCount = make([]int, 0, 100000)
	ba.bt.TxSigOpsCount = append(ba.bt.TxSigOpsCount, -1)

	// todo LOCK2(cs_main);
	indexPrev := core.ActiveChain.Tip()
	ba.height = indexPrev.Height + 1
	ba.bt.Block.BlockHeader.Version = int32(blockchain.ComputeBlockVersion(indexPrev, msg.ActiveNetParams, nil)) // todo deal with nil param
	// -regtest only: allow overriding block.nVersion with
	// -blockversion=N to test forking scenarios
	if ba.chainParams.MineBlocksOnDemands {
		ba.bt.Block.BlockHeader.Version = int32(utils.GetArg("-blockversion", int64(ba.bt.Block.BlockHeader.Version)))
	}
	ba.bt.Block.BlockHeader.Time = uint32(utils.GetAdjustedTime())
	ba.maxGeneratedBlockSize = computeMaxGeneratedBlockSize(indexPrev)
	if consensus.StandardLocktimeVerifyFlags&consensus.LocktimeMedianTimePast != 0 {
		ba.lockTimeCutoff = indexPrev.GetMedianTimePast()
	} else {
		ba.lockTimeCutoff = int64(ba.bt.Block.BlockHeader.Time)
	}

	descendantsUpdated := ba.addPackageTxs()

	time1 := utils.GetMockTimeInMicros()

	// record last mining info for getmininginfo rpc using
	lastBlockTx = ba.blockTx
	lastBlockSize = ba.blockSize

	// Create coinbase transaction
	coinbaseTx := core.NewTx()
	coinbaseTx.Ins = make([]*core.TxIn, 1)
	sig := core.Script{}
	sig.PushInt64(int64(ba.height))
	sig.PushOpCode(core.OP_0)
	coinbaseTx.Ins[0] = core.NewTxIn(&core.OutPoint{Hash: utils.HashZero, Index: 0xffffffff}, sig.GetScriptByte())
	coinbaseTx.Outs = make([]*core.TxOut, 1)

	// value represents total reward(fee and block generate reward)
	value := ba.fees + blockchain.GetBlockSubsidy(ba.height, ba.chainParams)
	coinbaseTx.Outs[0] = core.NewTxOut(int64(value), []byte{core.OP_1})
	ba.bt.Block.Txs[0] = coinbaseTx
	ba.bt.TxFees[0] = -1 * ba.fees // coinbase's fee item is equal to tx fee sum for negative value

	serializeSize := ba.bt.Block.SerializeSize()
	logs.Info("CreateNewBlock(): total size: %d txs: %d fees: %d sigops %d\n",
		serializeSize, ba.blockTx, ba.fees, ba.blockSigOps)

	// Fill in header.
	ba.bt.Block.BlockHeader.HashPrevBlock = *indexPrev.GetBlockHash()
	ba.bt.Block.UpdateTime(indexPrev)
	pow := blockchain.Pow{}
	ba.bt.Block.BlockHeader.Bits = pow.GetNextWorkRequired(indexPrev, &ba.bt.Block.BlockHeader, ba.chainParams)
	ba.bt.Block.BlockHeader.Nonce = 0
	ba.bt.TxSigOpsCount[0] = ba.bt.Block.Txs[0].GetSigOpCountWithoutP2SH()

	state := core.ValidationState{}
	if !blockchain.TestBlockValidity(ba.chainParams, &state, ba.bt.Block, indexPrev, false, false) {
		panic(fmt.Sprintf("CreateNewBlock(): TestBlockValidity failed: %s", state.FormatStateMessage()))
	}

	time2 := utils.GetMockTimeInMicros()
	log.Print("bench", "debug", "CreateNewBlock() packages: %.2fms (%d packages, %d "+
		"updated descendants), validity: %.2fms (total %.2fms)\n", 0.001*float64(time1-timeStart),
		ba.blockTx, descendantsUpdated, 0.001*float64(time2-time1), 0.001*float64(time2-timeStart))

	return ba.bt
}

func (ba *BlockAssembler) onlyUnconfirmed(entrySet map[*mempool.TxEntry]struct{}) {
	for entry := range entrySet {
		if _, ok := ba.inBlock[entry]; ok {
			delete(entrySet, entry)
		}
	}
}

// Perform transaction-level checks before adding to block:
// - transaction finality (locktime)
// - serialized size (in case -blockmaxsize is in use)
func (ba *BlockAssembler) testPackageTransactions(entrySet map[*mempool.TxEntry]struct{}) bool {
	potentialBlockSize := ba.blockSize
	for entry := range entrySet {
		state := core.ValidationState{}
		if blockchain.ContextualCheckTransaction(ba.chainParams, entry.Tx, &state, ba.height, ba.lockTimeCutoff) {
			return false
		}

		if potentialBlockSize+uint64(entry.TxSize) >= ba.maxGeneratedBlockSize {
			return false
		}
		potentialBlockSize += uint64(entry.TxSize)
	}

	return true
}

func (ba *BlockAssembler) updatePackagesForAdded(txSet *btree.BTree, alreadyAdded map[*mempool.TxEntry]struct{}) int {
	descendantUpdate := 0
	for entry := range alreadyAdded {
		descendants := make(map[*mempool.TxEntry]struct{})
		blockchain.GMemPool.CalculateDescendants(entry, descendants)
		// Insert all descendants (not yet in block) into the modified set.
		// use reflect function if there are so many strategies
		for desc := range descendants {
			descendantUpdate++
			switch strategy {
			case sortByFee:
				item := EntryFeeSort(*desc)
				// remove the old one
				txSet.Delete(item)
				// update origin data
				desc.SumSizeWitAncestors -= entry.SumSizeWitAncestors
				desc.SumFeeWithAncestors -= entry.SumFeeWithAncestors
				desc.SumSigOpCountWithAncestors -= entry.SumSigOpCountWithAncestors
				item = EntryFeeSort(*desc)
				// insert the modified one
				txSet.ReplaceOrInsert(item)
			case sortByFeeRate:
				item := EntryAncestorFeeRateSort(*desc)
				// remove the old one
				txSet.Delete(item)
				// update origin data
				desc.SumSizeWitAncestors -= entry.SumSizeWitAncestors
				desc.SumFeeWithAncestors -= entry.SumFeeWithAncestors
				desc.SumSigOpCountWithAncestors -= entry.SumSigOpCountWithAncestors
				item = EntryAncestorFeeRateSort(*desc)
				// insert the modified one
				txSet.ReplaceOrInsert(item)
			}
		}
	}
	return descendantUpdate
}
