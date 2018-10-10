package mining

import (
	"bytes"
	"encoding/binary"
	"math"
	"strconv"
	"sync"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblock"
	"github.com/copernet/copernicus/logic/lmerkleroot"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/versionbits"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"

	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/google/btree"
)

const (
	// Limit the number of attempts to add transactions to the block when it is
	// close to full; this is just a simple heuristic to finish quickly if the
	// mempool has a lot of entries.
	maxConsecutiveFailures = 1000
	CoinbaseFlag           = ""
)

// global value for getmininginfo rpc use
var (
	lastBlockTx   uint64
	lastBlockSize uint64
)

func GetLastBlockTx() uint64 {
	return lastBlockTx
}

func GetLastBlockSize() uint64 {
	return lastBlockSize
}

type BlockTemplate struct {
	Block         *block.Block
	TxFees        []amount.Amount
	TxSigOpsCount []int
}

func newBlockTemplate() *BlockTemplate {
	return &BlockTemplate{
		Block:         block.NewBlock(),
		TxFees:        make([]amount.Amount, 0),
		TxSigOpsCount: make([]int, 0),
	}
}

// BlockAssembler Generate a new block, without valid proof-of-work
type BlockAssembler struct {
	sync.RWMutex
	bt                    *BlockTemplate
	maxGeneratedBlockSize uint64
	blockMinFeeRate       util.FeeRate
	blockSize             uint64
	blockTx               uint64
	blockSigOps           uint64
	fees                  amount.Amount
	inBlock               map[util.Hash]struct{}
	height                int32
	lockTimeCutoff        int64
	chainParams           *model.BitcoinParams
}

func NewBlockAssembler(params *model.BitcoinParams) *BlockAssembler {
	ba := new(BlockAssembler)
	ba.bt = newBlockTemplate()
	ba.chainParams = params
	v := conf.Cfg.Mining.BlockMinTxFee
	ba.blockMinFeeRate = *util.NewFeeRate(v) // todo confirm
	ba.maxGeneratedBlockSize = computeMaxGeneratedBlockSize()
	return ba
}

func (ba *BlockAssembler) resetBlockAssembler() {
	ba.inBlock = make(map[util.Hash]struct{})
	// Reserve space for coinbase tx.
	ba.blockSize = 1000
	ba.blockSigOps = 100

	// These counters do not include coinbase tx.
	ba.blockTx = 0
	ba.fees = 0
}

func (ba *BlockAssembler) testPackage(packageSize uint64, packageSigOps int64, add *tx.Tx) bool {
	blockSizeWithPackage := ba.blockSize + packageSize
	if blockSizeWithPackage >= ba.maxGeneratedBlockSize {
		return false
	}
	if ba.blockSigOps+uint64(packageSigOps) >= consensus.GetMaxBlockSigOpsCount(blockSizeWithPackage) {
		return false
	}
	return true
}

func (ba *BlockAssembler) addToBlock(te *mempool.TxEntry) {
	ba.bt.Block.Txs = append(ba.bt.Block.Txs, te.Tx)
	ba.bt.TxFees = append(ba.bt.TxFees, amount.Amount(te.TxFee))
	ba.bt.TxSigOpsCount = append(ba.bt.TxSigOpsCount, te.SigOpCount)
	ba.blockSize += uint64(te.TxSize)
	ba.blockTx++
	ba.blockSigOps += uint64(te.SigOpCount)
	ba.fees += amount.Amount(te.TxFee)
	ba.inBlock[te.Tx.GetHash()] = struct{}{}
}

