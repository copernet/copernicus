package blockindex

import (
	"math/big"
	"sort"
	"fmt"

	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/util"
)

/**
 * The block chain is a tree shaped structure starting with the genesis block at
 * the root, with each block potentially having multiple candidates to be the
 * next block. A blockIndex may have multiple prev pointing to it, but at most
 * one of them can be part of the currently active branch.
 */

 const (
 	statusHeaderValid uint32 = 1 << iota
 	statusAllValid
 	statusIndexStored
 	statusAllStored
	statusMissData
	statusAccepted

	//NOTE: This must be defined last in order to avoid influencing iota
	statusNone  = 0
 )

type BlockIndex struct {
	Header block.BlockHeader
	// pointer to the hash of the block, if any.
	BlockHash util.Hash
	// pointer to the index of the predecessor of this block
	Prev *BlockIndex
	// pointer to the index of some further predecessor of this block
	Skip *BlockIndex
	// height of the entry in the chain. The genesis block has height 0；
	Height int
	// Which # file this block is stored in (blk?????.dat)
	File int
	// Byte offset within blk?????.dat where this block's data is stored
	DataPos int
	// Byte offset within rev?????.dat where this block's undo data is stored
	UndoPos int
	// (memory only) Total amount of work (expected number of hashes) in the
	// chain up to and including this block
	ChainWork big.Int
	// Number of transactions in this block.
	// Note: in a potential headers-first mode, this number cannot be relied
	// upon
	TxCount int
	// (memory only) Number of transactions in the chain up to and including
	// this block.
	// This value will be non-zero only if and only if transactions for this
	// block and all its parents are available. Change to 64-bit type when
	// necessary; won't happen before 2030
	ChainTxCount int
	//status of this block. See enum
	Status uint32
	// (memory only) Sequential id assigned to distinguish order in which
	// blocks are received.
	SequenceID int32
	// (memory only) Maximum time in the chain upto and including this block.
	TimeMax uint32
}

const medianTimeSpan = 11

func (bIndex *BlockIndex) SetNull() {
	bIndex.Header.SetNull()
	bIndex.BlockHash = util.Hash{}
	bIndex.Prev = nil
	bIndex.Skip = nil

	bIndex.Height = 0
	bIndex.File = -1
	bIndex.DataPos = -1
	bIndex.UndoPos = -1
	bIndex.ChainWork = big.Int{}
	bIndex.ChainTxCount = 0
	bIndex.TxCount = 0
	bIndex.Status = 0
	bIndex.SequenceID = 0
	bIndex.TimeMax = 0
}

func (bIndex *BlockIndex) HaveData() bool {
	return bIndex.Status & statusMissData == 0
}

func (bIndex *BlockIndex) HeaderValid() bool {
	return bIndex.Status & statusHeaderValid != 0
}

func (bIndex *BlockIndex) AllValid() bool {
	return bIndex.Status & statusAllValid != 0
}

func (bIndex *BlockIndex) IndexStored() bool {
	return bIndex.Status & statusIndexStored != 0
}

func (bIndex *BlockIndex) AllStored() bool {
	return bIndex.Status & statusAllStored != 0
}

func (bIndex *BlockIndex) Accepted() bool {
	return bIndex.Status & statusAccepted != 0
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

	return &bIndex.BlockHash
}

func (bIndex *BlockIndex) GetBlockTime() uint32 {

	return bIndex.Header.Time
}

func (bIndex *BlockIndex) GetBlockTimeMax() uint32 {
	return bIndex.TimeMax
}

func (bIndex *BlockIndex) GetMedianTimePast() int64 {
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
	if bIndex.Prev != nil {
		bIndex.Skip = bIndex.Prev.GetAncestor(getSkipHeight(bIndex.Height))
	}
}

// Turn the lowest '1' bit in the binary representation of a number into a '0'.
func invertLowestOne(n int) int {
	return n & (n - 1)
}

// getSkipHeight Compute what height to jump back to with the CBlockIndex::pskip pointer.
func getSkipHeight(height int) int {
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

// GetAncestor efficiently find an ancestor of this block.
func (bIndex *BlockIndex) GetAncestor(height int) *BlockIndex {
	if height > bIndex.Height || height < 0 {
		return nil
	}
	indexWalk := bIndex
	heightWalk := bIndex.Height
	for heightWalk > height {
		heightSkip := getSkipHeight(heightWalk)
		heightSkipPrev := getSkipHeight(heightWalk - 1)
		if indexWalk.Skip != nil && (heightSkip == height ||
			(heightSkip > height && !(heightSkipPrev < heightSkip-2 && heightSkipPrev >= height))) {
			// Only follow skip if prev->skip isn't better than skip->prev.
			indexWalk = indexWalk.Skip
			heightWalk = heightSkip
		} else {
			if indexWalk.Prev == nil {
				panic("The blockIndex pointer should not be nil")
			}
			indexWalk = indexWalk.Prev
			heightWalk--
		}
	}

	return indexWalk
}

func (bIndex *BlockIndex) ToString() string {
	hash := bIndex.GetBlockHash()
	return fmt.Sprintf("BlockIndex(pprev=%p, height=%d, merkle=%s, hashBlock=%s)\n", bIndex.Prev,
		bIndex.Height, bIndex.Header.MerkleRoot.ToString(), hash.ToString())
}

func NewBlockIndex(blkHeader *block.BlockHeader) *BlockIndex {
	blockIndex := new(BlockIndex)
	blockIndex.SetNull()
	blockIndex.Header = *blkHeader

	return blockIndex
}


