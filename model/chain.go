package model

import (
	"sort"
)

// Chain An in-memory indexed chain of blocks.
type Chain struct {
	VChain []*BlockIndex
}

// Genesis Returns the index entry for the genesis block of this chain,
// or nullptr if none.
func (chain *Chain) Genesis() *BlockIndex {
	if len(chain.VChain) > 0 {
		return chain.VChain[0]
	}

	return nil
}

// Tip Returns the index entry for the tip of this chain, or nullptr if none.
func (chain *Chain) Tip() *BlockIndex {
	if len(chain.VChain) > 0 {
		return chain.VChain[len(chain.VChain)-1]
	}

	return nil
}

// GetSpecIndex Returns the index entry at a particular height in this chain, or nullptr
// if no such height exists.
func (chain *Chain) GetSpecIndex(height int) *BlockIndex {
	if height < 0 || height >= len(chain.VChain) {
		return nil
	}

	return chain.VChain[height]
}

// Equal Compare two chains efficiently.
func (chain *Chain) Equal(dst *Chain) bool {
	return len(chain.VChain) == len(dst.VChain) &&
		chain.VChain[len(chain.VChain)-1] == dst.VChain[len(dst.VChain)-1]
}

// Contains /** Efficiently check whether a block is present in this chain
func (chain *Chain) Contains(pindex *BlockIndex) bool {
	return chain.GetSpecIndex(pindex.Height) == pindex
}

// Next Find the successor of a block in this chain, or nullptr if the given
// index is not found or is the tip.
func (chain *Chain) Next(pindex *BlockIndex) *BlockIndex {
	if chain.Contains(pindex) {
		return chain.GetSpecIndex(pindex.Height + 1)
	}
	return nil
}

//Height Return the maximal height in the chain. Is equal to chain.Tip() ?
// chain.Tip()->nHeight : -1.
func (chain *Chain) Height() int {
	return len(chain.VChain) - 1
}

//SetTip Set/initialize a chain with a given tip.
func (chain *Chain) SetTip(pindex *BlockIndex) {
	if pindex == nil {
		chain.VChain = []*BlockIndex{}
		return
	}

	tmp := make([]*BlockIndex, pindex.Height+1)
	copy(tmp, chain.VChain)
	chain.VChain = tmp
	for pindex != nil && chain.VChain[pindex.Height] != pindex {
		chain.VChain[pindex.Height] = pindex
		pindex = pindex.PPrev
	}
}

//GetLocator Return a CBlockLocator that refers to a block in this chain (by default
//the tip).
func (chain *Chain) GetLocator(pindex *BlockIndex) {

}

//FindFork Find the last common block between this chain and a block index entry.
func (chain *Chain) FindFork(pindex *BlockIndex) *BlockIndex {
	if pindex == nil {
		return nil
	}

	if pindex.Height > chain.Height() {
		pindex = pindex.GetAncestor(chain.Height())
	}

	for pindex != nil && !chain.Contains(pindex) {
		pindex = pindex.PPrev
	}
	return pindex
}

//FindEarliestAtLeast Find the earliest block with timestamp equal or greater than the given.
func (chain *Chain) FindEarliestAtLeast(time int64) *BlockIndex {
	i := sort.Search(len(chain.VChain), func(i int) bool {
		return int64(chain.VChain[i].GetBlockTimeMax()) > time
	})
	if i == len(chain.VChain) {
		return nil
	}

	return chain.VChain[i]
}
