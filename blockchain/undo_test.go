package blockchain

import (
	"bytes"
	"testing"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

type CoinsViewTest struct {
	hashBestBlock utils.Hash
}

func newCoinsViewTest() *CoinsViewTest {
	return &CoinsViewTest{}
}

func (coinsViewTest *CoinsViewTest) GetCoin(outPoint *core.OutPoint, coin *utxo.Coin) bool {
	return true
}

func (coinsViewTest *CoinsViewTest) HaveCoin(point *core.OutPoint) bool {
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

func UpdateUTXOSet(block *core.Block, cache *utxo.CoinsViewCache, undo *BlockUndo,
	param *msg.BitcoinParams, height int) {

	coinbaseTx := block.Txs[0]
	cache.UpdateCoins(coinbaseTx, height)

	for i := 1; i < len(block.Txs); i++ {
		tx := block.Txs[1]

		tmp := newTxUndo()
		undo.txundo = append(undo.txundo, tmp)
		tmp.PrevOut = append(tmp.PrevOut, cache.UpdateCoins(tx, height)...)
	}

	cache.SetBestBlock(*block.Hash)

}

func UndoBlock(block *core.Block, cache *utxo.CoinsViewCache, undo *BlockUndo,
	params *msg.BitcoinParams, height int) {

	header := core.NewBlockHeader()
	index := core.NewBlockIndex(header)
	index.Height = height
	ApplyBlockUndo(undo, block, index, cache)
}

func HasSpendableCoin(cache *utxo.CoinsViewCache, txid *utils.Hash) bool {
	return !cache.AccessCoin(core.NewOutPoint(*txid, 0)).IsSpent()
}

func TestConnectUtxoExtBlock(t *testing.T) {
	chainparams := msg.ActiveNetParams
	block := core.NewBlock()

	coinsDummy := newCoinsViewTest()

	cache := utxo.CoinsViewCache{
		CacheCoins: make(utxo.CacheCoins),
	}
	cache.Base = coinsDummy
	//genesis block hash, and set genesis block to utxo
	randomHash := *utils.GetRandHash()
	block.BlockHeader.HashPrevBlock = randomHash
	cache.SetBestBlock(randomHash)

	tx := core.NewTx()
	// Create a block with coinbase and resolution transaction.
	tx.Ins = make([]*core.TxIn, 1)
	tx.Outs = make([]*core.TxOut, 1)

	tx.Ins[0] = core.NewTxIn(nil, make([]byte, 10))
	tx.Outs[0] = core.NewTxOut(42, []byte{})
	tx.Hash = tx.TxHash()
	coinbaseTx := tx.Copy()

	block.Txs = make([]*core.Tx, 2)
	block.Txs[0] = coinbaseTx

	tx.Outs[0].Script = core.NewScriptRaw([]byte{core.OP_TRUE})
	tx.Ins[0].PreviousOutPoint = core.NewOutPoint(*utils.GetRandHash(), 0)
	tx.Ins[0].Sequence = core.SequenceFinal
	tx.Ins[0].Script = core.NewScriptRaw([]byte{})
	tx.Version = 2
	tx.Hash = utils.HashZero
	tx.Hash = tx.TxHash()

	prevTx0 := tx.Copy()
	utxo.AddCoins(cache, *prevTx0, 100)

	tx.Ins[0].PreviousOutPoint.Hash = tx.Hash
	tx.Hash = utils.HashZero
	tx.Hash = tx.TxHash()
	block.Txs[1] = tx

	buf := bytes.NewBuffer(nil)
	block.Serialize(buf)
	*block.Hash = crypto.DoubleSha256Hash(buf.Bytes()[:80])

	// Now update hte UTXO set
	undo := &BlockUndo{
		txundo: make([]*TxUndo, 0),
	}

	UpdateUTXOSet(block, &cache, undo, chainparams, 123456)

	if cache.GetBestBlock() != *block.Hash {
		t.Error("this block should have been stored in the cache")
	}
	if !HasSpendableCoin(&cache, &coinbaseTx.Hash) {
		t.Error("this coinbase transaction should have been unlocked")
	}
	if !HasSpendableCoin(&cache, &tx.Hash) {
		t.Error("the specified transaction should be spendable")
	}
	if HasSpendableCoin(&cache, &prevTx0.Hash) {
		t.Error("this transaction should be not spendable")
	}

	UndoBlock(block, &cache, undo, chainparams, 123456)
	if len(undo.txundo) != 1 {
		t.Error("block undo information number should be 1, because only one common tx ")
		return
	}

	if cache.GetBestBlock() != block.BlockHeader.HashPrevBlock {
		t.Error("this block should have been stored in the cache : ", block.Hash)
	}
	if HasSpendableCoin(&cache, &coinbaseTx.Hash) {
		t.Error("this coinbase transaction should not have been unlocked")
	}
	if HasSpendableCoin(&cache, &tx.Hash) {
		t.Error("the specified transaction should not be spendable")
	}
	if !HasSpendableCoin(&cache, &prevTx0.Hash) {
		t.Error("this transaction should be spendable")
	}
}
