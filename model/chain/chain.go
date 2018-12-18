package chain

import (
	"errors"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/disk"
	"math/big"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/util"
	"gopkg.in/eapache/queue.v1"
	"gopkg.in/fatih/set.v0"
)

// Chain An in-memory blIndexed chain of blocks.
type Chain struct {
	active    []*blockindex.BlockIndex
	branch    []*blockindex.BlockIndex
	waitForTx map[util.Hash]*blockindex.BlockIndex
	orphan    map[util.Hash][]*blockindex.BlockIndex // preHash : *index
	indexMap  map[util.Hash]*blockindex.BlockIndex   // selfHash :*index
	receiveID uint64
	params    *model.BitcoinParams

	// The notifications field stores a slice of callbacks to be executed on
	// certain blockchain events.
	notificationsLock sync.RWMutex
	notifications     []NotificationCallback

	// the current most chainwork index of blockheader
	pindexBestHeader     *blockindex.BlockIndex
	pindexBestHeaderLock sync.RWMutex

	tip atomic.Value
	*SyncingState
}

var globalChain *Chain
var HashAssumeValid util.Hash

func GetInstance() *Chain {
	if globalChain == nil {
		panic("globalChain do not init")
	}
	return globalChain
}

func InitGlobalChain(btd *blkdb.BlockTreeDB) bool {
	if globalChain == nil {
		globalChain = NewChain()
	}
	if len(conf.Cfg.Chain.AssumeValid) > 0 {
		assumeValid := strings.TrimPrefix(conf.Cfg.Chain.AssumeValid, "0x")
		hash, err := util.GetHashFromStr(assumeValid)
		if err != nil {
			panic("AssumeValid config err")
		}
		HashAssumeValid = *hash
	} else {
		HashAssumeValid = model.ActiveNetParams.DefaultAssumeValid
	}
	if !globalChain.loadBlockIndex(btd) {
		return false
	}

	return true
}

func (c *Chain) loadBlockIndex(btd *blkdb.BlockTreeDB) bool {
	globalBlockIndexMap := make(map[util.Hash]*blockindex.BlockIndex)
	branch := make([]*blockindex.BlockIndex, 0, 20)

	// Load blockindex from DB
	if !btd.LoadBlockIndexGuts(globalBlockIndexMap, c.GetParams()) {
		return false
	}
	sortedByHeight := make([]*blockindex.BlockIndex, 0, len(globalBlockIndexMap))
	for _, index := range globalBlockIndexMap {
		sortedByHeight = append(sortedByHeight, index)
	}
	//sort by decrease
	sort.SliceStable(sortedByHeight, func(i, j int) bool {
		return sortedByHeight[i].Height < sortedByHeight[j].Height
	})
	for _, index := range sortedByHeight {
		timeMax := index.Header.Time
		if index.Prev != nil {
			sum := big.NewInt(0)
			sum.Add(&index.Prev.ChainWork, pow.GetBlockProof(index))
			index.ChainWork = *sum
			if index.Header.Time < index.Prev.Header.Time {
				timeMax = index.Prev.Header.Time
			}
		} else {
			index.ChainWork = *pow.GetBlockProof(index)
		}
		index.TimeMax = timeMax
		// We can link the chain of blocks for which we've received transactions
		// at some point. Pruned nodes may have deleted the block.
		if index.TxCount < 0 {
			log.Error("index's Txcount is < 0 ")
			panic("index's Txcount is < 0 ")
		}
		if index.Prev != nil {
			if index.Prev.ChainTxCount != 0 && index.TxCount != 0 {
				index.ChainTxCount = index.Prev.ChainTxCount + index.TxCount
				branch = append(branch, index)
			} else {
				index.ChainTxCount = 0
				c.AddToOrphan(index)
			}
		} else {
			// genesis block
			index.ChainTxCount = index.TxCount
			branch = append(branch, index)
		}

		if index.Prev != nil {
			index.BuildSkip()
		}

		if index.IsValid(blockindex.BlockValidTree) {
			idxBestHeader := c.GetIndexBestHeader()
			if idxBestHeader == nil ||
				idxBestHeader.ChainWork.Cmp(&index.ChainWork) == -1 {
				c.SetIndexBestHeader(index)
			}
		}
	}
	log.Debug("loadBlockIndex, BlockIndexMap len:%d, Branch len:%d, Orphan len:%d",
		len(globalBlockIndexMap), len(branch), c.ChainOrphanLen())

	// Check presence of block index files
	setBlkDataFiles := set.New()
	log.Debug("Checking all blk files are present...")
	for _, item := range globalBlockIndexMap {
		index := item
		if index.HasData() {
			setBlkDataFiles.Add(index.File)
		}
	}
	l := setBlkDataFiles.List()
	for _, item := range l {
		pos := &block.DiskBlockPos{
			File: item.(int32),
			Pos:  0,
		}
		file := disk.OpenBlockFile(pos, true)
		if file == nil {
			log.Debug("Check block file: %d error, please delete blocks and chainstate and run again", pos.File)
			panic("loadBlockIndex: check block file err")
		}
		file.Close()
	}

	// Build chain's active
	c.InitLoad(globalBlockIndexMap, branch)
	bestHash, err := utxo.GetUtxoCacheInstance().GetBestBlock()
	log.Debug("find bestblock hash:%s and err:%v from utxo", bestHash, err)
	if err == nil {
		tip, ok := globalBlockIndexMap[bestHash]
		if !ok {
			//shoud reindex from db
			log.Debug("can't find beskblock from blockindex db, please delete blocks and chainstate and run again")
			panic("can't find tip from blockindex db")
		}
		// init active chain by tip[load from db]
		c.SetTip(tip)
		log.Debug("loadBlockIndex(): hashBestChain=%s height=%d date=%s, tiphash:%s\n",
			c.Tip().GetBlockHash(), c.Height(),
			time.Unix(int64(c.Tip().GetBlockTime()), 0).Format("2006-01-02 15:04:05"),
			c.Tip().GetBlockHash())
	}

	return true
}

