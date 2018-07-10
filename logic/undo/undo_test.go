package undo

import (
	"testing"

	"github.com/copernet/copernicus/logic/tx"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chainparams"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	mtx "github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"bytes"
)

func UpdateUTXOSet(blocks *block.Block, undos *undo.BlockUndo, coinMap *utxo.CoinsMap, param *chainparams.BitcoinParams, height int) {

	coinbaseTx := blocks.Txs[0]
	txundo := undo.NewTxUndo()
	tx.UpdateCoins(coinbaseTx, coinMap, txundo, int32(height))

	for i := 1; i < len(blocks.Txs); i++ {
		txs := blocks.Txs[1]

		tmp := undo.NewTxUndo()
		txundo := undos.GetTxundo()
		txundo = append(txundo, tmp)
		undos.SetTxUndo(txundo)
		tx.UpdateCoins(txs, coinMap, undos.GetTxundo()[len(undos.GetTxundo())-1], int32(height))
	}

	coinMap.SetBestBlock(blocks.GetHash())
	coinMap.Flush(blocks.GetHash())
}

func UndoBlock(blocks *block.Block, coinMap *utxo.CoinsMap, undos *undo.BlockUndo, params *chainparams.BitcoinParams, height int) {

	header := block.NewBlockHeader()
	index := blockindex.NewBlockIndex(header)
	index.Height = int32(height)
	ApplyBlockUndo(undos, blocks, coinMap)
}

func HasSpendableCoin(coinMap *utxo.CoinsMap, txid util.Hash) bool {
	return !coinMap.AccessCoin(outpoint.NewOutPoint(txid, 0)).IsSpent()
}

func TestMain(m *testing.M) {
	config := utxo.UtxoConfig{Do: &db.DBOption{FilePath: "/tmp/undotest", CacheSize: 10000}}
	utxo.InitUtxoLruTip(&config)
	m.Run()
}

func TestConnectUtxoExtBlock(t *testing.T) {
	chainparam := chainparams.ActiveNetParams
	blocks := block.NewBlock()

	txs := mtx.NewTx(0, 2)

	coinsMap := utxo.NewEmptyCoinsMap()
	//genesis block hash, and set genesis block to utxo
	randomHash := *util.GetRandHash()
	blocks.Header.HashPrevBlock = randomHash
	coinsMap.SetBestBlock(randomHash)
	coinsMap.Flush(randomHash)

	// Create a block with coinbase and resolution transaction.
	Ins := txs.GetIns()
	Ins = make([]*txin.TxIn, 1)
	Outs := txs.GetOuts()
	Outs = make([]*txout.TxOut, 1)

	Ins[0] = txin.NewTxIn(nil, script.NewEmptyScript(), 0000)
	Outs[0] = txout.NewTxOut(42, script.NewEmptyScript())
	txs.GetHash()
	coinbaseTx := txs.Copy()
	coinbaseTx.GetHash()
	blocks.Txs = make([]*mtx.Tx, 2)
	blocks.Txs[0] = coinbaseTx

	Outs[0].SetScriptPubKey(script.NewScriptRaw([]byte{opcodes.OP_TRUE}))
	Ins[0].PreviousOutPoint = outpoint.NewOutPoint(*util.GetRandHash(), 0)
	Ins[0].Sequence = script.SequenceFinal
	Ins[0].SetScriptSig(script.NewScriptRaw([]byte{}))

	prevTx0 := txs.Copy()
	prevTx0.GetHash()
	tx.AddCoins(prevTx0, coinsMap, 100)

	Ins[0].PreviousOutPoint.Hash = prevTx0.GetHash()

	txs.GetHash()
	blocks.Txs[1] = txs

	buf := bytes.NewBuffer(nil)
	blocks.Serialize(buf)
	blocks.GetHash()

	// Now update hte UTXO set
	undos := undo.NewBlockUndo(10)

	UpdateUTXOSet(blocks, undos, coinsMap, chainparam, 123456)

	cvt := utxo.GetUtxoCacheInstance()
	h, err := cvt.GetBestBlock()
	if err != nil {
		panic("get best block failed..")
	}
	if h != blocks.GetHash() {
		t.Error("this block should have been stored in the cache")
	}

	if !HasSpendableCoin(coinsMap, coinbaseTx.GetHash()) {
		t.Error("this coinbase transaction should have been unlocked")
	}
	if !HasSpendableCoin(coinsMap, txs.GetHash()) {
		t.Error("the specified transaction should be spendable")
	}
	if !HasSpendableCoin(coinsMap, prevTx0.GetHash()) {
		t.Error("this transaction should be not spendable")
	}

	UndoBlock(blocks, coinsMap, undos, chainparam, 123456)
	if len(undos.GetTxundo()) != 1 {
		t.Error("block undo information number should be 1, because only one common tx ")
		return
	}

}
