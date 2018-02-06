package consensus

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

/* This implements a constant-space merkle root/path calculator, limited to 2^32
 * leaves. */
func merkleComputation(leaves []utils.Hash, root *utils.Hash, pmutated *bool, branchpos uint32, pbranch *[]utils.Hash) {
	if pbranch != nil {
		*pbranch = (*pbranch)[:0]
	}
	if len(leaves) == 0 {
		if pmutated != nil {
			*pmutated = false
		}
		if root != nil {
			*root = utils.Hash{}
		}
		return
	}
	mutated := false
	// count is the number of leaves processed so far.
	count := uint32(0)
	// inner is an array of eagerly computed subtree hashes, indexed by tree
	// level (0 being the leaves).
	// For example, when count is 25 (11001 in binary), inner[4] is the hash of
	// the first 16 leaves, inner[3] of the next 8 leaves, and inner[0] equal to
	// the last leaf. The other inner entries are undefined.
	var inner [32]utils.Hash
	// Which position in inner is a hash that depends on the matching leaf.
	matchlevel := -1
	// First process all leaves into 'inner' values.
	for int(count) < len(leaves) {
		h := leaves[count]
		match := count == branchpos
		count++
		level := 0
		// For each of the lower bits in count that are 0, do 1 step. Each
		// corresponds to an inner value that existed before processing the
		// current leaf, and each needs a hash to combine it.
		for ; (count & (uint32(1) << uint(level))) == 0; level++ {
			if pbranch != nil {
				if match {
					*pbranch = append(*pbranch, inner[level])
				} else if matchlevel == level {
					*pbranch = append(*pbranch, h)
					match = true
				}
			}
			if inner[level].IsEqual(&h) {
				mutated = true
			}
			var tmp []byte
			tmp = append(tmp, inner[level][:]...)
			tmp = append(tmp, h[:]...)
			h = core.DoubleSha256Hash(tmp)
		}
		inner[level] = h
		if match {
			matchlevel = level
		}
	}
	// Do a final 'sweep' over the rightmost branch of the tree to process
	// odd levels, and reduce everything to a single top value.
	// Level is the level (counted from the bottom) up to which we've swept.
	level := 0
	// As long as bit number level in count is zero, skip it. It means there
	// is nothing left at this level.
	for ; (count & (uint32(1) << uint(level))) == 0; level++ {
	}
	h := inner[level]
	matchh := matchlevel == level
	for count != (uint32(1) << uint(level)) {
		// If we reach this point, h is an inner value that is not the top.
		// We combine it with itself (Bitcoin's special rule for odd levels in
		// the tree) to produce a higher level one.
		if pbranch != nil && matchh {
			*pbranch = append(*pbranch, h)
		}
		var tmp []byte
		tmp = append(tmp, h[:]...)
		tmp = append(tmp, h[:]...)
		h = core.DoubleSha256Hash(tmp)
		// Increment count to the value it would have if two entries at this
		// level had existed.
		count += (uint32(1) << uint(level))
		level++
		// And propagate the result upwards accordingly.
		for ; (count & (uint32(1) << uint(level))) == 0; level++ {
			if pbranch != nil {
				if matchh {
					*pbranch = append(*pbranch, inner[level])
				} else if matchlevel == level {
					*pbranch = append(*pbranch, h)
					matchh = true
				}
			}
			var tmp []byte
			tmp = append(tmp, inner[level][:]...)
			tmp = append(tmp, h[:]...)
			h = core.DoubleSha256Hash(tmp)
		}
	}
	if pmutated != nil {
		*pmutated = mutated
	}
	if root != nil {
		*root = h
	}
}

func ComputeMerkleRoot(leaves []utils.Hash, mutated *bool) utils.Hash {
	var hash utils.Hash
	merkleComputation(leaves, &hash, mutated, 0xffffffff, nil)
	return hash
}

func ComputeMerkleBranch(leaves []utils.Hash, position uint32) []utils.Hash {
	var ret []utils.Hash
	merkleComputation(leaves, nil, nil, position, &ret)
	return ret
}

func ComputeMerkleRootFromBranch(leaf *utils.Hash, branch []utils.Hash, index uint32) utils.Hash {
	hash := *leaf
	for i := 0; i < len(branch); i++ {
		if (index & 1) == 1 {
			var tmp []byte
			tmp = append(tmp, branch[i][:]...)
			tmp = append(tmp, hash[:]...)
			hash = core.DoubleSha256Hash(tmp)
		} else {
			var tmp []byte
			tmp = append(tmp, hash[:]...)
			tmp = append(tmp, branch[i][:]...)
			hash = core.DoubleSha256Hash(tmp)
		}
		index >>= 1
	}
	return hash
}

func BlockMerkleRoot(block *model.Block, mutated *bool) utils.Hash {
	leaves := make([]utils.Hash, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		leaves[i] = block.Transactions[i].TxHash()
	}
	return ComputeMerkleRoot(leaves, mutated)
}

func BlockMerkleBranch(block *model.Block, position uint32) []utils.Hash {
	leaves := make([]utils.Hash, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		leaves[i] = block.Transactions[i].TxHash()
	}
	return ComputeMerkleBranch(leaves, position)
}