// Close FIXME: this is only for test. We must do it in a graceful way
func Close() {
	globalChain = nil
}

func NewChain() *Chain {
	c := &Chain{}
	c.params = model.ActiveNetParams
	c.orphan = make(map[util.Hash][]*blockindex.BlockIndex)
	c.SyncingState = &SyncingState{}
	return c
}

func (c *Chain) GetParams() *model.BitcoinParams {
	return c.params
}

//InitLoad load the maps of the chain
func (c *Chain) InitLoad(indexMap map[util.Hash]*blockindex.BlockIndex, branch []*blockindex.BlockIndex) {
	c.indexMap = indexMap
	c.branch = branch
}

// Genesis Returns the blIndex entry for the genesis block of this chain,
// or nullptr if none.
func (c *Chain) Genesis() *blockindex.BlockIndex {
	if len(c.active) > 0 {
		return c.active[0]
	}

	return nil
}

func (c *Chain) GetReceivedID() uint64 {
	c.receiveID++
	return c.receiveID - 1
}

// FindHashInActive finds blockindex from active
func (c *Chain) FindHashInActive(hash util.Hash) *blockindex.BlockIndex {
	bi, ok := c.indexMap[hash]
	if ok {
		if c.Contains(bi) {
			return bi
		}
		return nil
	}

	return nil
}

// FindBlockIndex finds blockindex from blockIndexMap
func (c *Chain) FindBlockIndex(hash util.Hash) *blockindex.BlockIndex {
	bi, ok := c.indexMap[hash]
	if ok {
		//log.Trace("current chain Tip header height : %d", bi.Height)
		return bi
	}

	return nil
}

// GetChainTips Returns fork tip, no activate chain tip
func (c *Chain) GetChainTips() *set.Set {
	setTips := set.New() // element type:
	setOrphans := set.New()
	setPrevs := set.New()

	gchain := GetInstance()
	setTips.Add(gchain.Tip())
	for _, index := range gchain.indexMap {
		if !gchain.Contains(index) {
			setOrphans.Add(index)
			setPrevs.Add(index.Prev)
		}
	}

	setOrphans.Each(func(item interface{}) bool {
		bindex := item.(*blockindex.BlockIndex)
		if !setPrevs.Has(bindex) {
			setTips.Add(bindex)
		}
		return true
	})

	return setTips
}

