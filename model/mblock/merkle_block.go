package mblock

import (
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
		txid := transaction.TxHash()
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

func (mb *MerkleBlock) Serialize() []byte { // todo
	return nil
}

func (mb *MerkleBlock) Unserialize() *MerkleBlock { // todo
	return &MerkleBlock{}
}
