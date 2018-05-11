package chain

import (
	"sort"

	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/util"
)

// Chain An in-memory blIndexed chain of blocks.
type Chain struct {
	active   		[]* blockindex.BlockIndex
	branch   		[]* blockindex.BlockIndex
	waitForTx     	map[util.Hash]* blockindex.BlockIndex
	orphan        	[]* blockindex.BlockIndex
	blockIndexMap 	map[util.Hash]* blockindex.BlockIndex
	newestBlock   	*blockindex.BlockIndex
	receiveID     	uint64
}


// Genesis Returns the blIndex entry for the genesis block of this chain,
// or nullptr if none.
func (c *Chain) Genesis() *blockindex.BlockIndex {
	if len(c.active) > 0 {
		return c.active[0]
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

func (c *Chain) TipHeight() int {
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
func (c *Chain) GetIndex(height int) *blockindex.BlockIndex {
	if height < 0 || height >= len(c.active) {
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
func (c *Chain) Height() int {
	return len(c.active) - 1
}

// SetTip Set/initialize a chain with a given tip.
func (c *Chain) SetTip(index *BlockIndex) {
	if index == nil {
		c.active = []*BlockIndex{}
		return
	}

	tmp := make([]*BlockIndex, index.Height+1)
	copy(tmp, c.active)
	c.active = tmp
	for index != nil && c.active[index.Height] != index {
		c.active[index.Height] = index
		index = index.Prev
	}
}

// FindFork Find the last common block between this chain and a block blIndex entry.
func (chain *Chain) FindFork(blIndex *BlockIndex) *BlockIndex {
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
func (chain *Chain) FindEarliestAtLeast(time int64) *BlockIndex {
	i := sort.Search(len(chain.Chain), func(i int) bool {
		return int64(chain.Chain[i].GetBlockTimeMax()) > time
	})
	if i == len(chain.Chain) {
		return nil
	}

	return chain.Chain[i]
}

func ActiveBestChain(bi *BlockIndex) bool {
	forkBlock := ActiveChain.FindFork(bi)
	if forkBlock == nil {
		return false
	}

	// maintain global variable NewestBlock
	NewestBlock = bi

	subHeight := bi.Height - forkBlock.Height
	tmpBi := make([]*BlockIndex, subHeight)
	tmpBi[subHeight-1] = bi
	for i := 0; i < subHeight; i++ {
		bi = bi.Prev
		tmpBi[subHeight-i-2] = bi
	}

	// maintain the global variable ActiveChain
	// todo should be locked
	ActiveChain.Chain = append(ActiveChain.Chain[:bi.Height+1], tmpBi...)

	// maintain global variable BranchChain
	removeBlockIndexFromBranchChain(tmpBi)
	addBlockIndexToBranchChain(tmpBi)

	return true
}

// should be before addBlockIndexToBranchChain()
func removeBlockIndexFromBranchChain(bis []*BlockIndex) {
	for i := 0; i < len(bis); i++ {
		for j := 0; j < len(BranchChain); {
			if BranchChain[j].BlockHash == bis[i].BlockHash {
				BranchChain = append(BranchChain[:j], BranchChain[j+1:]...)
			} else {
				j++
			}
		}
	}
}

// should be after removeBlockIndexFromBranchChain()
func addBlockIndexToBranchChain(bis []*BlockIndex) {
	BranchChain = append(BranchChain, bis...)
}

func FindMostWorkChain() *BlockIndex {
	// todo complete
	return nil
}