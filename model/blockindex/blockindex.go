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
	// Verification status of this block. See enum BlockStatus
	Status uint32
	// (memory only) Sequential id assigned to distinguish order in which
	// blocks are received.
	SequenceID int32
	// (memory only) Maximum time in the chain upto and including this block.
	TimeMax uint32
}

const medianTimeSpan = 11

func (blIndex *BlockIndex) SetNull() {
	blIndex.Header.SetNull()
	blIndex.BlockHash = utils.Hash{}
	blIndex.Prev = nil
	blIndex.Skip = nil

	blIndex.Height = 0
	blIndex.File = 0
	blIndex.DataPos = 0
	blIndex.UndoPos = 0
	blIndex.ChainWork = big.Int{}
	blIndex.ChainTxCount = 0
	blIndex.TxCount = 0
	blIndex.Status = 0
	blIndex.SequenceID = 0
	blIndex.TimeMax = 0
}

func (blIndex *BlockIndex) DelBlockIndex() bool {
	_, exist := BlockIndexMap[blIndex.BlockHash]
	if exist {
		delete(BlockIndexMap, blIndex.BlockHash)
		return true
	}
	return false
}

func (blIndex *BlockIndex) ReadBlockIndex(hash *utils.Hash) *BlockIndex {
	// The caller should justify return value by comparing to nil
	// to be sure for validation
	return BlockIndexMap[blIndex.BlockHash]
}

func (blIndex *BlockIndex) GetBlockPos() DiskBlockPos {
	var ret DiskBlockPos
	if (blIndex.Status & BlockHaveData) != 0 {
		ret.File = blIndex.File
		ret.Pos = blIndex.DataPos
	}

	return ret
}

func (blIndex *BlockIndex) GetUndoPos() DiskBlockPos {
	var ret DiskBlockPos
	if (blIndex.Status & BlockHaveUndo) != 0 {
		ret.File = blIndex.File
		ret.Pos = blIndex.UndoPos
	}

	return ret
}

func (blIndex *BlockIndex) GetBlockHeader() *BlockHeader {
	return &blIndex.Header
}

func (blIndex *BlockIndex) GetBlockHash() *utils.Hash {
	return &blIndex.BlockHash
}

func (blIndex *BlockIndex) GetBlockTime() uint32 {
	return blIndex.Header.Time
}

func (blIndex *BlockIndex) GetBlockTimeMax() uint32 {
	return blIndex.TimeMax
}

func (blIndex *BlockIndex) GetMedianTimePast() int64 {
	median := make([]int64, 0, medianTimeSpan)
	index := blIndex
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
func (blIndex *BlockIndex) IsValid(upto uint32) bool {
	// Only validity flags allowed.
	if upto&(^BlockValidMask) != 0 {
		panic("Only validity flags allowed.")
	}
	if (blIndex.Status & BlockValidMask) != 0 {
		return false
	}
	return (blIndex.Status & BlockValidMask) >= upto
}

// RaiseValidity Raise the validity level of this block index entry.
// Returns true if the validity was changed.
func (blIndex *BlockIndex) RaiseValidity(upto uint32) bool {
	// Only validity flags allowed.
	if upto&(^BlockValidMask) != 0 {
		panic("Only validity flags allowed.")
	}
	if blIndex.Status&BlockValidMask != 0 {
		return false
	}
	if (blIndex.Status & BlockValidMask) < upto {
		blIndex.Status = (blIndex.Status & (^BlockValidMask)) | upto
		return true
	}
	return false
}

func (blIndex *BlockIndex) BuildSkip() {
	if blIndex.Prev != nil {
		blIndex.Skip = blIndex.Prev.GetAncestor(getSkipHeight(blIndex.Height))
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
func (blIndex *BlockIndex) GetAncestor(height int) *BlockIndex {
	if height > blIndex.Height || height < 0 {
		return nil
	}
	indexWalk := blIndex
	heightWalk := blIndex.Height
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

func (blIndex *BlockIndex) ToString() string {
	hash := blIndex.GetBlockHash()
	return fmt.Sprintf("BlockIndex(pprev=%p, height=%d, merkle=%s, hashBlock=%s)\n", blIndex.Prev,
		blIndex.Height, blIndex.Header.MerkleRoot.ToString(), hash.ToString())
}

func NewBlockIndex(blkHeader *BlockHeader) *BlockIndex {
	blockIndex := new(BlockIndex)
	blockIndex.SetNull()
	blockIndex.Header = *blkHeader
	return blockIndex
}
