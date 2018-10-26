package lundo

import (
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
)

func UpdateUTXOSet(block *block.Block, bkundo *undo.BlockUndo, coinMap *utxo.CoinsMap, height int) {
	ltx.UpdateTxCoins(block.Txs[0], coinMap, nil, int32(height))

	for i := 1; i < len(block.Txs); i++ {
		txn := block.Txs[i]

		txundo := undo.NewTxUndo()
		ltx.UpdateTxCoins(txn, coinMap, txundo, int32(height))

		bkundo.SetTxUndo(append(bkundo.GetTxundo(), txundo))
	}

	blockHash := block.GetHash()
	utxo.GetUtxoCacheInstance().UpdateCoins(coinMap, &blockHash)
	utxo.GetUtxoCacheInstance().Flush()
}

func AddCoins(txs *tx.Tx, coinMap *utxo.CoinsMap, height int32) {
	isCoinbase := txs.IsCoinBase()
	txid := txs.GetHash()
	for idx, out := range txs.GetOuts() {
		op := outpoint.NewOutPoint(txid, uint32(idx))
		coin := utxo.NewFreshCoin(out, height, isCoinbase)
		coinMap.AddCoin(op, coin, false)
	}
}

func HasSpendableCoin(coinMap *utxo.CoinsMap, txid util.Hash) bool {
	coin := utxo.GetUtxoCacheInstance().GetCoin(outpoint.NewOutPoint(txid, 0))
	return coin != nil && !coin.IsSpent()
}

func TestMain(m *testing.M) {
	path, _ := ioutil.TempDir("/tmp", "undotest")
	defer os.RemoveAll(path)

	config := utxo.UtxoConfig{Do: &db.DBOption{FilePath: path, CacheSize: 10000}}
	utxo.InitUtxoLruTip(&config)

	m.Run()
}

func TestBlockUndo__single_tx_case(t *testing.T) {
	block := block.NewBlock()

	coinsMap := utxo.NewEmptyCoinsMap()
	block.Header.HashPrevBlock = updateTipHashInUtxo(coinsMap, t)

	coinbaseTx := makeCoinbaseTx()
	txn1 := makeNormalTx()
	block.Txs = []*tx.Tx{coinbaseTx, txn1}

	// Now update the UTXO set
	undos := undo.NewBlockUndo(2)
	UpdateUTXOSet(block, undos, coinsMap, 100)

	assert.True(t, HasSpendableCoin(coinsMap, coinbaseTx.GetHash()))
	assert.True(t, HasSpendableCoin(coinsMap, txn1.GetHash()))

	ret := ApplyBlockUndo(undos, block, coinsMap)
	blockHash := block.GetHash()
	utxo.GetUtxoCacheInstance().UpdateCoins(coinsMap, &blockHash)
	utxo.GetUtxoCacheInstance().Flush()

	assert.Equal(t, undo.DisconnectOk, ret)
	assert.Equal(t, 1, len(undos.GetTxundo()))
	assert.False(t, HasSpendableCoin(coinsMap, coinbaseTx.GetHash()))
	assert.False(t, HasSpendableCoin(coinsMap, txn1.GetHash()), "already undo the block, so no coins should exists.")
}

func TestBlockUndo__should_fail__when_txundo_count_not_equal_to__txns_count_in_block(t *testing.T) {
	block := block.NewBlock()

	coinsMap := utxo.NewEmptyCoinsMap()
	block.Header.HashPrevBlock = updateTipHashInUtxo(coinsMap, t)

	block.Txs = []*tx.Tx{makeCoinbaseTx(), makeNormalTx()}

	ret := ApplyBlockUndo(undo.NewBlockUndo(0), block, coinsMap)

	assert.Equal(t, undo.DisconnectFailed, ret)
}

func TestBlockUndo__should_fail__when_coins_count_in_txundo___not_equal_to__txns_ins_count(t *testing.T) {
	block := block.NewBlock()

	coinsMap := utxo.NewEmptyCoinsMap()

	block.Txs = []*tx.Tx{makeCoinbaseTx(), makeNormalTx()}

	blockUndo := undo.NewBlockUndo(0)
	txundo := undo.NewTxUndo()
	txundo.SetUndoCoins(make([]*utxo.Coin, 100))
	blockUndo.AddTxUndo(txundo)

	ret := ApplyBlockUndo(blockUndo, block, coinsMap)

	assert.Equal(t, undo.DisconnectFailed, ret)
}

