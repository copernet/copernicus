package mblock

import (
	"io"

	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/util"
	"gopkg.in/fatih/set.v0"
)

type Matched struct {
	N int
	H util.Hash
}

type MerkleBlock struct {
	Header     block.BlockHeader
	Txn        *PartialMerkleTree
	MatchedTxn []Matched
}

func NewMerkleBlock(bk *block.Block, txids *set.Set) *MerkleBlock {
	ret := MerkleBlock{}

	ret.Header = bk.Header

	match := make([]bool, 0, len(bk.Txs))
	hashes := make([]util.Hash, 0, len(bk.Txs))

	for _, transaction := range bk.Txs {
		txid := transaction.GetHash()
		if txids.Has(txid) {
			match = append(match, true)
		} else {
			match = append(match, false)
		}

		hashes = append(hashes, txid)
	}
	ret.Txn = NewPartialMerkleTree(hashes, match)

	return &ret
}

func (mb *MerkleBlock) Serialize(w io.Writer) (err error) {
	err = mb.Header.Serialize(w)
	if err != nil {
		return
	}

	err = mb.Txn.Serialize(w)
	if err != nil {
		return
	}

	return
}

func (mb *MerkleBlock) Unserialize(r io.Reader) (err error) {
	err = mb.Header.Unserialize(r)
	if err != nil {
		return
	}

	pmt := PartialMerkleTree{}
	err = pmt.Unserialize(r)
	if err != nil {
		return
	}
	mb.Txn = &pmt

	return
}