// Tip Returns the blIndex entry for the tip of this chain, or nullptr if none.
func (c *Chain) Tip() *blockindex.BlockIndex {
	if t := c.tip.Load(); t != nil {
		return t.(*blockindex.BlockIndex)
	}

	return nil
}

func (c *Chain) TipHeight() int32 {
	if t := c.tip.Load(); t != nil {
		return t.(*blockindex.BlockIndex).Height
	}

	return 0
}

func (c *Chain) GetSpendHeight(hash *util.Hash) int32 {
	index, ok := c.indexMap[*hash]
	if ok {
		return index.Height + 1
	}

	return -1
}

func (c *Chain) GetBlockScriptFlags(pindex *blockindex.BlockIndex) uint32 {
	// TODO: AssertLockHeld(cs_main);
	// var sc sync.RWMutex
	// sc.Lock()
	// defer sc.Unlock()

	// BIP16 didn't become active until Apr 1 2012
	/** Activation time for P2SH (April 1st 2012) */
	var flags uint32
	P2SHActivationTime := 1333234914
	//nBIP16SwitchTime := 1333238400
	if pindex.GetMedianTimePast() >= int64(P2SHActivationTime) {
		flags = script.ScriptVerifyP2SH
	} else {
		flags = script.ScriptVerifyNone
	}
	//fStrictPayToScriptHash := int(pindex.GetBlockTime()) >= nBIP16SwitchTime
	param := c.params
	// Start enforcing the DERSIG (BIP66) rule

	if pindex.Height+1 >= param.BIP66Height {
		flags |= script.ScriptVerifyDersig
	}

	// Start enforcing CHECKLOCKTIMEVERIFY (BIP65) rule
	if pindex.Height+1 >= param.BIP65Height {
		flags |= script.ScriptVerifyCheckLockTimeVerify
	}

	// Start enforcing CSV (BIP68, BIP112 and BIP113) rule.
	if pindex.Height+1 >= param.CSVHeight {
		flags |= script.ScriptVerifyCheckSequenceVerify
	}
	//// Start enforcing BIP112 (CHECKSEQUENCEVERIFY) using versionbits logic.
	//if versionbits.VersionBitsState(pindex, param, consensus.DeploymentCSV, versionbits.VBCache) == versionbits.ThresholdActive {
	//	flags |= script.ScriptVerifyCheckSequenceVerify
	//}

	// If the UAHF is enabled, we start accepting replay protected txns
	if model.IsUAHFEnabled(pindex.Height) {
		flags |= script.ScriptVerifyStrictEnc
		flags |= script.ScriptEnableSigHashForkID
	}

	// If the Cash HF is enabled, we start rejecting transaction that use a high
	// s in their signature. We also make sure that signature that are supposed
	// to fail (for instance in multisig or other forms of smart contracts) are
	// null.
	//if pindex.IsCashHFEnabled(param) {
	//	flags |= script.ScriptVerifyLowS
	//	flags |= script.ScriptVerifyNullFail
	//}
	if model.IsDAAEnabled(pindex.Height) {
		flags |= script.ScriptVerifyLowS
		flags |= script.ScriptVerifyNullFail
	}

	// When the magnetic anomaly fork is enabled, we start accepting
	// transactions using the OP_CHECKDATASIG opcode and it's verify
	// alternative. We also start enforcing push only signatures and
	// clean stack.
	if model.IsMagneticAnomalyEnabled(pindex.GetMedianTimePast()) {
		flags |= script.ScriptEnableCheckDataSig
		flags |= script.ScriptVerifySigPushOnly
		flags |= script.ScriptVerifyCleanStack
	}

	// We make sure this node will have replay protection during the next hard
	// fork.
	if model.IsReplayProtectionEnabled(pindex.GetMedianTimePast()) {
		flags |= script.ScriptEnableReplayProtection
	}

	return flags
}

// GetIndex Returns the blIndex entry at a particular height in this chain, or nullptr
// if no such height exists.
func (c *Chain) GetIndex(height int32) *blockindex.BlockIndex {
	if height < 0 || height >= int32(len(c.active)) {
		return nil
	}

	return c.active[height]
}

