package ldb

import (
	"crypto"
	"errors"
	"fmt"
	"github.com/btccom/copernicus/utils"
)

const (
	//  [0:4]  Block file (4 bytes)
	//  [4:8]  File offset (4 bytes)
	//  [8:12] Block length (4 bytes)
	BlockLocationSize = 12
	BlockHeaderSize   = 16 + utils.MaxHashStringSize
)

type pendingBlock struct {
	hash  *crypto.Hash
	bytes []byte
}

//   <bucketid><key>
func bucketizedKey(bucketID [4]byte, key []byte) []byte {
	bKey := make([]byte, 4+len(key))
	copy(bKey, bucketID[:])
	copy(bKey[4:], key)
	return bKey
}

type transaction struct {
	managed          bool
	closed           bool
	writable         bool
	pendingBlocks    map[utils.Hash]int
	pendingBlockData []pendingBlock
}

func (tx *transaction) checkClosed() error {
	// The transaction is no longer valid if it has been closed.
	if tx.closed {
		return errors.New("the transaction is closed")
	}
	return nil
}

//todo implement ths haskey flow
func (tx *transaction) hasKey(_ []byte) bool {
	return !tx.writable
}

func (tx *transaction) hasBlock(hash *utils.Hash) bool {
	if _, exists := tx.pendingBlocks[*hash]; exists {
		return true
	}
	return tx.hasKey(bucketizedKey(blockIdxBucketID, hash[:]))
}

func (tx *transaction) HasBlock(hash *utils.Hash) (bool, error) {
	if err := tx.checkClosed(); err != nil {
		return false, err
	}
	return tx.hasBlock(hash), nil
}

func (tx *transaction) HasBlocks(hashes []utils.Hash) ([]bool, error) {
	if err := tx.checkClosed(); err != nil {
		return nil, err
	}
	results := make([]bool, len(hashes))
	for i := range hashes {
		results[i] = tx.hasBlock(&hashes[i])
	}
	return results, nil
}

//todo implement the blockIdxBucket
func (tx *transaction) fetchBlockRow(hash *utils.Hash) ([]byte, error) {
	//blockRow := tx.blockIdxBucket.Get(hash[:])
	var blockRow []byte
	if blockRow == nil {
		return blockRow, fmt.Errorf("block %s does not exist", hash)
	}
	return blockRow, nil
}

func (tx *transaction) FetchBlockHeader(hash *utils.Hash) ([]byte, error) {
	if err := tx.checkClosed(); err != nil {
		return nil, err
	}
	if idx, exists := tx.pendingBlocks[*hash]; exists {
		blockBytes := tx.pendingBlockData[idx].bytes
		return blockBytes[0:BlockHeaderSize:BlockHeaderSize], nil
	}
	blockRow, err := tx.fetchBlockRow(hash)
	if err != nil {
		return nil, err
	}
	endOffset := BlockLocationSize + BlockHeaderSize
	return blockRow[BlockLocationSize:endOffset:endOffset], nil
}
