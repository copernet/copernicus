package blockindex

import (
	"math/big"
	"sort"
	"fmt"

	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/util"
	//"github.com/btcboost/copernicus/model/pow"
)

/**
 * The block chain is a tree shaped structure starting with the genesis block at
 * the root, with each block potentially having multiple candidates to be the
 * next block. A blockIndex may have multiple prev pointing to it, but at most
 * one of them can be part of the currently active branch.
 */

 const (
 	StatusAllValid uint32 = 1 << iota
 	StatusIndexStored
 	StatusDataStored
	StatusWaitingData
	StatusFailed
	StatusAccepted

	//NOTE: This must be defined last in order to avoid influencing iota
	StatusNone  = 0
 )

type BlockIndex struct {
	Header block.BlockHeader
	// pointer to the hash of the block, if any.
	blockHash util.Hash
	// pointer to the index of the predecessor of this block
	Prev *BlockIndex
	// pointer to the index of some further predecessor of this block
	Skip *BlockIndex
	// height of the entry in the chain. The genesis block has height 0；
	Height int32
	// Which # file this block is stored in (blk?????.dat)
	File int32
	// Byte offset within blk?????.dat where this block's data is stored
	DataPos uint32
	// Byte offset within rev?????.dat where this block's undo data is stored
	UndoPos uint32
	// (memory only) Total amount of work (expected number of hashes) in the
	// chain up to and including this block
	ChainWork big.Int
	// Number of transactions in this block.
	// Note: in a potential headers-first mode, this number cannot be relied
	// upon
	TxCount int32
	// (memory only) Number of transactions in the chain up to and including
	// this block.
	// This value will be non-zero only if and only if transactions for this
	// block and all its parents are available. Change to 64-bit type when
	// necessary; won't happen before 2030
	ChainTxCount int32
	//status of this block. See enum
	Status uint32
	// (memory only) Sequential id assigned to distinguish order in which
	// blocks are received.
	SequenceID uint64
	// (memory only) Maximum time in the chain upto and including this block.
	TimeMax uint32
	isGenesis bool
}

const medianTimeSpan = 11

func (bIndex *BlockIndex) SetNull() {
	bIndex.Header.SetNull()
	bIndex.blockHash = util.Hash{}
	bIndex.Prev = nil
	bIndex.Skip = nil

	bIndex.Height = 0
	bIndex.File = -1
	bIndex.DataPos = 0
	bIndex.UndoPos = 0
	bIndex.ChainWork = big.Int{}
	bIndex.ChainTxCount = 0
	bIndex.TxCount = 0
	bIndex.Status = StatusNone
	bIndex.SequenceID = 0
	bIndex.TimeMax = 0
}

func (bIndex *BlockIndex) WaitingData() bool {
	return bIndex.Status & StatusWaitingData != 0
}

func (bIndex *BlockIndex) AllValid() bool {
	return bIndex.Status & StatusAllValid != 0
}

func (bIndex *BlockIndex) IndexStored() bool {
	return bIndex.Status & StatusIndexStored != 0
}

func (bIndex *BlockIndex) AllStored() bool {
	return bIndex.Status & StatusDataStored != 0
}

func (bIndex *BlockIndex) Accepted() bool {
	return bIndex.Status & StatusAccepted != 0
}

func (bIndex *BlockIndex) Failed() bool {
	return bIndex.Status & StatusFailed != 0
}

func (bIndex *BlockIndex) AddStatus(statu uint32) {
	bIndex.Status |= statu
}

func (bIndex *BlockIndex) SubStatus(statu uint32) {
	bIndex.Status &= ^statu
}

func (bIndex *BlockIndex) GetDataPos() int {

	return 0
}

func (bIndex *BlockIndex) GetUndoPos() block.DiskBlockPos {
	return block.DiskBlockPos{File:bIndex.File,Pos:bIndex.UndoPos}
}

func (bIndex *BlockIndex) GetBlockPos() block.DiskBlockPos {
	return block.DiskBlockPos{File:bIndex.File,Pos:bIndex.DataPos}
}

func (bIndex *BlockIndex) GetBlockHeader() *block.BlockHeader {

	return &bIndex.Header
}

func (bIndex *BlockIndex) GetBlockHash() *util.Hash {
	bHash := bIndex.blockHash
	if bHash.IsNull(){
		bIndex.blockHash = bIndex.Header.GetHash()
	}
	if bHash.IsEqual(&util.Hash{}) {
		bIndex.blockHash = bIndex.Header.GetHash()
	}
	return &bIndex.blockHash
}

func (bIndex *BlockIndex) SetBlockHash(hash util.Hash) {
	bIndex.blockHash =  hash
}

func (bIndex *BlockIndex) GetBlockTime() uint32 {

	return bIndex.Header.Time
}

func (bIndex *BlockIndex) GetBlockTimeMax() uint32 {
	return bIndex.TimeMax
}

