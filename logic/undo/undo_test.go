package undo

import (
	"testing"

	"bytes"
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
	"github.com/davecgh/go-spew/spew"
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

	coinsMap := utxo.NewEmptyCoinsMap()
	//genesis block hash, and set genesis block to utxo
	randomHash := *util.GetRandHash()
	blocks.Header.HashPrevBlock = randomHash
	coinsMap.SetBestBlock(randomHash)
	coinsMap.Flush(randomHash)

	coinbaseTx := mtx.NewTx(0, 2)
	Ins1 := coinbaseTx.GetIns()
	Ins1 = make([]*txin.TxIn, 1)
	Outs1 := coinbaseTx.GetOuts()
	Outs1 = make([]*txout.TxOut, 1)
	Ins1[0] = txin.NewTxIn(nil, script.NewScriptRaw(make([]byte, 10)), 00000000)
	Outs1[0] = txout.NewTxOut(42, script.NewScriptRaw([]byte{}))
	coinbaseTx.GetHash()
	blocks.Txs = make([]*mtx.Tx, 2)
	blocks.Txs[0] = coinbaseTx
	Outs1[0].SetScriptPubKey(script.NewScriptRaw([]byte{opcodes.OP_FALSE}))
	Ins1[0].PreviousOutPoint = outpoint.NewOutPoint(*util.GetRandHash(), 0)
	Ins1[0].Sequence = script.SequenceFinal
	Ins1[0].SetScriptSig(script.NewScriptRaw([]byte{}))

	prevTx0 := mtx.NewEmptyTx()
	Ins2 := prevTx0.GetIns()
	Ins2 = make([]*txin.TxIn, 1)
	Outs2 := prevTx0.GetOuts()
	Outs2 = make([]*txout.TxOut, 1)
	Ins2[0] = txin.NewTxIn(nil, script.NewScriptRaw(make([]byte, 10)), 00000000)
	Outs2[0] = txout.NewTxOut(42, script.NewScriptRaw([]byte{}))
	prevTx0.GetHash()

	//tx.AddCoins(prevTx0, coinsMap, 100)
	//////////////////////////////////// todo

	Ins1[0].PreviousOutPoint.Hash = prevTx0.GetHash()
	prevTx0.GetHash()
	blocks.Txs[1] = prevTx0

	buf := bytes.NewBuffer(nil)
	blocks.Serialize(buf)
	blocks.GetHash()

	// Now update hte UTXO set
	undos := undo.NewBlockUndo(1)

	UpdateUTXOSet(blocks, undos, coinsMap, chainparam, 123456)

	cvt := utxo.GetUtxoCacheInstance()
	h, err := cvt.GetBestBlock()
	if err != nil {
		panic("get best block failed..")
	}
	if h != blocks.GetHash() {
		t.Error("this block should have been stored in the cache")
	}

	spew.Dump("coinbaseTx.GetHash()", coinbaseTx.GetHash())
	if !HasSpendableCoin(coinsMap, coinbaseTx.GetHash()) {
		t.Error("this coinbase transaction should have been unlocked")
	}

	spew.Dump("prevTx0.GetHash()", prevTx0.GetHash())
	if !HasSpendableCoin(coinsMap, prevTx0.GetHash()) {
		t.Error("this transaction should be not spendable")
	}

	UndoBlock(blocks, coinsMap, undos, chainparam, 123456)
	if len(undos.GetTxundo()) != 1 {
		t.Error("block undo information number should be 1, because only one common tx ")
		return
	}
}
