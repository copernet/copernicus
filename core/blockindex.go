package core

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/btcboost/copernicus/utils"
)

/**
 * The block chain is a tree shaped structure starting with the genesis block at
 * the root, with each block potentially having multiple candidates to be the
 * next block. A blockindex may have multiple pprev pointing to it, but at most
 * one of them can be part of the currently active branch.
 */

type BlockIndex struct {
	//! pointer to the hash of the block, if any.
	BlockHash utils.Hash
	//! pointer to the index of the predecessor of this block
	Prev *BlockIndex
	//! pointer to the index of some further predecessor of this block
	Skip *BlockIndex
	//! height of the entry in the chain. The genesis block has height 0；
	Height int
	//! Which # file this block is stored in (blk?????.dat)
	File int
	//! Byte offset within blk?????.dat where this block's data is stored
	DataPos int
	//! Byte offset within rev?????.dat where this block's undo data is stored
	UndoPos int
	//! (memory only) Total amount of work (expected number of hashes) in the
	//! chain up to and including this block
	ChainWork big.Int
	//! Number of transactions in this block.
	//! Note: in a potential headers-first mode, this number cannot be relied
	//! upon
	TxCount int
	//! (memory only) Number of transactions in the chain up to and including
	//! this block.
	//! This value will be non-zero only if and only if transactions for this
	//! block and all its parents are available. Change to 64-bit type when
	//! necessary; won't happen before 2030
	ChainTxCount int
	//! Verification status of this block. See enum BlockStatus
	Status uint32
	// block header
	Version    int32
	MerkleRoot utils.Hash
	TimeStamp  uint32
	Bits       uint32
	Nonce      uint32
	//! (memory only) Sequential id assigned to distinguish order in which
	//! blocks are received.
	SequenceID int32
	//! (memory only) Maximum nTime in the chain upto and including this block.
	TimeMax uint32
}

const medianTimeSpan = 11

func (blIndex *BlockIndex) SetNull() {
	blIndex.BlockHash = utils.Hash{}
	blIndex.Prev = nil
	blIndex.Skip = nil
	blIndex.MerkleRoot = utils.Hash{}

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

	blIndex.Version = 0
	blIndex.TimeStamp = 0
	blIndex.Bits = 0
	blIndex.Nonce = 0
}

func (blIndex *BlockIndex) GetBlockPos() DiskBlockPos {
	var ret DiskBlockPos
	if (blIndex.Status & BLOCK_HAVE_DATA) != 0 {
		ret.File = blIndex.File
		ret.Pos = blIndex.DataPos
	}

	return ret
}

func (blIndex *BlockIndex) GetUndoPos() DiskBlockPos {
	var ret DiskBlockPos
	if (blIndex.Status & BLOCK_HAVE_UNDO) != 0 {
		ret.File = blIndex.File
		ret.Pos = blIndex.UndoPos
	}

	return ret
}

func (blIndex *BlockIndex) GetBlockHeader() BlockHeader {
	bl := BlockHeader{}
	bl.Version = blIndex.Version
	if blIndex.Prev != nil {
		bl.HashPrevBlock = *blIndex.Prev.GetBlockHash()
	}
	bl.MerkleRoot = blIndex.MerkleRoot
	bl.TimeStamp = blIndex.TimeStamp
	bl.Bits = blIndex.Bits
	bl.Nonce = blIndex.Nonce
	return bl
}

func (blIndex *BlockIndex) GetBlockHash() *utils.Hash {
	return &blIndex.BlockHash
}

func (blIndex *BlockIndex) GetBlockTime() uint32 {
	return blIndex.TimeStamp
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

// Check whether this block index entry is valid up to the passed validity
// level.
func (blIndex *BlockIndex) IsValid(upto uint32) bool {
	//Only validity flags allowed.
	if upto&(^BLOCK_VALID_MASK) != 0 {
		panic("Only validity flags allowed.")
	}
	if (blIndex.Status & BLOCK_VALID_MASK) != 0 {
		return false
	}
	return (blIndex.Status & BLOCK_VALID_MASK) >= upto
}

//RaiseValidity Raise the validity level of this block index entry.
//Returns true if the validity was changed.
func (blIndex *BlockIndex) RaiseValidity(upto uint32) bool {
	//Only validity flags allowed.
	if upto&(^BLOCK_VALID_MASK) != 0 {
		panic("Only validity flags allowed.")
	}
	if blIndex.Status&BLOCK_VALID_MASK != 0 {
		return false
	}
	if (blIndex.Status & BLOCK_VALID_MASK) < upto {
		blIndex.Status = (blIndex.Status & (^BLOCK_VALID_MASK)) | upto
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

//getSkipHeight Compute what height to jump back to with the CBlockIndex::pskip pointer.
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

//GetAncestor efficiently find an ancestor of this block.
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
		blIndex.Height, blIndex.MerkleRoot.ToString(), hash.ToString())
}

func NewBlockIndex(blkHeader *BlockHeader) *BlockIndex {
	blockIndex := new(BlockIndex)
	blockIndex.SetNull()
	blockIndex.Version = blkHeader.Version
	blockIndex.MerkleRoot = blkHeader.MerkleRoot
	blockIndex.TimeStamp = blkHeader.TimeStamp
	blockIndex.Bits = blkHeader.Bits
	blockIndex.Nonce = blkHeader.Nonce
	return blockIndex
}
