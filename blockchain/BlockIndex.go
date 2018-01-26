package blockchain

import (
	"math/big"

	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

/**
 * The block chain is a tree shaped structure starting with the genesis block at
 * the root, with each block potentially having multiple candidates to be the
 * next block. A blockindex may have multiple pprev pointing to it, but at most
 * one of them can be part of the currently active branch.
 */

type BlockIndex struct {
	PHashBlock utils.Hash
	PPrev      *BlockIndex

	//! pointer to the index of some further predecessor of this block
	PSkip  *BlockIndex
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

	Status uint32

	// block header
	Version    int32
	MerkleRoot utils.Hash
	Time       uint32
	Bits       uint32
	Nonce      uint32

	SequenceID int32
}

func (blockIndex *BlockIndex) SetNull() {
	blockIndex.PHashBlock = utils.Hash{}
	blockIndex.PPrev = nil
	blockIndex.PSkip = nil
	blockIndex.MerkleRoot = utils.Hash{}

	blockIndex.Height = 0
	blockIndex.File = 0
	blockIndex.DataPosition = 0
	blockIndex.UndoPosition = 0
	blockIndex.ChainWork = big.Int{}
	blockIndex.Txs = 0
	blockIndex.Status = 0
	blockIndex.Version = 0
	blockIndex.Time = 0
	blockIndex.Bits = 0
	blockIndex.Nonce = 0
	blockIndex.SequenceID = 0
	blockIndex.Bits = 0

}

func NewBlockIndex(block *model.Block) *BlockIndex {
	blockIndex := new(BlockIndex)
	blockIndex.Version = block.BlockHeader.Version
	blockIndex.MerkleRoot = block.BlockHeader.HashMerkleRoot
	blockIndex.Time = block.BlockHeader.Time
	blockIndex.Bits = block.BlockHeader.Bits
	blockIndex.Nonce = block.BlockHeader.Nonce
	return blockIndex

}
