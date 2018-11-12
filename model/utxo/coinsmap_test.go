package utxo

import (
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/stretchr/testify/assert"
	"math"
	"reflect"
	"testing"
)

func getTestCoin() (ropt *outpoint.OutPoint, coin *Coin) {
	hash1 := util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")
	ropt = &outpoint.OutPoint{Hash: *hash1, Index: 0}

	coinScript := script.NewScriptRaw([]byte{opcodes.OP_TRUE})
	txout2 := txout.NewTxOut(3, coinScript)

	rcoin := &Coin{
		txOut:         *txout2,
		height:        10000,
		isCoinBase:    false,
		isMempoolCoin: false,
		dirty:         false,
		fresh:         true,
	}

	return ropt, rcoin

}

func TestCoinsMap(t *testing.T) {
	necm := NewEmptyCoinsMap()

	outpoint1, coin1 := getTestCoin()

	coin1Spent := coin1.DeepCopy()
	coin1Spent.Clear()

	necm.AddCoin(outpoint1, coin1, false)

	c := necm.GetCoin(outpoint1)
	assert.NotNil(t, c, "the coin has been added is nil")

	assert.True(t, reflect.DeepEqual(c, coin1), "get coin failed")

	cByAccess := necm.AccessCoin(outpoint1)
	assert.True(t, reflect.DeepEqual(c, cByAccess))

	outpointNotExist := outpoint.OutPoint{Hash: outpoint1.Hash, Index: 1}
	emptyCoin := necm.AccessCoin(&outpointNotExist)
	assert.True(t, reflect.DeepEqual(emptyCoin, NewEmptyCoin()))

	coinsMap := necm.GetMap()
	assert.False(t, len(coinsMap) == 0)

	cc := necm.FetchCoin(outpoint1)
	assert.NotNil(t, cc)

	assert.True(t, reflect.DeepEqual(cc, coin1))

	// SpendCoin just delete it from map, the content of coin1 is not modified
	assert.True(t, reflect.DeepEqual(necm.SpendCoin(outpoint1), coin1))

	ccc := necm.GetCoin(outpoint1)
	assert.Nil(t, ccc, "get the coin has been spend should nil")

	necm.AddCoin(outpoint1, coin1, false)
	assert.True(t, reflect.DeepEqual(necm.SpendGlobalCoin(outpoint1), coin1))

	necm.AddCoin(outpoint1, coin1, false)
	necm.UnCache(outpoint1)
	ncc := necm.GetCoin(outpoint1)
	assert.Nil(t, ncc, "query uncached coin should got nil")

	DisplayCoinMap(necm)
}

func TestCoinsMap_DeepCopy(t *testing.T) {
	necm := NewEmptyCoinsMap()
	tpoint, tcoin := getTestCoin()
	necm.AddCoin(tpoint, tcoin, true)

	cnecm := necm.DeepCopy()
	assert.Equal(t, necm, cnecm)
}

func TestCoinsMap_GetValueIn(t *testing.T) {
	necm := NewEmptyCoinsMap()

	pubKey := script.NewEmptyScript()
	err := pubKey.PushOpCode(opcodes.OP_TRUE)
	if err != nil {
		t.Errorf("push opcode error:%v", err)
	}

	tpoint, tcoin := getTestCoin()
	necm.AddCoin(tpoint, tcoin, true)

	tx1 := tx.NewTx(0, 0x02)
	//tx1.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(util.HashZero, math.MaxUint32), script.NewEmptyScript(), 0xffffffff))
	tx1.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(tpoint.Hash, 0), pubKey, math.MaxUint32-1))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(20*util.COIN), pubKey))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(15*util.COIN), pubKey))

	valueIn := necm.GetValueIn(tx1)
	assert.Equal(t, amount.Amount(3), valueIn)

	txCoinBase := tx.NewTx(0, 0x02)
	tx1.AddTxIn(txin.NewTxIn(nil, pubKey, math.MaxUint32-1))

	valueIn = necm.GetValueIn(txCoinBase)
	assert.Equal(t, amount.Amount(0), valueIn)

}

func TestFetchCoinPanic(t *testing.T) {
	necm := NewEmptyCoinsMap()
	tpoint, tcoin := getTestCoin()
	tcoin.fresh = true
	tcoin.dirty = true
	necm.AddCoin(tpoint, tcoin, true)
	tcoin = necm.GetCoin(tpoint)
	tcoin.dirty = true

	tcoin.txOut.SetNull()
	assert.Panics(t, func() { necm.SpendCoin(tpoint) })
}

func TestFetchCoinFreshNotDirty(t *testing.T) {
	necm := NewEmptyCoinsMap()
	tpoint, tcoin := getTestCoin()
	necm.AddCoin(tpoint, tcoin, true)

	tcoin = necm.GetCoin(tpoint)
	tcoin.dirty = true

	tcoin.txOut.SetNull()
	assert.Panics(t, func() { necm.SpendCoin(tpoint) })
}

func TestCoinsMap_FetchCoin(t *testing.T) {
	necm := NewEmptyCoinsMap()
	tpoint, _ := getTestCoin()
	rcoin := necm.FetchCoin(tpoint)
	assert.Nil(t, rcoin)

	ok := necm.Flush(tpoint.Hash)
	assert.True(t, ok)
}

func TestCoinsMap_spend_fresh_added_coin(t *testing.T) {
	cm := NewEmptyCoinsMap()

	opt := outpoint.NewOutPoint(util.HashOne, 0)
	rtxout := txout.NewTxOut(amount.Amount(50), script.NewEmptyScript())
	coin := NewFreshCoin(rtxout, 1, true)
	cm.AddCoin(opt, coin, false)
	DisplayCoinMap(cm)

	_ = cm.SpendCoin(opt)
	assert.Nil(t, cm.GetCoin(opt))
}

func TestCoinsMapGetValueIn(t *testing.T) {
	cm := NewEmptyCoinsMap()

	outpoint1 := outpoint.NewOutPoint(util.HashOne, 0)
	txout1 := txout.NewTxOut(amount.Amount(50), script.NewEmptyScript())
	coin1 := NewFreshCoin(txout1, 1, true)
	cm.AddCoin(outpoint1, coin1, false)

	outpoint2 := outpoint.NewOutPoint(util.HashOne, 1)
	txout2 := txout.NewTxOut(amount.Amount(21), script.NewEmptyScript())
	coin2 := NewFreshCoin(txout2, 1, true)
	cm.AddCoin(outpoint2, coin2, false)

	txn := tx.NewTx(0, 1)
	txn.AddTxIn(txin.NewTxIn(outpoint1, script.NewEmptyScript(), script.SequenceFinal))
	txn.AddTxIn(txin.NewTxIn(outpoint2, script.NewEmptyScript(), script.SequenceFinal))

	assert.Equal(t, amount.Amount(50+21), cm.GetValueIn(txn))
}

func TestCoinsMap_value_in_of_coinbase_tx_should_be_zero(t *testing.T) {
	cm := NewEmptyCoinsMap()

	coinbaseTx := tx.NewTx(0, 1)
	coinbaseTx.AddTxIn(txin.NewTxIn(outpoint.NewDefaultOutPoint(), script.NewEmptyScript(), script.SequenceFinal))

	assert.Equal(t, amount.Amount(0), cm.GetValueIn(coinbaseTx))
}
