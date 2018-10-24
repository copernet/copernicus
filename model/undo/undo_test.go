package undo_test

import (
	"bytes"
	"errors"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/stretchr/testify/assert"
	"testing"
)

func makeDummyScriptPubKey() *script.Script {
	return script.NewScriptRaw([]byte{
		0x41, // OP_DATA_65
		0x04, 0x1b, 0x0e, 0x8c, 0x25, 0x67, 0xc1, 0x25,
		0x36, 0xaa, 0x13, 0x35, 0x7b, 0x79, 0xa0, 0x73,
		0xdc, 0x44, 0x44, 0xac, 0xb8, 0x3c, 0x4e, 0xc7,
		0xa0, 0xe2, 0xf9, 0x9d, 0xd7, 0x45, 0x75, 0x16,
		0xc5, 0x81, 0x72, 0x42, 0xda, 0x79, 0x69, 0x24,
		0xca, 0x4e, 0x99, 0x94, 0x7d, 0x08, 0x7f, 0xed,
		0xf9, 0xce, 0x46, 0x7c, 0xb9, 0xf7, 0xc6, 0x28,
		0x70, 0x78, 0xf8, 0x01, 0xdf, 0x27, 0x6f, 0xdf,
		0x84, // 65-byte signature
		0xac, // OP_CHECKSIG
	})
}

func Test_basic_undo_model_methods(t *testing.T) {
	crypto.InitSecp256()

	expectedAmount := amount.Amount(1 * util.COIN)
	expectedScriptPK := makeDummyScriptPubKey()
	expectedHeight := int32(12345)

	bkundo1 := newBlockUndo(expectedAmount, expectedScriptPK, expectedHeight)

	buf := bytes.NewBuffer(nil)
	assert.NoError(t, bkundo1.Serialize(buf))
	assert.Equal(t, bkundo1.SerializeSize(), buf.Len())

	bkundo2 := undo.NewBlockUndo(1)
	assert.NoError(t, bkundo2.Unserialize(buf))

	assert.Equal(t, 2, len(bkundo2.GetTxundo()))
	coins := bkundo2.GetTxundo()[0].GetUndoCoins()
	assert.Equal(t, 1, len(coins))

	assert.Equal(t, expectedAmount, coins[0].GetAmount())
	assert.Equal(t, expectedScriptPK, coins[0].GetScriptPubKey())
	assert.Equal(t, expectedHeight, coins[0].GetHeight())
}

func newBlockUndo(amount amount.Amount, scriptPK *script.Script, height int32) *undo.BlockUndo {
	txout := txout.NewTxOut(amount, scriptPK)
	coin := utxo.NewFreshCoin(txout, height, true)
	txundo := undo.NewTxUndo()
	txundo.SetUndoCoins([]*utxo.Coin{coin})

	bkundo := undo.NewBlockUndo(1)
	bkundo.SetTxUndo([]*undo.TxUndo{txundo})
	bkundo.AddTxUndo(txundo)
	return bkundo
}

type MockWriter struct {
	size int
	n    int
}

func (mw *MockWriter) Write(p []byte) (size int, err error) {
	mw.n = mw.n + len(p)
	if mw.n >= mw.size {
		mw.size--
		mw.n = 0
		return 0, errors.New("EOF")
	}

	return len(p), nil
}

func Test_txundo_serialize_failure_case(t *testing.T) {
	crypto.InitSecp256()
	txoutSpent := txout.NewTxOut(amount.Amount(1), makeDummyScriptPubKey())
	coin := utxo.NewFreshCoin(txoutSpent, 100, true)

	txundo := undo.NewTxUndo()
	txundo.SetUndoCoins([]*utxo.Coin{coin})

	bkundo := undo.NewBlockUndo(1)
	bkundo.SetTxUndo([]*undo.TxUndo{txundo})

	buf := bytes.NewBuffer(nil)
	bkundo.Serialize(buf)
	sz := buf.Len()

	writer := &MockWriter{size: sz}

	for i := 0; i < sz; i++ {
		err := bkundo.Serialize(writer)
		assert.Error(t, err)
	}
}

func Test_txundo_serialize(t *testing.T) {
	txoutSpent := txout.NewTxOut(amount.Amount(-1), makeDummyScriptPubKey())
	coin := utxo.NewFreshCoin(txoutSpent, 100, true)

	txundo := undo.NewTxUndo()
	txundo.SetUndoCoins([]*utxo.Coin{coin})

	buf := bytes.NewBuffer(nil)
	err := txundo.Serialize(buf)

	assert.Equal(t, errors.New("already spent"), err)
}

func Test_txundo_unserialize(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	txundo2 := undo.NewTxUndo()
	err := txundo2.Unserialize(buf)
	assert.Equal(t, errors.New("EOF"), err)
}

func Test_unserialize_failure_case(t *testing.T) {
	crypto.InitSecp256()
	expectedAmount := amount.Amount(1 * util.COIN)
	expectedScriptPK := makeDummyScriptPubKey()
	expectedHeight := int32(12345)

	bkundo1 := newBlockUndo(expectedAmount, expectedScriptPK, expectedHeight)

	buf := bytes.NewBuffer(nil)
	assert.NoError(t, bkundo1.Serialize(buf))

	for i := 0; i < buf.Len(); i++ {
		buf := bytes.NewBuffer(nil)
		assert.NoError(t, bkundo1.Serialize(buf))
		buf.Truncate(i)

		blundo2 := undo.NewBlockUndo(1)
		err := blundo2.Unserialize(buf)
		assert.Error(t, err)
	}
}

func Test_too_many_inputs_when_txundo_unserialize_from_buf(t *testing.T) {
	crypto.InitSecp256()

	txout := txout.NewTxOut(amount.Amount(1), makeDummyScriptPubKey())
	coins := make([]*utxo.Coin, 0, undo.MaxInputPerTx+1)

	for i := 0; i < undo.MaxInputPerTx+1; i++ {
		coin := utxo.NewFreshCoin(txout, 100, true)
		coins = append(coins, coin)
	}

	txundo := undo.NewTxUndo()
	txundo.SetUndoCoins(coins)

	bkundo := undo.NewBlockUndo(1)
	bkundo.SetTxUndo([]*undo.TxUndo{txundo})

	buf := bytes.NewBuffer(nil)
	assert.NoError(t, bkundo.Serialize(buf))

	bkundo2 := undo.NewBlockUndo(1)
	assert.Panics(t, func() {
		bkundo2.Unserialize(buf)
	})
}
