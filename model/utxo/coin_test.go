package utxo

import (
	"errors"
	"github.com/copernet/copernicus/util/amount"
	"github.com/stretchr/testify/assert"
	"testing"

	"bytes"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/davecgh/go-spew/spew"
	"reflect"
)

func TestCoin(t *testing.T) {
	script1 := script.NewEmptyScript()
	txout1 := txout.NewTxOut(2, script1)
	c := NewFreshCoin(txout1, 10, false)
	gto := c.GetTxOut()
	gh := c.GetHeight()
	ga := c.GetAmount()

	if ga != txout1.GetValue() {
		t.Error("get amount value is error, please check..")
	}

	if gto != *txout1 || gh != 10 || c.isCoinBase {
		t.Error("get value is faild...")
	}

	if c.isCoinBase {
		t.Error("the coin is coinbase , please check coin ")
	}

	exceptScript := c.GetScriptPubKey()

	if !reflect.DeepEqual(exceptScript, script1) {
		t.Error("get script pubkey is not equal script1, please check...")
	}

	if c.DynamicMemoryUsage() > 0 {
		t.Error("DynamicMemoryUsage not need test...")
	}

	if !reflect.DeepEqual(c.DeepCopy(), c) {
		t.Error("after deep copy, the value should equal coin")
	}

	c.Clear()
	if c.GetHeight() == 0 && c.GetAmount() == 0 {
		t.Error("there is one error in clear func...")
	}

	if c.IsMempoolCoin() {
		t.Errorf("isMempoolCoin default should false")
	}

	if c.IsCoinBase() && c.IsMempoolCoin() {
		t.Error("isCoinBase and isMempoolCoin value should false")
	}

	if c.GetScriptPubKey() != nil {
		t.Error("the script pubkey should nil")
	}

	script2 := script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})
	txout2 := txout.NewTxOut(3, script2)
	c2 := NewFreshCoin(txout2, 10, false)

	if c2.GetTxOut() != *txout2 || c2.GetHeight() != 10 {
		t.Error("get coin value is failed, please check..")
	}

	if c2.GetAmount() != txout2.GetValue() {
		t.Error("get amount failed, please check...")
	}

	if c2.isCoinBase {
		t.Error("the tx not is coinbase, please check...")
	}

	if !reflect.DeepEqual(c2.GetScriptPubKey(), script2) {
		t.Error("get script error,the value should equal script2, please check..")
	}

	if !reflect.DeepEqual(c2.DeepCopy(), c2) {
		t.Error("deep copy coin should equal c2")
	}

	c2.Clear()

	if reflect.DeepEqual(c2.DeepCopy(), c2) {
		t.Error("after clear, the value of deep copy coin not equal coin.")
	}

}

func TestCoinSec(t *testing.T) {
	script3 := script.NewScriptRaw([]byte{opcodes.OP_2DROP, opcodes.OP_2MUL})
	txout3 := txout.NewTxOut(4, script3)

	c3 := NewFreshCoin(txout3, 1000, true)
	spew.Dump("the coin  is: %v \n ", c3)

	w := bytes.NewBuffer(nil)
	err := c3.Serialize(w)
	if err != nil {
		t.Errorf("serialized faild:%v", err)
	}

	var target Coin
	err = target.Unserialize(bytes.NewReader(w.Bytes()))
	if err != nil {
		t.Errorf("unserlize failed...%v\n", err)
	}

	if reflect.DeepEqual(c3, target) {
		t.Error("after clear, the value of deep copy coin not equal coin.")
	}
}

func TestMempoolCoin(t *testing.T) {
	scriptM := script.NewEmptyScript()
	txoutM := txout.NewTxOut(1, scriptM)
	coinM := NewMempoolCoin(txoutM)
	if !coinM.IsMempoolCoin() {
		t.Errorf("coinM should be a mempool coin")
	}
}

func TestFreshCoinState(t *testing.T) {
	coin := NewFreshCoin(txout.NewTxOut(amount.Amount(1), script.NewEmptyScript()), 1, true)

	assert.True(t, coin.isCoinBase)
	assert.True(t, coin.fresh)
	assert.False(t, coin.dirty)
}

func Test_do_not_serialize_spent_coin(t *testing.T) {
	scriptPK := script.NewScriptRaw([]byte{opcodes.OP_TRUE})
	coin := NewFreshCoin(txout.NewTxOut(amount.Amount(1), scriptPK), 1, true)
	coin.Clear()

	buf := bytes.NewBuffer(nil)
	err := coin.Serialize(buf)

	assert.Equal(t, errors.New("already spent"), err)
}

type MockWriter struct {
}

func (mw *MockWriter) Write(p []byte) (size int, err error) {
	return 0, errors.New("EOF")
}

func Test_can_return_serialize_failure(t *testing.T) {
	scriptPK := script.NewScriptRaw([]byte{opcodes.OP_TRUE})
	coin := NewFreshCoin(txout.NewTxOut(amount.Amount(1), scriptPK), 1, true)

	writer := &MockWriter{}
	err := coin.Serialize(writer)

	assert.Equal(t, errors.New("EOF"), err)
}

func Test_test_unserialize_failure(t *testing.T) {
	coin := NewFreshEmptyCoin()

	buf := bytes.NewBuffer(nil)
	err := coin.Unserialize(buf)

	assert.Equal(t, errors.New("EOF"), err)
}
