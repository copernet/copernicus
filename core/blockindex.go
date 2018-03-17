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
	PHashBlock utils.Hash
	//! pointer to the index of the predecessor of this block
	PPrev *BlockIndex
	//! pointer to the index of some further predecessor of this block
	PSkip *BlockIndex
	//! height of the entry in the chain. The genesis block has height 0；
	Height int
	//! Which # file this block is stored in (blk?????.dat)
	File int
	//! Byte offset within blk?????.dat where this block's data is stored
	DataPosition int
	//! Byte offset within rev?????.dat where this block's undo data is stored
	UndoPosition int
	//! (memory only) Total amount of work (expected number of hashes) in the
	//! chain up to and including this block
	ChainWork big.Int
	//! Number of transactions in this block.
	//! Note: in a potential headers-first mode, this number cannot be relied
	//! upon
	Txs int
	//! (memory only) Number of transactions in the chain up to and including
	//! this block.
	//! This value will be non-zero only if and only if transactions for this
	//! block and all its parents are available. Change to 64-bit type when
	//! necessary; won't happen before 2030
	ChainTx int
	//! Verification status of this block. See enum BlockStatus
	Status uint32
	// block header
	Version    int32
	MerkleRoot utils.Hash
	Time       uint32
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
	blIndex.PHashBlock = utils.Hash{}
	blIndex.PPrev = nil
	blIndex.PSkip = nil
	blIndex.MerkleRoot = utils.Hash{}

	blIndex.Height = 0
	blIndex.File = 0
	blIndex.DataPosition = 0
	blIndex.UndoPosition = 0
	blIndex.ChainWork = big.Int{}
	blIndex.ChainTx = 0
	blIndex.Txs = 0
	blIndex.Status = 0
	blIndex.SequenceID = 0
	blIndex.TimeMax = 0

	blIndex.Version = 0
	blIndex.Time = 0
	blIndex.Bits = 0
	blIndex.Nonce = 0
}

func (blIndex *BlockIndex) GetBlockPos() DiskBlockPos {
	var ret DiskBlockPos
	if (blIndex.Status & BlockHaveData) != 0 {
		ret.File = blIndex.File
		ret.Pos = blIndex.DataPosition
	}

	return ret
}

func (blIndex *BlockIndex) GetUndoPos() DiskBlockPos {
	var ret DiskBlockPos
	if (blIndex.Status & BlockHaveUndo) != 0 {
		ret.File = blIndex.File
		ret.Pos = blIndex.UndoPosition
	}

	return ret
}

func (blIndex *BlockIndex) GetBlockHeader() BlockHeader {
	bl := BlockHeader{}
	bl.Version = blIndex.Version
	if blIndex.PPrev != nil {
		bl.HashPrevBlock = *blIndex.PPrev.GetBlockHash()
	}
	bl.HashMerkleRoot = blIndex.MerkleRoot
	bl.Time = blIndex.Time
	bl.Bits = blIndex.Bits
	bl.Nonce = blIndex.Nonce
	return bl
}

func (blIndex *BlockIndex) GetBlockHash() *utils.Hash {
	// return a pointer: index.GetBlockHash().ToString() works
	return &blIndex.PHashBlock
}

func (blIndex *BlockIndex) GetBlockTime() uint32 {
	return blIndex.Time
}

func (blIndex *BlockIndex) GetBlockTimeMax() uint32 {
	return blIndex.TimeMax
}

func (blIndex *BlockIndex) GetMedianTimePast() int64 {
	pmedian := make([]int64, 0, medianTimeSpan)
	pindex := blIndex
	numNodes := 0
	for i := 0; i < medianTimeSpan && pindex != nil; i++ {
		pmedian = append(pmedian, int64(pindex.GetBlockTime()))
		//fmt.Println("GetMedianTimePast index : ", i, ", time : ", pmedian[i], ", pindex.height : ", pindex.Height)
		pindex = pindex.PPrev
		numNodes++
	}
	pmedian = pmedian[:numNodes]
	sort.Slice(pmedian, func(i, j int) bool {
		return pmedian[i] < pmedian[j]
	})

	return pmedian[numNodes/2]
}

func (blIndex *BlockIndex) IsValid(upto uint32) bool {
	//Only validity flags allowed.
	if upto&(^BlockValidMask) != 0 {
		panic("Only validity flags allowed.")
	}
	if (blIndex.Status & BlockValidMask) != 0 {
		return false
	}
	return (blIndex.Status & BlockValidMask) >= upto
}

//RaiseValidity Raise the validity level of this block index entry.
//Returns true if the validity was changed.
func (blIndex *BlockIndex) RaiseValidity(upto uint32) bool {
	//Only validity flags allowed.
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
	if blIndex.PPrev != nil {
		//fmt.Println("BuildSkip height : ", blIndex.Height, ", blIndex.PPrev.height : ", blIndex.PPrev.Height)
		blIndex.PSkip = blIndex.PPrev.GetAncestor(getSkipHeight(blIndex.Height))
	}
}

func invertLowestOne(n int) int {
	return n & (n - 1)
}

//getSkipHeight Compute what height to jump back to with the CBlockIndex::pskip pointer.
func getSkipHeight(height int) int {
	if height < 2 {
		return 0
	}
	if (height & 1) > 0 {
		return invertLowestOne(invertLowestOne(height-1)) + 1
	}
	return invertLowestOne(height)
}

//GetAncestor Efficiently find an ancestor of this block.
func (blIndex *BlockIndex) GetAncestor(height int) *BlockIndex {
	//fmt.Println("height : ", height, ", blIndex.Height : ", blIndex.Height)
	if height > blIndex.Height || height < 0 {
		return nil
	}
	pindexWalk := blIndex
	heightWalk := blIndex.Height
	for heightWalk > height {
		heightSkip := getSkipHeight(heightWalk)
		heightSkipPrev := getSkipHeight(heightWalk - 1)
		if pindexWalk.PSkip != nil && (heightSkip == height ||
			(heightSkip > height && !(heightSkipPrev < heightSkip-2 && heightSkipPrev >= height))) {
			// Only follow pskip if pprev->pskip isn't better than pskip->pprev.
			pindexWalk = pindexWalk.PSkip
			heightWalk = heightSkip
		} else {
			if pindexWalk.PPrev == nil {
				panic("The blockIndex pointer should not be nil")
			}
			pindexWalk = pindexWalk.PPrev
			heightWalk--
		}
	}

	return pindexWalk
}

func (blIndex *BlockIndex) ToString() string {
	hash := blIndex.GetBlockHash()
	return fmt.Sprintf("BlockIndex(pprev=%p, height=%d, merkle=%s, hashBlock=%s)\n", blIndex.PPrev,
		blIndex.Height, blIndex.MerkleRoot.ToString(), hash.ToString())
}

func NewBlockIndex(blkHeader *BlockHeader) *BlockIndex {
	blockIndex := new(BlockIndex)
	blockIndex.SetNull()
	blockIndex.Version = blkHeader.Version
	blockIndex.MerkleRoot = blkHeader.HashMerkleRoot
	blockIndex.Time = blkHeader.Time
	blockIndex.Bits = blkHeader.Bits
	blockIndex.Nonce = blkHeader.Nonce
	return blockIndex
}