func computeMaxGeneratedBlockSize() uint64 {
	// Block resource limits
	// If -blockmaxsize is not given, limit to DEFAULT_MAX_GENERATED_BLOCK_SIZE
	// If only one is given, only restrict the specified resource.
	// If both are given, restrict both.
	maxGeneratedBlockSize := conf.Cfg.Mining.BlockMaxSize

	// Limit size to between 1K and MaxBlockSize-1K for sanity:
	csize := consensus.DefaultMaxBlockSize - 1000
	if uint64(csize) < maxGeneratedBlockSize {
		maxGeneratedBlockSize = uint64(csize)
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
	pool := mempool.GetInstance() // todo use global variable
	tmpStrategy := *getStrategy()

	consecutiveFailed := 0

	var txSet *btree.BTree
	switch tmpStrategy {
	case sortByFee:
		txSet = sortedByFeeWithAncestors()
	case sortByFeeRate:
		txSet = sortedByFeeRateWithAncestors()
	}

	//pendingTx := make(map[util.Hash]mempool.TxEntry)
	failedTx := make(map[util.Hash]mempool.TxEntry)
	for txSet.Len() > 0 {
		// select the max value item, and delete it. select strategy is descent.
		var entry mempool.TxEntry

		switch tmpStrategy {
		case sortByFee:
			entry = mempool.TxEntry(txSet.Max().(EntryFeeSort))
			txSet.DeleteMax()
		case sortByFeeRate:
			entry = mempool.TxEntry(txSet.Max().(EntryAncestorFeeRateSort))
			txSet.DeleteMax()
		}
		// if inBlock has the item, continue next loop
		if _, ok := ba.inBlock[entry.Tx.GetHash()]; ok {
			continue
		}
		// if the item has failed in packing into the block, continue next loop
		if _, ok := failedTx[entry.Tx.GetHash()]; ok {
			continue
		}

		packageSize := entry.SumTxSizeWitAncestors
		packageFee := entry.SumTxFeeWithAncestors
		packageSigOps := entry.SumTxSigOpCountWithAncestors

		// deal with several different mining strategies
		isEnd := false
		switch tmpStrategy {
		case sortByFee:
			// if the current fee lower than the specified min fee rate, stop loop directly.
			// because the following after this item must be lower than this
			if packageFee < ba.blockMinFeeRate.GetFee(int(packageSize)) {
				isEnd = true
			}
		case sortByFeeRate:
			currentFeeRate := util.NewFeeRateWithSize(packageFee, packageSize)
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

		pool.RLock()
		ancestors, _ := pool.CalculateMemPoolAncestors(entry.Tx, noLimit, noLimit, noLimit, noLimit, true)
		pool.RUnlock()

		ba.onlyUnconfirmed(ancestors)
		ancestors[&entry] = struct{}{} // add current item
		if !ba.testPackageTransactions(ancestors) {
			continue
		}

		// This transaction will make it in; reset the failed counter.
		consecutiveFailed = 0
		addset := make(map[util.Hash]mempool.TxEntry)
		for add := range ancestors {
			ba.addToBlock(add)
			addset[add.Tx.GetHash()] = *add
		}

		descendantsUpdated += ba.updatePackagesForAdded(txSet, ancestors)
	}
	return descendantsUpdated
}

func (ba *BlockAssembler) CreateNewBlock(coinbaseScript *script.Script) *BlockTemplate {
	timeStart := util.GetMockTimeInMicros()

	ba.resetBlockAssembler()

	// add dummy coinbase tx as first transaction
	ba.bt.Block.Txs = make([]*tx.Tx, 0, 100000)
	ba.bt.Block.Txs = append(ba.bt.Block.Txs, tx.NewTx(0, tx.DefaultVersion))
	ba.bt.TxFees = make([]amount.Amount, 0, 100000)
	ba.bt.TxFees = append(ba.bt.TxFees, -1)
	ba.bt.TxSigOpsCount = make([]int, 0, 100000)
	ba.bt.TxSigOpsCount = append(ba.bt.TxSigOpsCount, -1)

	ba.Lock()
	defer ba.Unlock()
	indexPrev := chain.GetInstance().Tip()

	// genesis block
	if indexPrev == nil {
		ba.height = 0
	} else {
		ba.height = indexPrev.Height + 1
	}

	blkVersion := versionbits.ComputeBlockVersion(indexPrev, model.ActiveNetParams, versionbits.VBCache)
	ba.bt.Block.Header.Version = int32(blkVersion)
	// -regtest only: allow overriding block.nVersion with
	// -blockversion=N to test forking scenarios
	if ba.chainParams.MineBlocksOnDemands {
		if conf.Cfg.Mining.BlockVersion != -1 {
			ba.bt.Block.Header.Version = conf.Cfg.Mining.BlockVersion
		}
	}
	ba.bt.Block.Header.Time = uint32(util.GetAdjustedTime())
	ba.maxGeneratedBlockSize = computeMaxGeneratedBlockSize()
	ba.lockTimeCutoff = indexPrev.GetMedianTimePast()

	descendantsUpdated := ba.addPackageTxs()
	time1 := util.GetMockTimeInMicros()

	// record last mining info for getmininginfo rpc using
	lastBlockTx = ba.blockTx
	lastBlockSize = ba.blockSize

	// Create coinbase transaction
	coinbaseTx := tx.NewTx(0, tx.DefaultVersion)

	outPoint := outpoint.OutPoint{Hash: util.HashZero, Index: 0xffffffff}
	heightNumb := script.NewScriptNum(int64(ba.height))

	//sc := script.NewScriptRaw(buf.Bytes())
	sc := script.NewEmptyScript()
	sc.PushScriptNum(heightNumb)
	coinbaseTx.AddTxIn(txin.NewTxIn(&outPoint, sc, 0xffffffff))

	// value represents total reward(fee and block generate reward)
	value := ba.fees + GetBlockSubsidy(ba.height, ba.chainParams)
	coinbaseTx.AddTxOut(txout.NewTxOut(value, coinbaseScript))
	ba.bt.Block.Txs[0] = coinbaseTx
	ba.bt.TxFees[0] = -1 * ba.fees // coinbase's fee item is equal to tx fee sum for negative value

	serializeSize := ba.bt.Block.SerializeSize()
	log.Info("CreateNewBlock(): total size: %d txs: %d fees: %d sigops %d\n",
		serializeSize, ba.blockTx, ba.fees, ba.blockSigOps)

	// Fill in header.
	if indexPrev == nil {
		ba.bt.Block.Header.HashPrevBlock = util.HashZero
	} else {
		ba.bt.Block.Header.HashPrevBlock = *indexPrev.GetBlockHash()
	}
	UpdateTime(ba.bt.Block, indexPrev)
	p := pow.Pow{}
	ba.bt.Block.Header.Bits = p.GetNextWorkRequired(indexPrev, &ba.bt.Block.Header, ba.chainParams)
	ba.bt.Block.Header.Nonce = 0
	ba.bt.TxSigOpsCount[0] = ba.bt.Block.Txs[0].GetSigOpCountWithoutP2SH()

	//check the validity of the block
	if !TestBlockValidity(ba.bt.Block, indexPrev) {
		log.Error("CreateNewBlock: TestBlockValidity failed.")
		return nil
	}

	time2 := util.GetMockTimeInMicros()
	log.Print("bench", "debug", "CreateNewBlock() packages: %.2fms (%d packages, %d "+
		"updated descendants), validity: %.2fms (total %.2fms)\n", 0.001*float64(time1-timeStart),
		ba.blockTx, descendantsUpdated, 0.001*float64(time2-time1), 0.001*float64(time2-timeStart))

	return ba.bt
}

func (ba *BlockAssembler) onlyUnconfirmed(entrySet map[*mempool.TxEntry]struct{}) {
	for entry := range entrySet {
		if _, ok := ba.inBlock[entry.Tx.GetHash()]; ok {
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
		err := ltx.ContextualCheckTransaction(entry.Tx, ba.height, ba.lockTimeCutoff)
		if err != nil {
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
	mpool := mempool.GetInstance()
	tmpStrategy := *getStrategy()

	for entry := range alreadyAdded {
		descendants := make(map[*mempool.TxEntry]struct{})
		mpool.CalculateDescendants(entry, descendants)
		// Insert all descendants (not yet in block) into the modified set.
		// use reflect function if there are so many strategies
		for desc := range descendants {
			descendantUpdate++
			switch tmpStrategy {
			case sortByFee:
				item := EntryFeeSort(*desc)
				// remove the old one
				txSet.Delete(item)
				// update origin data
				desc.SumTxSizeWitAncestors -= entry.SumTxSizeWitAncestors
				desc.SumTxFeeWithAncestors -= entry.SumTxFeeWithAncestors
				desc.SumTxSigOpCountWithAncestors -= entry.SumTxSigOpCountWithAncestors
				// insert the modified one
				txSet.ReplaceOrInsert(item)
			case sortByFeeRate:
				item := EntryAncestorFeeRateSort(*desc)
				// remove the old one
				txSet.Delete(item)
				// update origin data
				desc.SumTxSizeWitAncestors -= entry.SumTxSizeWitAncestors
				desc.SumTxFeeWithAncestors -= entry.SumTxFeeWithAncestors
				desc.SumTxSigOpCountWithAncestors -= entry.SumTxSigOpCountWithAncestors
				// insert the modified one
				txSet.ReplaceOrInsert(item)
			}
		}
	}
	return descendantUpdate
}

func IncrementExtraNonce(bk *block.Block, bindex *blockindex.BlockIndex) (extraNonce uint) {
	// Update nExtraNonce
	if bk.Header.HashPrevBlock != util.HashZero {
		extraNonce = 0
	}
	extraNonce++
	// Height first in coinbase required for block.version=2
	height := bindex.Height + 1

	// TODO lack of lscript builder to construct script conveniently<lscript>
	buf := bytes.NewBuffer(nil)
	bytesEight := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytesEight, uint64(height))
	buf.Write(bytesEight)

	binary.LittleEndian.PutUint64(bytesEight, uint64(extraNonce))
	buf.Write(bytesEight)

	buf.Write(getExcessiveBlockSizeSig())
	buf.Write([]byte(CoinbaseFlag))

	coinbaseScript := script.NewScriptRaw(buf.Bytes())
	bk.Txs[0].UpdateInScript(0, coinbaseScript)
	bk.UpdateSerializeSize()

	bk.Header.MerkleRoot = lmerkleroot.BlockMerkleRoot(bk.Txs, nil)

	return extraNonce
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
	v := int(maxBlockSize / (consensus.OneMegaByte))
	toStr := strconv.Itoa(v)
	ret := v / 10
	if ret <= 0 {
		return "0." + toStr
	}
	length := len(toStr)
	return toStr[:length-1] + "." + toStr[length-1:]
}

func getExcessiveBlockSizeSig() []byte {
	cbmsg := "/EB" + getSubVersionEB(consensus.DefaultMaxBlockSize) + "/"
	return []byte(cbmsg)
}

func UpdateTime(bk *block.Block, indexPrev *blockindex.BlockIndex) int64 {
	oldTime := int64(bk.Header.Time)
	var newTime int64
	mt := int64(0) + 1
	at := util.GetAdjustedTime()
	if mt > at {
		newTime = mt
	} else {
		newTime = at
	}
	if oldTime < newTime {
		bk.Header.Time = uint32(newTime)
	}

	// Updating time can change work required on testnet:
	if model.ActiveNetParams.FPowAllowMinDifficultyBlocks {
		p := pow.Pow{}
		bk.Header.Bits = p.GetNextWorkRequired(indexPrev, &bk.Header, model.ActiveNetParams)
	}

	return newTime - oldTime
}

func GetBlockSubsidy(height int32, params *model.BitcoinParams) amount.Amount {
	halvings := height / params.SubsidyReductionInterval
	// Force block reward to zero when right shift is undefined.
	if halvings >= 64 {
		return 0
	}

	nSubsidy := amount.Amount(50 * util.COIN)
	// Subsidy is cut in half every 210,000 blocks which will occur
	// approximately every 4 years.
	return amount.Amount(uint(nSubsidy) >> uint(halvings))
}

func TestBlockValidity(block *block.Block, indexPrev *blockindex.BlockIndex) bool {
	persist.CsMain.Lock()
	defer persist.CsMain.Unlock()

	if !(indexPrev != nil && indexPrev == chain.GetInstance().Tip()) {
		log.Error("TestBlockValidity(): error")
		return false
	}

	if !lblockindex.CheckIndexAgainstCheckpoint(indexPrev) {
		log.Error("mining: CheckIndexAgainstCheckpoint() failed, please check.")
		return false
	}

	coinMap := utxo.NewEmptyCoinsMap()
	coinMap.GetMap()
	blkHeader := block.GetBlockHeader()
	indexDummy := blockindex.NewBlockIndex(&blkHeader)
	indexDummy.Prev = indexPrev
	indexDummy.Height = indexPrev.Height + 1

	// NOTE: CheckBlockHeader is called by CheckBlock
	if !lblock.ContextualCheckBlockHeader(&blkHeader, indexPrev, util.GetAdjustedTime()) {
		log.Error("TestBlockValidity(): Consensus::ContextualCheckBlockHeader failed, please check.")
		return false
	}

	if err := lblock.CheckBlock(block, false, false); err != nil {
		log.Error("TestBlockValidity(): Consensus::CheckBlock failed: %v", err)
		return false
	}

	if err := lblock.ContextualCheckBlock(block, indexPrev); err != nil {
		log.Error("TestBlockValidity(): Consensus::ContextualCheckBlock failed: %v", err)
		return false
	}

	if err := lchain.ConnectBlock(block, indexDummy, coinMap, true); err != nil {
		log.Error("trying to connect to the block failed:%v", err)
		return false
	}

	return true
}
