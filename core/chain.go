package core

import (
	"sort"

	"github.com/btcboost/copernicus/utils"
)

// todo these global variable should be protected by lock
type Chain struct (
	ActiveChain   Chain
	BranchChain   []*BlockIndex
	WaitForTx     map[utils.Hash]*BlockIndex
	Orphan        []*BlockIndex
	BlockIndexMap map[utils.Hash]*BlockIndex
	NewestBlock   *BlockIndex
	ReceiveID     uint64
)

// Chain An in-memory blIndexed chain of blocks.
/*type Chain struct {
	Chain []*BlockIndex
}*/

// Genesis Returns the blIndex entry for the genesis block of this chain,
// or nullptr if none.
func (chain *Chain) Genesis() *BlockIndex {
	if len(chain.Chain) > 0 {
		return chain.Chain[0]
	}

	return nil
}

// Tip Returns the blIndex entry for the tip of this chain, or nullptr if none.
func (chain *Chain) Tip() *BlockIndex {
	if len(chain.Chain) > 0 {
		return chain.Chain[len(chain.Chain)-1]
	}

	return nil
}

// GetSpecIndex Returns the blIndex entry at a particular height in this chain, or nullptr
// if no such height exists.
func (chain *Chain) GetSpecIndex(height int) *BlockIndex {
	if height < 0 || height >= len(chain.Chain) {
		return nil
	}

	return chain.Chain[height]
}

// Equal Compare two chains efficiently.
func (chain *Chain) Equal(dst *Chain) bool {
	return len(chain.Chain) == len(dst.Chain) &&
		chain.Chain[len(chain.Chain)-1] == dst.Chain[len(dst.Chain)-1]
}

// Contains /** Efficiently check whether a block is present in this chain
func (chain *Chain) Contains(blIndex *BlockIndex) bool {
	return chain.GetSpecIndex(blIndex.Height) == blIndex
}

// Next Find the successor of a block in this chain, or nullptr if the given
// blIndex is not found or is the tip.
func (chain *Chain) Next(blIndex *BlockIndex) *BlockIndex {
	if chain.Contains(blIndex) {
		return chain.GetSpecIndex(blIndex.Height + 1)
	}
	return nil
}

// Height Return the maximal height in the chain. Is equal to chain.Tip() ?
// chain.Tip()->nHeight : -1.
func (chain *Chain) Height() int {
	return len(chain.Chain) - 1
}

// SetTip Set/initialize a chain with a given tip.
func (chain *Chain) SetTip(blIndex *BlockIndex) {
	if blIndex == nil {
		chain.Chain = []*BlockIndex{}
		return
	}

	tmp := make([]*BlockIndex, blIndex.Height+1)
	copy(tmp, chain.Chain)
	chain.Chain = tmp
	for blIndex != nil && chain.Chain[blIndex.Height] != blIndex {
		chain.Chain[blIndex.Height] = blIndex
		blIndex = blIndex.Prev
	}
}

// GetLocator Return a CBlockLocator that refers to a block in this chain (by default
// the tip).
func (chain *Chain) GetLocator(blIndex *BlockIndex) {

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