// Equal Compare two chains efficiently.

func (c *Chain) Equal(dst *Chain) bool {
	if dst == nil {
		return false
	}
	return len(c.active) == len(dst.active) &&
		c.active[len(c.active)-1] == dst.active[len(dst.active)-1]
}

// Contains /** Efficiently check whether a block is present in this chain
func (c *Chain) Contains(index *blockindex.BlockIndex) bool {
	if index == nil {
		return false
	}
	return c.GetIndex(index.Height) == index
}

// Next Find the successor of a block in this chain, or nullptr if the given
// index is not found or is the tip.
func (c *Chain) Next(index *blockindex.BlockIndex) *blockindex.BlockIndex {
	if index == nil {
		return nil
	}
	if c.Contains(index) {
		return c.GetIndex(index.Height + 1)
	}
	return nil
}

// Height Return the maximal height in the chain. Is equal to chain.Tip() ?
// chain.Tip()->nHeight : -1.
func (c *Chain) Height() int32 {
	if t := c.Tip(); t != nil {
		return t.Height
	}

	return -1
}

// SetTip Set/initialize a chain with a given tip.
func (c *Chain) SetTip(index *blockindex.BlockIndex) {
	if index == nil {
		c.active = []*blockindex.BlockIndex{}
		c.tip = atomic.Value{}
		return
	}

	newTip := index

	tmp := make([]*blockindex.BlockIndex, index.Height+1)
	copy(tmp, c.active)
	for index != nil && tmp[index.Height] != index {
		tmp[index.Height] = index
		index = index.Prev
	}

	c.active = tmp
	c.tip.Store(newTip)

	c.UpdateSyncingState()
}

// SetTip Set/initialize a chain with a given tip.
// func (c *Chain) SetTip1(index *blockindex.BlockIndex) {
// 	if index == nil {
// 		c.active = []*blockindex.BlockIndex{}
// 		return
// 	}
// 	endHeight := index.Height
// 	tmp := make([]*blockindex.BlockIndex, 0)
// 	if int(index.Height) > len(c.active){
//
// 	}
// 	forkHeight := index.Height
// 	for index != nil {
// 		if c.Contains(index){
// 			forkHeight = index.Height
// 			break
// 		}
// 		tmp = append(tmp, index)
// 		index = index.Prev
// 	}
// 	// todo update active
// }

// GetAncestor gets ancestor from active chain.
func (c *Chain) GetAncestor(height int32) *blockindex.BlockIndex {
	if height < 0 {
		return nil
	}
	if len(c.active) > int(height) {
		return c.active[height]
	}
	return nil
}

// GetLocator get a series blockHash, which slice contain blocks sort
// by height from Highest to lowest.
func (c *Chain) GetLocator(index *blockindex.BlockIndex) *BlockLocator {
	step := 1
	blockHashList := make([]util.Hash, 0, 32)
	if index == nil {
		index = c.Tip()
		log.Trace("GetLocator Tip hash: %s,  height: %d",
			index.GetBlockHash(), index.Height)
	}
	for {
		blockHashList = append(blockHashList, *index.GetBlockHash())
		if index.Height == 0 {
			break
		}
		height := index.Height - int32(step)
		if height < 0 {
			height = 0
		}
		if c.Contains(index) {
			index = c.GetIndex(height)
		} else {
			index = index.GetAncestor(height)
		}

		if len(blockHashList) > 10 {
			step *= 2
		}
	}
	return NewBlockLocator(blockHashList)
}

// FindFork Find the last common block between this chain and a block blIndex entry.
func (c *Chain) FindFork(blIndex *blockindex.BlockIndex) *blockindex.BlockIndex {
	if blIndex == nil {
		return nil
	}

	if blIndex.Height > c.Height() {
		blIndex = blIndex.GetAncestor(c.Height())
	}

	for blIndex != nil && !c.Contains(blIndex) {
		blIndex = blIndex.Prev
	}
	return blIndex
}

