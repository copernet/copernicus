package mining

import (
	"math"
	"sort"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblock"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
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
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/model/versionbits"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/algorithm/mapcontainer"
	"github.com/copernet/copernicus/util/amount"
)

const (
	// Limit the number of attempts to add transactions to the block when it is
	// close to full; this is just a simple heuristic to finish quickly if the
	// mempool has a lot of entries.
	maxConsecutiveFailures = 1000
	CoinbaseFlag           = "copernicus"
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
	maxSigOps, errSig := consensus.GetMaxBlockSigOpsCount(blockSizeWithPackage)
	if errSig != nil {
		log.Error("testPackage err :%v", errSig)
		return false
	}
	if ba.blockSigOps+uint64(packageSigOps) >= maxSigOps {
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

type ByAncsCount []*mempool.TxEntry

func (a ByAncsCount) Len() int      { return len(a) }
func (a ByAncsCount) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAncsCount) Less(i, j int) bool {
	return a[i].SumTxCountWithAncestors < a[j].SumTxCountWithAncestors
}

type sortTxs []*tx.Tx

func (s sortTxs) Len() int {
	return len(s)
}
func (s sortTxs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortTxs) Less(i, j int) bool {
	h1 := s[i].GetHash()
	h2 := s[j].GetHash()
	return pow.HashToBig(&h1).Cmp(pow.HashToBig(&h2)) < 0
}

func sortTxsByAncestorCount(ancestors map[*mempool.TxEntry]struct{}) (result []*mempool.TxEntry) {

	result = make([]*mempool.TxEntry, 0, len(ancestors))
	for item := range ancestors {
		result = append(result, item)
	}

	sort.Sort(ByAncsCount(result))
	return result
}

// This transaction selection algorithm orders the mempool based on feerate of a
// transaction including all unconfirmed ancestors. Since we don't remove
// transactions from the mempool as we select them for block inclusion, we need
// an alternate method of updating the feerate of a transaction with its
// not-yet-selected ancestors as we go.
func (ba *BlockAssembler) addPackageTxs(sortRecord map[util.Hash]int) int {
	descendantsUpdated := 0
	pool := mempool.GetInstance() // todo use global variable
	pool.RLock()
	defer pool.RUnlock()
	tmpStrategy := *getStrategy()

	consecutiveFailed := 0

	var txSet mapcontainer.MapContainer
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
			less, _ := txSet.Max()
			entry = mempool.TxEntry(less.(EntryFeeSort))
			txSet.DeleteMax()
		case sortByFeeRate:
			less, _ := txSet.Max()
			entry = mempool.TxEntry(less.(EntryAncestorFeeRateSort))
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

		mempoolAncestors, _ := pool.CalculateMemPoolAncestors(entry.Tx, noLimit, noLimit, noLimit, noLimit, true)
		ancestors := make(map[*mempool.TxEntry]struct{})
		for en := range mempoolAncestors {
			newentry := *en
			ancestors[&newentry] = struct{}{}
		}

		ancestors[&entry] = struct{}{} // add current item
		ancestorsList := sortTxsByAncestorCount(ancestors)
		ancestorsList = ba.onlyUnconfirmed(ancestorsList)

		if !ba.testPackageTransactions(ancestorsList) {
			continue
		}

		// This transaction will make it in; reset the failed counter.
		consecutiveFailed = 0
		addset := make(map[util.Hash]mempool.TxEntry)

		for _, item := range ancestorsList {
			ba.addToBlock(item)
			sortRecord[item.Tx.GetHash()] = len(ba.bt.TxFees) - 1
			addset[item.Tx.GetHash()] = *item
		}

		descendantsUpdated += ba.updatePackagesForAdded(txSet, ancestorsList)
	}
	return descendantsUpdated
}

func BasicScriptSig() *script.Script {
	height := script.NewScriptNum(int64(chain.GetInstance().Tip().Height + 1))
	scriptSig := script.NewEmptyScript()
	scriptSig.PushScriptNum(height)
	return scriptSig
}

