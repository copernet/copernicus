package mblock

import (
	"github.com/NebulousLabs/Sia/crypto"
	"github.com/btcboost/copernicus/util"
)

type PartialMerkleTree struct {
	txs    int
	bits   []bool
	hashes []util.Hash
	bad    bool
}

func NewPartialMerkleTree(txids []util.Hash, matches []bool) *PartialMerkleTree {
	pkt := PartialMerkleTree{}

	pkt.txs = len(txids)
	pkt.bad = false

	// calculate height of tree
	var height uint
	for pkt.calcTreeWidth(height) > 1 {
		height++
	}

	// traverse the partial tree
	pkt.TraverseAndBuild(height, 0, txids, matches)

	return &pkt
}

func (pkt *PartialMerkleTree) calcTreeWidth(height uint) uint {
	// define the type uint for height to avoid runtime panic while running Lsh or Rsh operation
	return uint((pkt.txs + (1 << height) - 1) >> height)
}

func (pkt *PartialMerkleTree) TraverseAndBuild(height uint, pos uint, txids []util.Hash, matches []bool) {
	// Determine whether this node is the parent of at least one matched txid.
	var parentOfMatch bool
	for p := pos << height; p < (pos+1)<<height && p < uint(pkt.txs); p++ {
		parentOfMatch = parentOfMatch || matches[p]
	}

	// Store as flag bit.
	if pkt.bits == nil {
		pkt.bits = make([]bool, 0)
	}
	pkt.bits = append(pkt.bits, parentOfMatch)
	if height == 0 || !parentOfMatch {
		if pkt.hashes == nil {
			pkt.hashes = make([]util.Hash, 0)
		}
		pkt.hashes = append(pkt.hashes, pkt.calcHash(height, pos, txids))
	} else {
		// Otherwise, don't store any hash, but descend into the subtrees.
		pkt.TraverseAndBuild(height-1, pos*2, txids, matches)
		if pos*2+1 < pkt.calcTreeWidth(height-1) {
			pkt.TraverseAndBuild(height-1, pos*2+1, txids, matches)
		}
	}
}

func (pkt *PartialMerkleTree) calcHash(height uint, pos uint, txids []util.Hash) util.Hash {
	if height == 0 {
		// hash at height 0 is the txids themself.
		return txids[pos]
	}
	// Calculate left hash.
	left := pkt.calcHash(height-1, pos*2, txids)
	var right util.Hash
	// Calculate right hash if not beyond the end of the array - copy left
	// hash otherwise1.
	if pos*2+1 < pkt.calcTreeWidth(height-1) {
		right = pkt.calcHash(height-1, pos*2+1, txids)
	} else {
		right = left
	}

	// Combine subhashes.
	ret := make([]byte, 2*crypto.HashSize)
	ret = append(ret, left[:]...)
	ret = append(ret, right[:]...)
	b := util.DoubleSha256Bytes(ret)

	var h util.Hash
	copy(h[:], b)
	return h
}