// ParentInBranch finds blockindex'parent in branch
func (c *Chain) ParentInBranch(pindex *blockindex.BlockIndex) bool {
	if pindex == nil {
		return false
	}
	for _, bi := range c.branch {
		bh := pindex.Header
		if bi.GetBlockHash().IsEqual(&bh.HashPrevBlock) {
			return true
		}
	}
	return false
}

// InBranch finds blockindex in branch
func (c *Chain) InBranch(pindex *blockindex.BlockIndex) bool {
	if pindex == nil {
		return false
	}
	for _, bi := range c.branch {
		bh := pindex.GetBlockHash()
		if bi.GetBlockHash().IsEqual(bh) {
			return true
		}
	}
	return false
}

// blocks in the branch ranks in the order of 'proof of work'
func (c *Chain) insertToBranch(bis *blockindex.BlockIndex) {
	c.branch = append(c.branch, bis)
	sort.SliceStable(c.branch, func(i, j int) bool {
		// First sort by most total work, ...
		jWork := c.branch[j].ChainWork
		//return c.branch[i].ChainWork.Cmp(&jWork) == -1
		if c.branch[i].ChainWork.Cmp(&jWork) == 1 {
			return false
		}
		if c.branch[i].ChainWork.Cmp(&jWork) == -1 {
			return true
		}

		// ... then by earliest time received, ...
		//if c.branch[i].SequenceID < c.branch[j].SequenceID {
		//	return false
		//}
		//if c.branch[i].SequenceID > c.branch[j].SequenceID {
		//	return true
		//}

		return false
	})
}

func (c *Chain) ResetBlockFailureFlags(targetBI *blockindex.BlockIndex) {
	for _, bi := range c.indexMap {
		if bi.IsInvalid() && bi.GetAncestor(targetBI.Height) == targetBI {
			bi.SubStatus(blockindex.BlockFailedParent)
			persist.GetInstance().AddDirtyBlockIndex(bi)

			if bi.IsValid(blockindex.BlockValidTransactions) && bi.ChainTxCount > 0 {
				if !c.InBranch(bi) {
					c.insertToBranch(bi)
				}
			}
		}
	}
}

func (c *Chain) AddToBranch(bis *blockindex.BlockIndex) error {
	if bis == nil {
		return errors.New("nil blockIndex")
	}
	if bis.Prev == nil && !bis.IsGenesis(c.params) {
		return errors.New("no blockIndexPrev")
	}

	q := queue.New()
	q.Add(bis)
	// Recursively process any descendant blocks that now may be eligible to
	// be connected.
	for q.Length() > 0 {
		qindex := q.Remove()
		pindex := qindex.(*blockindex.BlockIndex)
		if !pindex.IsGenesis(c.params) {
			pindex.ChainTxCount = pindex.Prev.ChainTxCount + pindex.TxCount
		} else {
			pindex.ChainTxCount = pindex.TxCount
		}
		pindex.SequenceID = c.GetReceivedID()
		if !c.InBranch(pindex) {
			c.insertToBranch(pindex)
		}
		preHash := pindex.GetBlockHash()
		childList, ok := c.orphan[*preHash]
		if ok {
			for _, child := range childList {
				if child.HasData() {
					q.Add(child)
				}
			}

			delete(c.orphan, *preHash)
		}
	}
	return nil
}

func (c *Chain) RemoveFromBranch(bis *blockindex.BlockIndex) error {
	if bis == nil {
		return errors.New("nil blockIndex")
	}
	bh := bis.GetBlockHash()
	for i, bi := range c.branch {
		if bi.GetBlockHash().IsEqual(bh) {
			c.branch = append(c.branch[0:i], c.branch[i+1:]...)
			return nil
		}
	}
	return nil
}

func (c *Chain) FindMostWorkChain() *blockindex.BlockIndex {
	if len(c.branch) > 0 {
		return c.branch[len(c.branch)-1]
	}
	return nil
}

