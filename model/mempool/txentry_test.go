package mempool

import (
	"math"
	"testing"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util/amount"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/stretchr/testify/assert"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/block"
)

func TestTxentry(t *testing.T) {
	//create tx
	tx1 := tx.NewTx(0, 0x02)
	tx1.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(util.HashZero, math.MaxUint32), script.NewEmptyScript(), 0xffffffff))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(10*util.COIN), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(10*util.COIN), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL})))

	txentry := &TxEntry{
		Tx:         tx1,
		TxSize:     120,
		TxFee:      1000,
		TxHeight:   10000,
		SigOpCount: 1,
		time:       1540177584,
		usageSize:  10,
		lp: struct {
			Height        int32
			Time          int64
			MaxInputBlock *blockindex.BlockIndex
		}{Height: 10, Time: 1540177584, MaxInputBlock: nil},
		spendsCoinbase: true,
	}
	txentry.SumTxSigOpCountWithAncestors = 10

	sigOpCount := txentry.GetSigOpCountWithAncestors()
	assert.Equal(t, sigOpCount, int64(10))

	usageSize := txentry.GetUsageSize()
	assert.Equal(t, usageSize, int64(10))

	txTime := txentry.GetTime()
	assert.Equal(t, txTime, int64(1540177584))

	res := txentry.GetSpendsCoinbase()
	assert.Equal(t, res, true)

	lp := txentry.GetLockPointFromTxEntry()
	assert.Equal(t, lp.Time, int64(1540177584))
	assert.Equal(t, lp.Height, int32(10))

	fee := txentry.GetFeeRate()
	amounts := txentry.TxFee * 1000 / int64(txentry.TxSize)
	assert.Equal(t, fee, &util.FeeRate{SataoshisPerK: amounts})

	txmeminfo := txentry.GetInfo()
	assert.Equal(t, txmeminfo.Tx, txentry.Tx)
	assert.Equal(t, txmeminfo.Time, txentry.time)
	assert.Equal(t, txmeminfo.FeeRate, *txentry.GetFeeRate())

	ok := txentry.CheckLockPointValidity(nil)
	assert.Equal(t, ok, true)

	blkidx := createBlkIdx()
	expectLP := LockPoints{
		Height:        10,
		Time:          1540177584,
		MaxInputBlock: blkidx,
	}
	txentry.SetLockPointFromTxEntry(expectLP)
	wantLP := txentry.GetLockPointFromTxEntry()
	assert.Equal(t, expectLP, wantLP)
}

func createBlkIdx() *blockindex.BlockIndex {
	blkHeader := block.NewBlockHeader()
	blkHeader.Time = uint32(1534822771)
	blkHeader.Version = 536870912
	blkHeader.Bits = 486604799
	preHash := util.HashFromString("00000000000001bcd6b635a1249dfbe76c0d001592a7219a36cd9bbd002c7238")
	merkleRoot := util.HashFromString("7e814211a7de289a490380c0c20353e0fd4e62bf55a05b38e1628e0ea0b4fd3d")
	blkHeader.HashPrevBlock = *preHash
	blkHeader.Nonce = 1391785674
	blkHeader.MerkleRoot = *merkleRoot
	//init block index
	blkidx := blockindex.NewBlockIndex(blkHeader)
	return blkidx
}
