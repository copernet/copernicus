package mblock

import (
	"io"
	"math"

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
	pmt := PartialMerkleTree{}

	pmt.txs = len(txids)
	pmt.bad = false

	// calculate height of tree
	var height uint
	for pmt.calcTreeWidth(height) > 1 {
		height++
	}

	// traverse the partial tree
	pmt.TraverseAndBuild(height, 0, txids, matches)

	return &pmt
}

func (pmt *PartialMerkleTree) calcTreeWidth(height uint) uint {
	// define the type uint for height to avoid runtime panic while running Lsh or Rsh operation
	return uint((pmt.txs + (1 << height) - 1) >> height)
}

func (pmt *PartialMerkleTree) TraverseAndBuild(height uint, pos uint, txids []util.Hash, matches []bool) {
	// Determine whether this node is the parent of at least one matched txid.
	var parentOfMatch bool
	for p := pos << height; p < (pos+1)<<height && p < uint(pmt.txs); p++ {
		parentOfMatch = parentOfMatch || matches[p]
	}

	// Store as flag bit.
	if pmt.bits == nil {
		pmt.bits = make([]bool, 0)
	}
	pmt.bits = append(pmt.bits, parentOfMatch)
	if height == 0 || !parentOfMatch {
		if pmt.hashes == nil {
			pmt.hashes = make([]util.Hash, 0)
		}
		pmt.hashes = append(pmt.hashes, pmt.calcHash(height, pos, txids))
	} else {
		// Otherwise, don't store any hash, but descend into the subtrees.
		pmt.TraverseAndBuild(height-1, pos*2, txids, matches)
		if pos*2+1 < pmt.calcTreeWidth(height-1) {
			pmt.TraverseAndBuild(height-1, pos*2+1, txids, matches)
		}
	}
}

func (pmt *PartialMerkleTree) calcHash(height uint, pos uint, txids []util.Hash) util.Hash {
	if height == 0 {
		// hash at height 0 is the txids themself.
		return txids[pos]
	}
	// Calculate left hash.
	left := pmt.calcHash(height-1, pos*2, txids)
	var right util.Hash
	// Calculate right hash if not beyond the end of the array - copy left
	// hash otherwise1.
	if pos*2+1 < pmt.calcTreeWidth(height-1) {
		right = pmt.calcHash(height-1, pos*2+1, txids)
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

func (pmt *PartialMerkleTree) Serialize(w io.Writer) (err error) {
	err = util.WriteVarInt(w, uint64(pmt.txs))
	if err != nil {
		return
	}

	err = util.WriteVarInt(w, uint64(len(pmt.hashes)))
	if err != nil {
		return
	}
	for _, hash := range pmt.hashes {
		_, err = hash.Serialize(w)
		if err != nil {
			return
		}
	}

	bs := make([]byte, 0, (len(pmt.bits)+7)/8)
	for p := 0; p < len(pmt.bits); p++ {
		if pmt.bits[p] {
			bs[p/8] |= uint8(1 << uint(p%8))
		}
	}
	err = util.WriteVarBytes(w, bs)
	if err != nil {
		return
	}

	return
}

func (pmt *PartialMerkleTree) Unserialize(r io.Reader) (err error) {
	txs, err := util.ReadVarInt(r)
	if err != nil {
		return
	}
	pmt.txs = int(txs)

	length, err := util.ReadVarInt(r)
	if err != nil {
		return
	}

	hashes := make([]util.Hash, length)
	for i := uint64(0); i < length; i++ {
		_, err = hashes[i].Unserialize(r)
		if err != nil {
			return
		}
	}
	pmt.hashes = hashes

	bs, err := util.ReadVarBytes(r, math.MaxUint32, "")
	if err != nil {
		return
	}
	pmt.bits = make([]bool, len(bs)*8)
	for p := 0; p < len(pmt.bits); p++ {
		pmt.bits[p] = bs[p/8]&(1<<uint(p%8)) != 0
	}
	pmt.bad = false

	return
}