func (c *Chain) AddToIndexMap(bi *blockindex.BlockIndex) error {
	// We assign the sequence id to blocks only when the full data is available,
	// to avoid miners withholding blocks but broadcasting headers, to get a
	// competitive advantage.
	if bi == nil {
		return errors.New("nil blockIndex")
	}
	bi.SequenceID = 0
	bi.TimeMax = bi.Header.Time
	blockProof := pow.GetBlockProof(bi)
	bi.ChainWork = *blockProof
	hash := bi.GetBlockHash()

	c.indexMap[*hash] = bi

	pre, ok := c.indexMap[bi.Header.HashPrevBlock]
	if ok {
		bi.Prev = pre
		bi.Height = pre.Height + 1
	}
	if pre != nil {
		bi.BuildSkip()
		if pre.TimeMax > bi.TimeMax {
			bi.TimeMax = pre.TimeMax
		}
		bi.ChainWork = *bi.ChainWork.Add(&bi.ChainWork, &pre.ChainWork)
	}

	bi.RaiseValidity(blockindex.BlockValidTree)

	c.tryUpdateIndexBestHeader(bi)

	gPersist := persist.GetInstance()
	gPersist.AddDirtyBlockIndex(bi)
	return nil
}

func (c *Chain) tryUpdateIndexBestHeader(bi *blockindex.BlockIndex) {
	idxBestHeader := c.GetIndexBestHeader()
	if idxBestHeader == nil ||
		idxBestHeader.ChainWork.Cmp(&bi.ChainWork) == -1 {
		c.SetIndexBestHeader(bi)
	}
}

func (c *Chain) AddToOrphan(bi *blockindex.BlockIndex) error {
	bh := bi.Header
	childList, ok := c.orphan[bh.HashPrevBlock]
	if !ok {
		childList = make([]*blockindex.BlockIndex, 0, 1)
	}
	childList = append(childList, bi)
	c.orphan[bh.HashPrevBlock] = childList
	return nil
}

func (c *Chain) ChainOrphanLen() int32 {
	var orphanLen int32
	for _, childList := range c.orphan {
		orphanLen += int32(len(childList))
	}

	return orphanLen
}

func (c *Chain) ClearActive() {
	c.active = make([]*blockindex.BlockIndex, 100)
	c.tip = atomic.Value{}
}

func (c *Chain) IndexMapSize() int {
	return len(c.indexMap)
}

//BuildForwardTree Build forward-pointing map of the entire block tree.
func (c *Chain) BuildForwardTree() (forward map[*blockindex.BlockIndex][]*blockindex.BlockIndex) {
	forward = make(map[*blockindex.BlockIndex][]*blockindex.BlockIndex)
	for _, v := range c.indexMap {
		if len(forward[v.Prev]) == 0 {
			forward[v.Prev] = []*blockindex.BlockIndex{v}
		} else {
			forward[v.Prev] = append(forward[v.Prev], v)
		}
	}
	return
}

func (c *Chain) CanDirectFetch() bool {
	return int64(c.Tip().GetBlockTime()) > util.GetAdjustedTimeSec()-int64(c.params.TargetTimePerBlock)*20
}

func (c *Chain) GetIndexBestHeader() *blockindex.BlockIndex {
	c.pindexBestHeaderLock.RLock()
	idx := c.pindexBestHeader
	c.pindexBestHeaderLock.RUnlock()
	return idx
}

func (c *Chain) SetIndexBestHeader(idx *blockindex.BlockIndex) {
	c.pindexBestHeaderLock.Lock()
	c.pindexBestHeader = idx
	c.pindexBestHeaderLock.Unlock()
}

func (c *Chain) CheckIndexAgainstCheckpoint(preIndex *blockindex.BlockIndex) (err error) {
	if preIndex.IsGenesis(c.GetParams()) {
		return nil
	}

	nHeight := preIndex.Height + 1
	// Don't accept any forks from the main chain prior to last checkpoint
	params := c.GetParams()
	checkPoints := params.Checkpoints
	var checkPoint *model.Checkpoint
	for i := len(checkPoints) - 1; i >= 0; i-- {
		checkPointIndex := c.FindBlockIndex(*checkPoints[i].Hash)
		if checkPointIndex != nil {
			checkPoint = checkPoints[i]
			break
		}
	}

	if checkPoint != nil && nHeight < checkPoint.Height {
		return errcode.NewError(errcode.RejectCheckpoint, "bad-fork-prior-to-checkpoint")
	}

	return nil
}