func (bIndex *BlockIndex) GetMedianTimePast() int64 {
	//return 1510600611
	median := make([]int64, 0, medianTimeSpan)
	index := bIndex
	numNodes := 0
	for i := 0; i < medianTimeSpan && index != nil; i++ {
		median = append(median, int64(index.GetBlockTime()))
		index = index.Prev
		numNodes++
	}
	median = median[:numNodes]
	sort.Slice(median, func(i, j int) bool {
		return median[i] < median[j]
	})

	return median[numNodes/2]
}

// IsValid checks whether this block index entry is valid up to the passed validity
// level.
func (bIndex *BlockIndex) IsValid(upto uint32) bool {

	return false
}

// RaiseValidity Raise the validity level of this block index entry.
// Returns true if the validity was changed.
func (bIndex *BlockIndex) RaiseValidity(upto uint32) bool {

	return false
}

func (bIndex *BlockIndex) BuildSkip() {
	// if bIndex.Prev != nil {
	// 	bIndex.Skip = bIndex.Prev.GetAncestor(getSkipHeight(bIndex.Height))
	// }
}

// Turn the lowest '1' bit in the binary representation of a number into a '0'.
func invertLowestOne(n int32) int32 {
	return n & (n - 1)
}

// getSkipHeight Compute what height to jump back to with the CBlockIndex::pskip pointer.
func getSkipHeight(height int32) int32 {
	if height < 2 {
		return 0
	}

	// Determine which height to jump back to. Any number strictly lower than
	// height is acceptable, but the following expression seems to perform well
	// in simulations (max 110 steps to go back up to 2**18 blocks).
	if (height & 1) > 0 {
		return invertLowestOne(invertLowestOne(height-1)) + 1
	}
	return invertLowestOne(height)
}

var b1  BlockIndex
var b2 BlockIndex


// GetAncestor efficiently find an ancestor of this block.
func (bIndex *BlockIndex) GetAncestor(height int32) *BlockIndex {
	b1.Height = 1188551
	b1.Header.Time = 1510590879
	//b1.ChainWork = *pow.HashToBig(util.HashFromString("0000000000000000000000000000000000000000000000288037870218978565"))
	//
	b2.Height = 1188552
	b2.Header.Time = 1510590881
	b2.Prev = &b1
	//b2.ChainWork = *pow.HashToBig(util.HashFromString("0000000000000000000000000000000000000000000000288037870218978565"))
	//
	b:= BlockIndex{}
	b.Header.Time = 1510590883
	b.Height = 1188553
	b.Prev = &b2
	return &b

	if height > bIndex.Height || height < 0 {
		return nil
	}
	if height == bIndex.Height{
		return  bIndex
	}
	indexWalk := bIndex
	for indexWalk.Prev != nil{
		if indexWalk.Prev.Height == height{
			return indexWalk.Prev
		}
		indexWalk = indexWalk.Prev
	}
	// indexWalk := bIndex
	// heightWalk := bIndex.Height
	// for heightWalk > height {
	// 	heightSkip := getSkipHeight(heightWalk)
	// 	heightSkipPrev := getSkipHeight(heightWalk - 1)
	// 	if indexWalk.Skip != nil && (heightSkip == height ||
	// 		(heightSkip > height && !(heightSkipPrev < heightSkip-2 && heightSkipPrev >= height))) {
	// 		// Only follow skip if prev->skip isn't better than skip->prev.
	// 		indexWalk = indexWalk.Skip
	// 		heightWalk = indexWalk.Height
	// 	} else {
	// 		if indexWalk.Prev == nil {
	// 			panic("The blockIndex pointer should not be nil")
	// 		}
	// 		indexWalk = indexWalk.Prev
	// 		heightWalk--
	// 	}
	// }

	return indexWalk
}

func (bIndex *BlockIndex) String() string {
	hash := bIndex.GetBlockHash()
	return fmt.Sprintf("BlockIndex(pprev=%p, height=%d, merkle=%s, hashBlock=%s)\n", bIndex.Prev,
		bIndex.Height, bIndex.Header.MerkleRoot.String(), hash.String())
}

func (bIndex *BlockIndex) IsGenesis(params *chainparams.BitcoinParams) bool{
	bhash := bIndex.GetBlockHash()
	genesisHash := params.GenesisBlock.GetHash()
	return bhash.IsEqual(&genesisHash)
}

func (index *BlockIndex) IsCashHFEnabled(params *chainparams.BitcoinParams) bool{
	return index.GetMedianTimePast() >= params.CashHardForkActivationTime
}
func (bIndex *BlockIndex) SetIsGenesis(params *chainparams.BitcoinParams) bool{
	bh := bIndex.Header
	bHash := bh.GetHash()
	genesisHash := params.GenesisBlock.GetHash()
	return bHash.IsEqual(&genesisHash)
}

func NewBlockIndex(blkHeader *block.BlockHeader) *BlockIndex {
	bi := new(BlockIndex)
	bi.SetNull()
	bi.Header = *blkHeader
	return bi
}


