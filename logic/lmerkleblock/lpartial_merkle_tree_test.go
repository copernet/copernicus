package lmerkleblock

import (
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/fatih/set.v0"
	"math"
	"testing"
)

func TestPartialMerkleTree(t *testing.T) {

	txCount := 100
	txMatch := 47
	match := make([]bool, 0, txCount)
	hashes := make([]util.Hash, 0, txCount)
	setTxIds := set.New()

	for i := 0; i < txCount; i++ {
		lockTime := uint32(0)
		transaction := tx.NewTx(lockTime, tx.DefaultVersion)
		preOut := outpoint.NewOutPoint(*util.GetRandHash(), 0)
		newScript := script.NewEmptyScript()
		txIn := txin.NewTxIn(preOut, newScript, math.MaxUint32-1)
		transaction.AddTxIn(txIn)

		pubKey := script.NewEmptyScript()
		pubKey.PushOpCode(opcodes.OP_TRUE)
		txOut := txout.NewTxOut(10, pubKey)
		transaction.AddTxOut(txOut)

		txid := transaction.GetHash()

		if i < txMatch {
			setTxIds.Add(txid)
		}

		if setTxIds.Has(txid) {
			match = append(match, true)
		} else {
			match = append(match, false)
		}

		hashes = append(hashes, txid)

	}

	partial := NewPartialMerkleTree(hashes, match)
	matches := make([]util.Hash, 0)
	items := make([]int, 0)
	ret := partial.ExtractMatches(&matches, &items)
	assert.NotEqual(t, util.Hash{}, ret)
}