func (ba *BlockAssembler) CreateNewBlock(scriptPubKey, scriptSig *script.Script) *BlockTemplate {
	timeStart := util.GetTimeMicroSec()

	ba.resetBlockAssembler()

	// add dummy coinbase tx as first transaction
	ba.bt.Block.Txs = make([]*tx.Tx, 0, 100000)
	ba.bt.Block.Txs = append(ba.bt.Block.Txs, tx.NewTx(0, tx.DefaultVersion))
	ba.bt.TxFees = make([]amount.Amount, 0, 100000)
	ba.bt.TxFees = append(ba.bt.TxFees, -1)
	ba.bt.TxSigOpsCount = make([]int, 0, 100000)
	ba.bt.TxSigOpsCount = append(ba.bt.TxSigOpsCount, -1)

	indexPrev := chain.GetInstance().Tip()

	// genesis block
	if indexPrev == nil {
		ba.height = 0
	} else {
		ba.height = indexPrev.Height + 1
	}

	ba.bt.Block.Header.Version = versionbits.ComputeBlockVersion()
	// -regtest only: allow overriding block.nVersion with
	// -blockversion=N to test forking scenarios
	if ba.chainParams.MineBlocksOnDemands {
		if conf.Args.BlockVersion != -1 {
			ba.bt.Block.Header.Version = conf.Args.BlockVersion
		}
	}
	ba.bt.Block.Header.Time = uint32(util.GetAdjustedTimeSec())
	ba.maxGeneratedBlockSize = computeMaxGeneratedBlockSize()
	lockTimeCutoff := indexPrev.GetMedianTimePast()
	if tx.StandardLockTimeVerifyFlags&consensus.LocktimeMedianTimePast != 0 {
		ba.lockTimeCutoff = lockTimeCutoff
	} else {
		ba.lockTimeCutoff = int64(ba.bt.Block.GetBlockHeader().Time)
	}
	sortRecord := make(map[util.Hash]int)
	descendantsUpdated := ba.addPackageTxs(sortRecord)

	if model.IsMagneticAnomalyEnabled(indexPrev.GetMedianTimePast()) {
		// If magnetic anomaly is enabled, we make sure transaction are
		// canonically ordered.
		sort.Sort(sortTxs(ba.bt.Block.Txs[1:]))
		sortTxFees := make([]amount.Amount, len(ba.bt.TxFees))
		sortTxFees[0] = ba.bt.TxFees[0]
		sortTxSigOpCosts := make([]int, len(ba.bt.TxSigOpsCount))
		sortTxSigOpCosts[0] = ba.bt.TxSigOpsCount[0]
		for i, tmpTx := range ba.bt.Block.Txs[1:] {
			offset := sortRecord[tmpTx.GetHash()]
			sortTxFees[i+1] = ba.bt.TxFees[offset]
			sortTxSigOpCosts[i+1] = ba.bt.TxSigOpsCount[offset]
		}

		ba.bt.TxFees = sortTxFees
		ba.bt.TxSigOpsCount = sortTxSigOpCosts
	}
	time1 := util.GetTimeMicroSec()

	// record last mining info for getmininginfo rpc using
	lastBlockTx = ba.blockTx
	lastBlockSize = ba.blockSize

	// Create coinbase transaction
	coinbaseTx := tx.NewTx(0, tx.DefaultVersion)

	outPoint := outpoint.OutPoint{Hash: util.HashZero, Index: 0xffffffff}

	coinbaseTx.AddTxIn(txin.NewTxIn(&outPoint, scriptSig, 0xffffffff))
	coinbaseSerializeSize := coinbaseTx.SerializeSize()
	if coinbaseSerializeSize < consensus.MinTxSize {
		byteLen := consensus.MinTxSize - coinbaseSerializeSize - 1
		scriptSig.PushData(make([]byte, byteLen))
	}

	// value represents total reward(fee and block generate reward)

	value := ba.fees + model.GetBlockSubsidy(ba.height, ba.chainParams)
	coinbaseTx.AddTxOut(txout.NewTxOut(value, scriptPubKey))
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

	ba.bt.TxSigOpsCount[0] = ba.bt.Block.Txs[0].GetSigOpCountWithoutP2SH(uint32(script.StandardScriptVerifyFlags))

	//check the validity of the block
	if err := TestBlockValidity(ba.bt.Block, indexPrev, false, false); err != nil {
		log.Error("CreateNewBlock: TestBlockValidity failed, block is:%v, indexPrev:%v, err:%v", ba.bt.Block, indexPrev, err)
		return nil
	}

	time2 := util.GetTimeMicroSec()
	log.Print("bench", "debug", "CreateNewBlock() packages: %.2fms (%d packages, %d "+
		"updated descendants), validity: %.2fms (total %.2fms), txs number: %d\n", 0.001*float64(time1-timeStart),
		ba.blockTx, descendantsUpdated, 0.001*float64(time2-time1), 0.001*float64(time2-timeStart), len(ba.bt.Block.Txs))

	return ba.bt
}

func (ba *BlockAssembler) onlyUnconfirmed(entryList []*mempool.TxEntry) []*mempool.TxEntry {
	result := make([]*mempool.TxEntry, 0)
	for _, entry := range entryList {
		if _, ok := ba.inBlock[entry.Tx.GetHash()]; !ok {
			result = append(result, entry)
		}
	}
	return result
}

