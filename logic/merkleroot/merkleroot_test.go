package merkleroot

import (
	"bytes"
	"crypto/sha256"

	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/util"
)

// Older version of the merkle root computation code, for comparison.
func BlockBuildMerkleTree(bk *block.Block, mutated *bool, merkleTree []util.Hash) util.Hash {
	// Safe upper bound for the number of total nodes.
	merkleTree = make([]util.Hash, 0, len(bk.Txs)*2+16)
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
		return util.HashZero
	}

	return merkleTree[len(merkleTree)-1]
}
