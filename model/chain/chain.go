package chain

import (
	"github.com/copernet/copernicus/model"
	"sort"
	"sync"
	"time"

	"errors"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/blockindex"

	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/versionbits"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/util"
	"gopkg.in/eapache/queue.v1"
)

// Chain An in-memory blIndexed chain of blocks.
type Chain struct {
	active      []*blockindex.BlockIndex
	branch      []*blockindex.BlockIndex
	waitForTx   map[util.Hash]*blockindex.BlockIndex
	orphan      map[util.Hash][]*blockindex.BlockIndex // preHash : *index
	indexMap    map[util.Hash]*blockindex.BlockIndex   // selfHash :*index
	newestBlock *blockindex.BlockIndex
	receiveID   uint64
	params      *model.BitcoinParams

	// The notifications field stores a slice of callbacks to be executed on
	// certain blockchain events.
	notificationsLock sync.RWMutex
	notifications     []NotificationCallback

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

func InitGlobalChain() {
	if globalChain == nil {
		globalChain = NewChain()
	}
	if len(conf.Cfg.Chain.AssumeValid) > 0 {
		hash, err := util.GetHashFromStr(conf.Cfg.Chain.AssumeValid)
		if err != nil {
			panic("AssumeValid config err")
		}
		HashAssumeValid = *hash
	} else {
		HashAssumeValid = model.ActiveNetParams.DefaultAssumeValid
	}
}

func NewChain() *Chain {
	c := &Chain{}
	c.params = model.ActiveNetParams
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

// Tip Returns the blIndex entry for the tip of this chain, or nullptr if none.
func (c *Chain) Tip() *blockindex.BlockIndex {
	if len(c.active) > 0 {
		return c.active[len(c.active)-1]
	}

	return nil
}

func (c *Chain) TipHeight() int32 {
	if len(c.active) > 0 {
		return c.active[len(c.active)-1].Height
	}

	return 0
}

// IsCurrent returns whether or not the chain believes it is current.  Several
// factors are used to guess, but the key factors that allow the chain to
// believe it is current are:
//  - Latest block height is after the latest checkpoint (if enabled)
//  - Latest block has a timestamp newer than 24 hours ago
func (c *Chain) IsCurrent() bool {
	// Not current if the latest main (best) chain height is before the
	// latest known good checkpoint (when checkpoints are enabled).
	//TODO: checkpoint
	//checkpoint := b.LatestCheckpoint()
	//if checkpoint != nil && b.bestChain.Tip().height < checkpoint.Height {
	//	return false
	//}

	// Not current if the latest best block has a timestamp before 24 hours ago.
	minus24Hours := time.Now().Add(-24 * time.Hour).Unix()
	tipTime := int64(c.Tip().GetBlockTime())

	return tipTime >= minus24Hours
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
	//if pindex.Height+1 >= param.CSVHeight {
	//	flags |= script.ScriptVerifyCheckSequenceVerify
	//}
	// Start enforcing BIP112 (CHECKSEQUENCEVERIFY) using versionbits logic.
	if versionbits.VersionBitsState(pindex, param, consensus.DeploymentCSV, versionbits.VBCache) == versionbits.ThresholdActive {
		flags |= script.ScriptVerifyCheckSequenceVerify
	}

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
	//The monolith HF enable a set of opcodes.
	if model.IsMonolithEnabled(pindex.GetMedianTimePast()) {
		flags |= script.ScriptEnableMonolithOpcodes
	}
	//if chainparams.IsMagneticAnomalyEnable(pindex.GetMedianTimePast()) {
	//	flags |= script.ScriptEnableCheckDataSig
	//	flags |= script.ScriptVerifySigPushOnly
	//	flags |= script.ScriptVerifyCleanStack
	//}
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
	chainLen := int32(len(c.active))
	if chainLen > 0 {
		return chainLen - 1
	}
	return 0
}

// SetTip Set/initialize a chain with a given tip.
func (c *Chain) SetTip(index *blockindex.BlockIndex) {
	if index == nil {
		c.active = []*blockindex.BlockIndex{}
		return
	}

	tmp := make([]*blockindex.BlockIndex, index.Height+1)
	copy(tmp, c.active)
	c.active = tmp
	for index != nil && c.active[index.Height] != index {
		c.active[index.Height] = index
		index = index.Prev
	}

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
			index.GetBlockHash().String(), index.Height)
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
		log.Trace("GetLocator contain hash : %s, height : %d .",
			index.GetBlockHash().String(), index.Height)
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
		jWork := c.branch[j].ChainWork
		return c.branch[i].ChainWork.Cmp(&jWork) == -1
	})
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
				q.Add(child)
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
	branchLen := len(c.branch)
	for i, bi := range c.branch {
		bh := bis.GetBlockHash()
		if bi.GetBlockHash().IsEqual(bh) {
			branch := c.branch[0:i]
			if branchLen-1 > i {
				c.branch = append(branch, c.branch[i+1:]...)
			} else {
				c.branch = branch
			}
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
		if pre.TimeMax > bi.TimeMax {
			bi.TimeMax = pre.TimeMax
		}
		bi.ChainWork = *bi.ChainWork.Add(&bi.ChainWork, &pre.ChainWork)
	}
	log.Debug("AddToIndexMap:%s index height:%d", hash.String(), bi.Height)
	bi.RaiseValidity(blockindex.BlockValidTree)
	gPersist := persist.GetInstance()
	gPersist.AddDirtyBlockIndex(bi)
	return nil
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

func (c *Chain) IndexMapSize() int {
	return len(c.indexMap)
}