// Perform transaction-level checks before adding to block:
// - transaction finality (locktime)
// - serialized size (in case -blockmaxsize is in use)
func (ba *BlockAssembler) testPackageTransactions(entrySet []*mempool.TxEntry) bool {
	potentialBlockSize := ba.blockSize
	for _, entry := range entrySet {
		err := ltx.ContextualCheckTransaction(entry.Tx, ba.height, ba.lockTimeCutoff, ba.lockTimeCutoff)
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

func (ba *BlockAssembler) updatePackagesForAdded(txSet mapcontainer.MapContainer, alreadyAdded []*mempool.TxEntry) int {
	descendantUpdate := 0
	mpool := mempool.GetInstance()
	tmpStrategy := *getStrategy()

	for _, entry := range alreadyAdded {
		descendants := mpool.CalculateDescendants(entry)

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
				item.SumTxSizeWitAncestors -= entry.SumTxSizeWitAncestors
				item.SumTxFeeWithAncestors -= entry.SumTxFeeWithAncestors
				item.SumTxSigOpCountWithAncestors -= entry.SumTxSigOpCountWithAncestors
				// insert the modified one
				txSet.ReplaceOrInsert(item)
			case sortByFeeRate:
				item := EntryAncestorFeeRateSort(*desc)
				// remove the old one
				txSet.Delete(item)
				// update origin data
				item.SumTxSizeWitAncestors -= entry.SumTxSizeWitAncestors
				item.SumTxFeeWithAncestors -= entry.SumTxFeeWithAncestors
				item.SumTxSigOpCountWithAncestors -= entry.SumTxSigOpCountWithAncestors
				// insert the modified one
				txSet.ReplaceOrInsert(item)
			}
		}
	}
	return descendantUpdate
}

func CoinbaseScriptSig(extraNonce uint) *script.Script {
	scriptSig := script.NewEmptyScript()

	height := uint64(chain.GetInstance().Tip().Height + 1)
	heightNum := script.NewScriptNum(int64(height))
	scriptSig.PushScriptNum(heightNum)

	extraNonceNum := script.NewScriptNum(int64(extraNonce))
	scriptSig.PushScriptNum(extraNonceNum)

	scriptSig.PushData(append(getExcessiveBlockSizeSig(), []byte(CoinbaseFlag)...))

	return scriptSig
}

func getExcessiveBlockSizeSig() []byte {
	cbmsg := "/" + conf.GetSubVersionEB() + "/"
	return []byte(cbmsg)
}

func UpdateTime(bk *block.Block, indexPrev *blockindex.BlockIndex) int64 {
	oldTime := int64(bk.Header.Time)
	var newTime int64
	mt := indexPrev.GetMedianTimePast() + 1
	at := util.GetAdjustedTimeSec()
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

func TestBlockValidity(block *block.Block, indexPrev *blockindex.BlockIndex, checkHeader bool, checkMerlke bool) (err error) {
	if !(indexPrev != nil && indexPrev == chain.GetInstance().Tip()) {
		log.Error("TestBlockValidity(): indexPrev:%v, chain tip:%v.", indexPrev, chain.GetInstance().Tip())
		return errcode.NewError(errcode.RejectInvalid, "indexPrev is not the chain tip")
	}

	if err = lblockindex.CheckIndexAgainstCheckpoint(indexPrev); err != nil {
		log.Error("TestBlockValidity(): check index against check point failed, indexPrev:%v.", indexPrev)
		return err
	}

	coinMap := utxo.NewEmptyCoinsMap()
	coinMap.GetMap()
	blkHeader := block.GetBlockHeader()
	indexDummy := blockindex.NewBlockIndex(&blkHeader)
	indexDummy.Prev = indexPrev
	indexDummy.Height = indexPrev.Height + 1

	// NOTE: CheckBlockHeader is called by CheckBlock
	if err := lblock.ContextualCheckBlockHeader(&blkHeader, indexPrev, util.GetAdjustedTimeSec()); err != nil {
		log.Error("TestBlockValidity():ContextualCheckBlockHeader failed, blkHeader:%v, indexPrev:%v.", blkHeader, indexPrev)
		return err
	}

	if err := lblock.CheckBlock(block, checkHeader, checkMerlke); err != nil {
		log.Error("TestBlockValidity(): check block:%v error: %v,", block, err)
		return err
	}

	if err := lblock.ContextualCheckBlock(block, indexPrev); err != nil {
		log.Error("TestBlockValidity(): contextual check block:%v, indexPrev:%v error: %v", block, indexPrev, err)
		return err
	}

	if err := lchain.ConnectBlock(block, indexDummy, coinMap, true); err != nil {
		log.Error("trying to connect to the block:%v, indexDummy:%v, coinMap:%v, error:%v", block, indexDummy, coinMap, err)
		return err
	}

	return nil
}
