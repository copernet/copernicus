package blockchain

import (
	"bytes"
	"testing"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/msg"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

type CoinsViewTest struct {
	hashBestBlock utils.Hash
}

func newCoinsViewTest() *CoinsViewTest {
	return &CoinsViewTest{}
}

func (coinsViewTest *CoinsViewTest) GetCoin(outPoint *model.OutPoint, coin *utxo.Coin) bool {
	return true
}

func (coinsViewTest *CoinsViewTest) HaveCoin(point *model.OutPoint) bool {
	return true
}

func (coinsViewTest *CoinsViewTest) GetBestBlock() utils.Hash {
	return coinsViewTest.hashBestBlock
}
func (coinsViewTest *CoinsViewTest) EstimateSize() uint64 {
	return 0
}

func (coinsViewTest *CoinsViewTest) BatchWrite(cacheCoins utxo.CacheCoins, hashBlock *utils.Hash) bool {
	return true
}

func UpdateUTXOSet(block *model.Block, cache *utxo.CoinsViewCache, undo *BlockUndo, param *msg.BitcoinParams, height int) {
	coinbaseTx := block.Transactions[0]
	UpdateCoins(coinbaseTx, cache, nil, height)

	for i := 1; i < len(block.Transactions); i++ {
		tx := block.Transactions[1]

		tmp := newTxUndo()
		undo.txundo = append(undo.txundo, tmp)
		UpdateCoins(tx, cache, tmp, height)
	}

	cache.SetBestBlock(block.Hash)

}

func UndoBlock(block *model.Block, cache *utxo.CoinsViewCache, undo *BlockUndo, params *msg.BitcoinParams, height int) {
	header := model.NewBlockHeader()
	index := model.NewBlockIndex(header)
	index.Height = height
	ApplyBlockUndo(undo, block, index, cache)
}

func HasSpendableCoin(cache *utxo.CoinsViewCache, txid *utils.Hash) bool {
	return !cache.AccessCoin(model.NewOutPoint(*txid, 0)).IsSpent()
}

func copyTx(tx *model.Tx) *model.Tx {
	result := *tx
	ins := len(tx.Ins)
	outs := len(tx.Outs)
	result.Ins = make([]*model.TxIn, ins)
	result.Outs = make([]*model.TxOut, outs)
	for i := 0; i < ins; i++ {
		var in *model.TxIn
		if tx.IsCoinBase() {
			in := model.NewTxIn(nil, tx.Ins[i].Script.GetScriptByte())
			in.Sequence = tx.Ins[i].Sequence
			in.SigOpCount = tx.Ins[i].SigOpCount
			result.Ins[i] = in
			continue
		}
		in = model.NewTxIn(model.NewOutPoint(tx.Ins[i].PreviousOutPoint.Hash, tx.Ins[i].PreviousOutPoint.Index), tx.Ins[i].Script.GetScriptByte())
		in.Sequence = tx.Ins[i].Sequence
		in.SigOpCount = tx.Ins[i].SigOpCount
		result.Ins[i] = in
	}

	for j := 0; j < outs; j++ {
		out := model.NewTxOut(tx.Outs[j].Value, tx.Outs[j].Script.GetScriptByte())
		out.SigOpCount = tx.Outs[j].SigOpCount
		result.Outs[j] = out
	}

	return &result
}

func GetID(tx *model.Tx) utils.Hash {
	buf := bytes.NewBuffer(nil)
	tx.Serialize(buf)
	return core.DoubleSha256Hash(buf.Bytes())
}

func TestConnectUtxoExtBlock(t *testing.T) {
	chainparams := msg.ActiveNetParams
	block := model.NewBlock()

	coinsDummy := newCoinsViewTest()

	cache := utxo.CoinsViewCache{
		CacheCoins: make(utxo.CacheCoins),
	}
	cache.Base = coinsDummy

	randomHash := *utils.GetRandHash()
	block.BlockHeader.HashPrevBlock = randomHash
	cache.SetBestBlock(randomHash)

	tx := model.NewTx()
	// Create a block with coinbase and resolution transaction.
	tx.Ins = make([]*model.TxIn, 1)
	tx.Outs = make([]*model.TxOut, 1)

	tx.Ins[0] = model.NewTxIn(nil, bytes.Repeat([]byte{}, 10))
	tx.Outs[0] = model.NewTxOut(42, []byte{})
	tx.Hash = GetID(tx)

	coinbaseTx := copyTx(tx)

	block.Transactions = make([]*model.Tx, 2)
	block.Transactions[0] = copyTx(tx)

	tx.Outs[0].Script = model.NewScriptRaw([]byte{model.OP_TRUE})
	tx.Ins[0].PreviousOutPoint = model.NewOutPoint(*utils.GetRandHash(), 0)
	tx.Ins[0].Sequence = model.SEQUENCE_FINAL
	tx.Ins[0].Script = model.NewScriptRaw([]byte{})
	tx.Version = 2
	tx.Hash = GetID(tx)

	prevTx0 := copyTx(tx)

	utxo.AddCoins(cache, *prevTx0, 100)

	tx.Ins[0].PreviousOutPoint.Hash = prevTx0.Hash

	tx.Hash = GetID(tx)
	tx0 := copyTx(tx)
	block.Transactions[1] = tx0

	buf := bytes.NewBuffer(nil)
	block.Serialize(buf)
	block.Hash = core.DoubleSha256Hash(buf.Bytes()[:80])

	// Now update hte UTXO set
	undo := &BlockUndo{
		txundo: make([]*TxUndo, 0),
	}

	UpdateUTXOSet(block, &cache, undo, chainparams, 123456)

	if cache.GetBestBlock() != block.Hash {
		t.Error("this block should have been stored in the cache")
	}
	if !HasSpendableCoin(&cache, &coinbaseTx.Hash) {
		t.Error("this coinbase transaction should have been unlocked")
	}
	if !HasSpendableCoin(&cache, &tx0.Hash) {
		t.Error("the specified transaction should be spendable")
	}
	if HasSpendableCoin(&cache, &prevTx0.Hash) {
		t.Error("this transaction should be not spendable")
	}

	UndoBlock(block, &cache, undo, chainparams, 123456)

	if cache.GetBestBlock() != block.BlockHeader.HashPrevBlock {
		t.Error("this block should have been stored in the cache")
	}
	if HasSpendableCoin(&cache, &coinbaseTx.Hash) {
		t.Error("this coinbase transaction should not have been unlocked")
	}
	if HasSpendableCoin(&cache, &tx0.Hash) {
		t.Error("the specified transaction should not be spendable")
	}
	if !HasSpendableCoin(&cache, &prevTx0.Hash) {
		t.Error("this transaction should be spendable")
	}
}
