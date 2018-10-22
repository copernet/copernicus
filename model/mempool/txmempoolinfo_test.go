package mempool

import (
	"bytes"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"math"
	"reflect"
	"testing"
)

func TestTxMempoolInfo(t *testing.T) {
	//create tx
	tx1 := tx.NewTx(0, 0x02)
	tx1.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(util.HashZero, math.MaxUint32), script.NewEmptyScript(), 0xffffffff))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(10*util.COIN), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(10*util.COIN), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})))

	txMemInfo := TxMempoolInfo{
		Tx:   tx1,
		Time: 0,
		FeeRate: util.FeeRate{
			SataoshisPerK: 100,
		},
	}

	buf := bytes.NewBuffer(nil)
	err := txMemInfo.Serialize(buf)
	if err != nil {
		t.Errorf("Serialize err:%s", err.Error())
	}

	var target TxMempoolInfo
	target.Tx = tx.NewEmptyTx()
	err = target.Unserialize(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Errorf("Unserialize err:%s", err.Error())
	}

	if !reflect.DeepEqual(txMemInfo, target) {
		t.Errorf("the target mempool info:%v\n not equal tx mempool info:%v\n", target, txMemInfo)
	}
}
