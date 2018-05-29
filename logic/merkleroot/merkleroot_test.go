package merkleroot

import (
	"bytes"
	"crypto/sha256"
	"math"
	"testing"

	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/util"
)

// Older version of the merkle root computation code, for comparison.
func BlockBuildMerkleTree(bk *block.Block, mutated *bool) (util.Hash, []util.Hash) {
	// Safe upper bound for the number of total nodes.
	merkleTree := make([]util.Hash, 0, len(bk.Txs)*2+16)
	for _, item := range bk.Txs {
		merkleTree = append(merkleTree, item.GetHash())
	}

	var j int
	var mutate bool
	for size := len(bk.Txs); size > 1; size = (size + 1) / 2 {
		for i := 0; i < size; i += 2 {
			i2 := i + 1
			if i+1 > size-1 {
				i2 = size - 1
			}

			if i2 == i+1 && i2+1 == size && merkleTree[j+i] == merkleTree[j+i2] {
				// Two identical hashes at the end of the list at a particular
				// level.
				mutate = true
			}
			buf := bytes.NewBuffer(make([]byte, 0, sha256.Size*2))
			buf.Write(merkleTree[j+i][:])
			buf.Write(merkleTree[j+i2][:])
			merkleTree = append(merkleTree, util.DoubleSha256Hash(buf.Bytes()))
		}
		j += size
	}

	if mutated != nil {
		*mutated = mutate
	}

	if len(merkleTree) == 0 {
		return util.HashZero, nil
	}

	return merkleTree[len(merkleTree)-1], merkleTree
}

// Older version of the merkle branch computation code, for comparison.
func BlockGetMerkleBranch(bk *block.Block, merkletree []util.Hash, index int) []util.Hash {
	merkleBranch := make([]util.Hash, 0)
	var j int
	for size := len(bk.Txs); size > 1; size = (size + 1) / 2 {
		i := index ^ 1
		if i > size-1 {
			i = size - 1
		}
		merkleBranch = append(merkleBranch, merkletree[j+i])
		index = index >> 1
		j += size
	}

	return merkleBranch
}

func ctz(i uint32) int {
	if i == 0 {
		return 0
	}

	var j int
	for i&1 == 0 {
		j++
		i = i >> 1
	}

	return j
}

func TestMerkle(t *testing.T) {
	for i := 0; i < 32; i++ {
		// Try 32 block sizes: all sizes from 0 to 16 inclusive, and then 15
		// random sizes.
		ntx := i
		if i<<16 == 0 {
			ntx = 17 + util.GetRandInt(math.MaxUint32)%4000
			// Try up to 3 mutations.
			for mutate := 0; mutate <= 3; mutate++ {
				// The last how many transactions to duplicate first.
				duplicate1 := 0
				if mutate >= 1 {
					duplicate1 = 1 << uint(ctz(uint32(ntx)))
				}
				if duplicate1 >= ntx {
					// Duplication of the entire tree results in a different root
					// (it adds a level).
					break
				}

				// The resulting number of transactions after the first duplication.
				ntx1 := ntx + duplicate1
				// Likewise for the second mutation.
				duplicate2 := 0
				if mutate >= 2 {
					duplicate2 = 1 << uint(ctz(uint32(ntx1)))
				}
				if duplicate2 > ntx1 {
					break
				}
				ntx2 := ntx1 + duplicate2
				// And for the third mutation.
				duplicate3 := 0
				if mutate >= 3 {
					duplicate3 = 1 << uint(ctz(uint32(ntx2)))
				}
				if duplicate3 >= ntx2 {
					break
				}
				//ntx3 := ntx2 + duplicate3
				// Build a block with ntx different transactions
				bk := block.NewBlock()
				bk.Txs = make([]*tx.Tx, ntx)
				for j := 0; j < ntx; j++ {
					bk.Txs[j] = tx.NewTx(uint32(j), 0x01)
				}

				// Compute the root of the block before mutating it.
				var unmutatedMutated bool
				unmutateRoot := BlockMerkleRoot(bk, &unmutatedMutated)
				if unmutatedMutated {
					t.Error("calculate error")
				}

				// Optionally mutate by duplicating the last transactions, resulting
				// in the same merkle root.
				for j := 0; j < duplicate1; j++ {
					bk.Txs = append(bk.Txs, bk.Txs[ntx+j-duplicate1])
				}
				for j := 0; j < duplicate2; j++ {
					bk.Txs = append(bk.Txs, bk.Txs[ntx1+j-duplicate2])
				}
				for j := 0; j < duplicate3; j++ {
					bk.Txs = append(bk.Txs, bk.Txs[ntx2+j-duplicate3])
				}

				// Compute the merkle root and merkle tree using the old mechanism.
				var oldMutated bool
				oldRoot, merkletree := BlockBuildMerkleTree(bk, &oldMutated)
				// Compute the merkle root using the new mechanism.
				var newMutated bool
				newRoot := BlockMerkleRoot(bk, &newMutated)
				if oldRoot != newRoot {
					t.Error("error")
				}
				if newRoot != unmutateRoot {
					t.Error("error")
				}
				if !((newRoot == util.HashZero) == (ntx == 0)) {
					t.Error("error")
				}
				if oldMutated != newMutated {
					t.Error("error")
				}

				// If no mutation was done (once for every ntx value), try up to 16
				// branches.

				if mutate == 0 {
					for loop := 0; loop < min(ntx, 16); loop++ {
						// If ntx <= 16, try all branches. Otherise, try 16 random
						// ones.
						mtx := loop
						if ntx > 16 {
							mtx = util.GetRandInt(math.MaxUint32) % ntx
						}
						newBranch := BlockMerkleBranch(bk, uint32(mtx))
						oldBranch := BlockGetMerkleBranch(bk, merkletree, mtx)
						if len(newBranch) != len(oldBranch) {
							t.Error("error")
						} else {
							for i := 0; i < len(oldBranch); i++ {
								if oldBranch[i] != newBranch[i] {
									t.Error("error")
								}
							}
						}

						hash := bk.Txs[mtx].GetHash()
						if ComputeMerkleRootFromBranch(&hash, newBranch, uint32(mtx)) != oldRoot {
							t.Error("error")
						}
					}
				}
			}
		}
	}
}

func min(a, b int) int {
	if a <= b {
		return a
	}

	return b
}
