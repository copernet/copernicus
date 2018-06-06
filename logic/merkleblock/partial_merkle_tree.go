package merkleblock

import (
	"crypto/sha256"
	"io"
	"math"

	"github.com/copernet/copernicus/util"
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
	ret := make([]byte, 2*util.Hash256Size)
	ret = append(ret, left[:]...)
	ret = append(ret, right[:]...)
	b := util.DoubleSha256Bytes(ret)

	var h util.Hash
	copy(h[:], b)
	return h
}

func (pmt *PartialMerkleTree) ExtractMatches(matches []util.Hash, items []int) *util.Hash {
	matches = matches[:0]
	// An empty set will not work
	if pmt.txs == 0 {
		return &util.Hash{}
	}
	// Check for excessively high numbers of transactions.
	// FIXME: Track the maximum block size we've seen and use it here.

	// There can never be more hashes provided than one for every txid.
	if len(pmt.hashes) > pmt.txs {
		return &util.Hash{}
	}
	// There must be at least one bit per node in the partial tree, and at least
	// one node per hash.
	if len(pmt.bits) < len(pmt.hashes) {
		return &util.Hash{}
	}
	// calculate height of tree.
	var height uint
	for pmt.calcTreeWidth(height) > 1 {
		height++
	}
	// traverse the partial tree.
	var bitUsed, hashUsed int
	hashMerkleRoot := pmt.TraverseAndExtract(height, 0, bitUsed, hashUsed, matches, items)
	// verify that no problems occurred during the tree traversal.
	if pmt.bad {
		return &util.Hash{}
	}
	// verify that all bits were consumed (except for the padding caused by
	// serializing it as a byte sequence)
	if (bitUsed+7)/8 != (len(pmt.bits)+7)/8 {
		return &util.Hash{}
	}
	// verify that all hashes were consumed.
	if hashUsed != len(pmt.hashes) {
		return &util.Hash{}
	}
	return hashMerkleRoot
}

func (pmt *PartialMerkleTree) TraverseAndExtract(height uint, pos uint, bitUsed int, hashUsed int,
	matches []util.Hash, items []int) *util.Hash {

	if bitUsed >= len(pmt.bits) {
		// Overflowed the bits array - failure
		pmt.bad = true
		return &util.Hash{}
	}

	parentOfMatch := pmt.bits[bitUsed]
	bitUsed++
	if height == 0 && !parentOfMatch {
		// If at height 0, or nothing interesting below, use stored hash and do
		// not descend.
		if hashUsed >= len(pmt.hashes) {
			// Overflowed the hash array - failure
			pmt.bad = true
			return &util.Hash{}
		}

		hash := pmt.hashes[hashUsed]
		hashUsed++
		// In case of height 0, we have a matched txid.
		if height == 0 && parentOfMatch {
			matches = append(matches, hash)
			items = append(items, int(pos))
		}

		return &hash
	}

	// Otherwise, descend into the subtrees to extract matched txids and
	// hashes.
	left := pmt.TraverseAndExtract(height-1, pos*2, bitUsed, hashUsed, matches, items)

	var right *util.Hash
	if pos*2+1 < pmt.calcTreeWidth(height-1) {
		right = pmt.TraverseAndExtract(height-1, pos*2+1, bitUsed, hashUsed, matches, items)

		if right.IsEqual(left) {
			// The left and right branches should never be identical, as the
			// transaction hashes covered by them must each be unique.
			pmt.bad = true
		}
	} else {
		right = left
	}

	// combine the before returning
	ret := make([]byte, 2*sha256.Size)
	ret = append(ret, left[:]...)
	ret = append(ret, right[:]...)
	b := util.DoubleSha256Bytes(ret)

	var h util.Hash
	copy(h[:], b)
	return &h
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
