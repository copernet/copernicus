package chain

import (
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/util"
)

// Chain An in-memory blIndexed chain of blocks.
type Chain struct {
	active      []*blockindex.BlockIndex
	branch      []*blockindex.BlockIndex
	waitForTx   map[util.Hash]*blockindex.BlockIndex
	orphan      []*blockindex.BlockIndex
	indexMap    map[util.Hash]*blockindex.BlockIndex
	newestBlock *blockindex.BlockIndex
	receiveID   uint64
}

var GlobalChain *Chain

func GetInstance() *Chain {
	if GlobalChain == nil {
		GlobalChain = NewChain()
	}

	return GlobalChain
}

func NewChain() *Chain {
	return &Chain{}
}

// Genesis Returns the blIndex entry for the genesis block of this chain,
// or nullptr if none.
func (c *Chain) Genesis() *blockindex.BlockIndex {
	if len(c.active) > 0 {
		return c.active[0]
	}

	return nil
}

//find blockindex from blockIndexMap
func (c *Chain) FindBlockIndex(hash util.Hash) *blockindex.BlockIndex {
	bi, ok := c.indexMap[hash]
	if ok {
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

func (c *Chain) GetMedianTimePast() int64 {

	return 0
}

func (c *Chain) GetVersionState() int {

	return 0
}

func (c *Chain) GetTipTime() int64 {

	return 0
}

func (c *Chain) GetScriptFlags() uint32 {

	return 0
}

// GetSpecIndex Returns the blIndex entry at a particular height in this chain, or nullptr
// if no such height exists.
func (c *Chain) GetIndex(height int32) *blockindex.BlockIndex {
	if height < 0 || height >= int32(len(c.active)) {
		return nil
	}

	return c.active[height]
}

// Equal Compare two chains efficiently.
func (c *Chain) Equal(dst *Chain) bool {
	return len(c.active) == len(dst.active) &&
		c.active[len(c.active)-1] == dst.active[len(dst.active)-1]
}

// Contains /** Efficiently check whether a block is present in this chain
func (c *Chain) Contains(index *blockindex.BlockIndex) bool {
	return c.GetIndex(index.Height) == index
}

// Next Find the successor of a block in this chain, or nullptr if the given
// index is not found or is the tip.
func (c *Chain) Next(index *blockindex.BlockIndex) *blockindex.BlockIndex {
	if c.Contains(index) {
		return c.GetIndex(index.Height + 1)
	}
	return nil
}

// Height Return the maximal height in the chain. Is equal to chain.Tip() ?
// chain.Tip()->nHeight : -1.
func (c *Chain) Height() int32 {
	return int32(len(c.active) - 1)
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
}

func (c *Chain) GetAncestor(height int32) *blockindex.BlockIndex {
	// todo
	return nil
}

func (ch *Chain) GetLocator(index *blockindex.BlockIndex) *BlockLocator {
	step := 1
	blockHashList := make([]util.Hash, 0, 32)
	if index == nil {
		index = ch.Tip()
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
		if ch.Contains(index) {
			index = ch.GetIndex(height)
		} else {
			index = ch.GetAncestor(height)
		}
		if len(blockHashList) > 10 {
			step *= 2
		}
	}
	return NewBlockLocator(blockHashList)
}

// FindFork Find the last common block between this chain and a block blIndex entry.
func (chain *Chain) FindFork(blIndex *blockindex.BlockIndex) *blockindex.BlockIndex {
	if blIndex == nil {
		return nil
	}

	if blIndex.Height > chain.Height() {
		blIndex = blIndex.GetAncestor(chain.Height())
	}

	for blIndex != nil && !chain.Contains(blIndex) {
		blIndex = blIndex.Prev
	}
	return blIndex
}

// FindEarliestAtLeast Find the earliest block with timestamp equal or greater than the given.
func (chain *Chain) FindEarliestAtLeast(time int64) *blockindex.BlockIndex {

	return nil
}

func (chain *Chain) ActiveBest(bi *blockindex.BlockIndex) error {

	return nil
}

func (chain *Chain) RemoveFromBranch(bis []*blockindex.BlockIndex) {

}

func (chain *Chain) AddToBranch(bis []*blockindex.BlockIndex) {

}

func (chain *Chain) FindMostWorkChain() *blockindex.BlockIndex {

	return nil
}

func (c *Chain) AddToIndexMap(bi *blockindex.BlockIndex) error {
	return nil
}

func (c *Chain) GetActiveHeight(hash *util.Hash) (int32, error) {
	return 0, nil
}
