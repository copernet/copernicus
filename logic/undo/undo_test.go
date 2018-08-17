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
	txundos := undo.NewTxUndo()
	tx.TxUpdateCoins(coinbaseTx, coinMap, txundos, int32(height))

	//len(blocks.Txs)=2
	for i := 1; i < len(blocks.Txs); i++ {
		txs := blocks.Txs[1]
		txundo := undos.GetTxundo()
		txundo = append(txundo, txundos)
		undos.SetTxUndo(txundo)
		tx.TxUpdateCoins(txs, coinMap, undos.GetTxundo()[len(undos.GetTxundo())-1], int32(height))
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

func AddCoins(txs *mtx.Tx, coinMap *utxo.CoinsMap, height int32) {
	isCoinbase := txs.IsCoinBase()
	txid := txs.GetHash()
	for idx, out := range txs.GetOuts() {
		op := outpoint.NewOutPoint(txid, uint32(idx))
		coin := utxo.NewCoin(out, height, isCoinbase)
		coinMap.AddCoin(op, coin, false)
	}
}

func HasSpendableCoin(coinMap *utxo.CoinsMap, txid util.Hash) bool {
	//fmt.Println(coinMap.AccessCoin(outpoint.NewOutPoint(txid, 0)))
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
	//hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	//outpoint1 := outpoint.OutPoint{Hash: *hash1, Index: 0}
	//script2 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	//txout2 := txout.NewTxOut(3, script2)
	//coin := utxo.NewCoin(txout2, 123456, false)
	//coinsMap.AddCoin(&outpoint1, coin)
	//c := coinsMap.GetCoin(&outpoint1)
	//spew.Println(c)

	//genesis block hash, and set genesis block to utxo
	randomhash := *util.GetRandHash()
	blocks.Header.HashPrevBlock = randomhash
	coinsMap.SetBestBlock(randomhash)
	coinsMap.Flush(randomhash)

	coinbaseTx := mtx.NewTx(0, 2)
	Ins1 := txin.NewTxIn(nil, script.NewScriptRaw(make([]byte, 10)), 00000000)
	Outs1 := txout.NewTxOut(42, script.NewScriptRaw([]byte{opcodes.OP_2MUL}))
	Outs1.SetScriptPubKey(script.NewScriptRaw([]byte{opcodes.OP_FALSE}))
	Ins1.PreviousOutPoint = outpoint.NewOutPoint(*util.GetRandHash(), 0)
	Ins1.Sequence = script.SequenceFinal
	Ins1.SetScriptSig(script.NewScriptRaw([]byte{opcodes.OP_2DROP}))
	coinbaseTx.AddTxIn(Ins1)
	coinbaseTx.AddTxOut(Outs1)
	coinbaseTx.GetHash()

	blocks.Txs = make([]*mtx.Tx, 2)
	blocks.Txs[0] = coinbaseTx
	spew.Dump("coinbasetx", blocks.Txs[0])

	prevTx0 := mtx.NewEmptyTx()
	Ins2 := txin.NewTxIn(nil, script.NewScriptRaw(make([]byte, 10)), 00000000)
	Outs2 := txout.NewTxOut(42, script.NewScriptRaw([]byte{opcodes.OP_0NOTEQUAL}))
	Outs2.SetScriptPubKey(script.NewScriptRaw([]byte{opcodes.OP_FALSE}))
	Ins2.PreviousOutPoint = outpoint.NewOutPoint(*util.GetRandHash(), 0)
	Ins2.Sequence = script.SequenceFinal
	Ins2.SetScriptSig(script.NewScriptRaw([]byte{opcodes.OP_2DIV}))
	prevTx0.AddTxOut(Outs2)
	prevTx0.AddTxIn(Ins2)
	phash := prevTx0.GetHash()

	AddCoins(prevTx0, coinsMap, 100)

	Ins1.PreviousOutPoint.Hash = phash
	blocks.Txs[1] = prevTx0
	spew.Dump("prevtx0", prevTx0)

	buf := bytes.NewBuffer(nil)
	err := blocks.Serialize(buf)
	if err != nil {
		t.Error("serialize block failed.")
	}
	blocks.GetHash()

	cvt := utxo.GetUtxoCacheInstance()
	h, err := cvt.GetBestBlock()
	if h != randomhash {
		t.Error("the hash value should equal")
	}
	if err != nil {
		t.Error("get best block failed..")
	}

	// Now update hte UTXO set
	undos := undo.NewBlockUndo(1)
	UpdateUTXOSet(blocks, undos, coinsMap, chainparam, 123456)

	if !HasSpendableCoin(coinsMap, coinbaseTx.GetHash()) {
		t.Error("this coinbase transaction should have been unlocked")
	}

	UndoBlock(blocks, coinsMap, undos, chainparam, 123456)
	if len(undos.GetTxundo()) != 1 {
		t.Error("block undo information number should be 1, because only one common tx ")
		return
	}

	if !HasSpendableCoin(coinsMap, prevTx0.GetHash()) {
		t.Error("this transaction should be not spendable")
	}

}