func TestBlockUndo__2_normal_tx_and_1_unspendable_opreturn_tx(t *testing.T) {
	block := block.NewBlock()

	coinsMap := utxo.NewEmptyCoinsMap()
	block.Header.HashPrevBlock = updateTipHashInUtxo(coinsMap, t)

	coinbaseTx := makeCoinbaseTx()
	txn1 := makeNormalTx()
	txn2 := makeUnspendableTx()
	txn3 := makeNormalTx()
	AddCoins(txn1, coinsMap, 100)
	AddCoins(txn3, coinsMap, 100)
	block.Txs = []*tx.Tx{coinbaseTx, txn1, txn2, txn3}

	// Now update the UTXO set
	undos := undo.NewBlockUndo(2)
	UpdateUTXOSet(block, undos, coinsMap, 100)

	assert.True(t, HasSpendableCoin(coinsMap, coinbaseTx.GetHash()))
	assert.True(t, HasSpendableCoin(coinsMap, txn1.GetHash()))
	assert.False(t, HasSpendableCoin(coinsMap, txn2.GetHash()))
	assert.True(t, HasSpendableCoin(coinsMap, txn3.GetHash()))

	ret := ApplyBlockUndo(undos, block, coinsMap)
	blockHash := block.GetHash()
	utxo.GetUtxoCacheInstance().UpdateCoins(coinsMap, &blockHash)
	utxo.GetUtxoCacheInstance().Flush()

	assert.Equal(t, undo.DisconnectOk, ret)
	assert.Equal(t, 3, len(undos.GetTxundo()))
	assert.False(t, HasSpendableCoin(coinsMap, coinbaseTx.GetHash()))
	assert.False(t, HasSpendableCoin(coinsMap, txn1.GetHash()), "already undo the block, so no coins should exists.")
	assert.False(t, HasSpendableCoin(coinsMap, txn2.GetHash()), "already undo the block, so no coins should exists.")
	assert.False(t, HasSpendableCoin(coinsMap, txn3.GetHash()), "already undo the block, so no coins should exists.")
}

func makeNormalTx() *tx.Tx {
	txn := tx.NewEmptyTx()
	Ins2 := txin.NewTxIn(outpoint.NewOutPoint(*util.GetRandHash(), 0), script.NewEmptyScript(), script.SequenceFinal)
	txn.AddTxIn(Ins2)
	txn.AddTxOut(txout.NewTxOut(42, script.NewEmptyScript()))
	return txn
}

func makeUnspendableTx() *tx.Tx {
	txn := tx.NewEmptyTx()
	Ins2 := txin.NewTxIn(outpoint.NewOutPoint(*util.GetRandHash(), 0), script.NewEmptyScript(), script.SequenceFinal)
	txn.AddTxIn(Ins2)
	txn.AddTxOut(txout.NewTxOut(42, script.NewScriptRaw([]byte{opcodes.OP_RETURN})))
	return txn
}

func makeCoinbaseTx() *tx.Tx {
	coinbaseTx := tx.NewTx(0, 2)
	Ins1 := txin.NewTxIn(outpoint.NewOutPoint(util.HashZero, math.MaxUint32), script.NewEmptyScript(), script.SequenceFinal)
	coinbaseTx.AddTxIn(Ins1)
	coinbaseTx.AddTxOut(txout.NewTxOut(42, script.NewEmptyScript()))
	return coinbaseTx
}

func updateTipHashInUtxo(coinsMap *utxo.CoinsMap, t *testing.T) util.Hash {
	//genesis block hash, and set genesis block to utxo
	genesisHash := *util.GetRandHash()
	utxo.GetUtxoCacheInstance().UpdateCoins(coinsMap, &genesisHash)
	h, _ := utxo.GetUtxoCacheInstance().GetBestBlock()
	assert.Equal(t, genesisHash, h)
	return genesisHash
}
