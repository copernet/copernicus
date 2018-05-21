package mining

import (
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/opcodes"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/txin"
	"github.com/btcboost/copernicus/model/txout"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/util/amount"
)

type TestMemPoolEntry struct {
	Fee            amount.Amount
	Time           int64
	Priority       float64
	Height         int
	SpendsCoinbase bool
	SigOpCost      int
	lp             *tx.LockPoints
}

func NewTestMemPoolEntry() *TestMemPoolEntry {
	t := TestMemPoolEntry{}
	t.Fee = 0
	t.Time = 0
	t.Priority = 0.0
	t.Height = 1
	t.SpendsCoinbase = false
	t.SigOpCost = 4
	t.lp = nil
	return &t
}

func (t *TestMemPoolEntry) SetFee(fee amount.Amount) *TestMemPoolEntry {
	t.Fee = fee
	return t
}

func (t *TestMemPoolEntry) SetTime(time int64) *TestMemPoolEntry {
	t.Time = time
	return t
}

func (t *TestMemPoolEntry) SetHeight(height int) *TestMemPoolEntry {
	t.Height = height
	return t
}

func (t *TestMemPoolEntry) SetSpendCoinbase(flag bool) *TestMemPoolEntry {
	t.SpendsCoinbase = flag
	return t
}

func (t *TestMemPoolEntry) SetSigOpsCost(sigOpsCost int) *TestMemPoolEntry {
	t.SigOpCost = sigOpsCost
	return t
}

func (t *TestMemPoolEntry) FromTxToEntry(transaction *tx.Tx) *mempool.TxEntry {
	lp := tx.LockPoints{}
	if t.lp != nil {
		lp = *(t.lp)
	}
	entry := mempool.NewTxentry(transaction, int64(t.Fee), t.Time, int(t.Height), lp, int(t.SigOpCost), t.SpendsCoinbase)
	return entry
}

func createTx() []*mempool.TxEntry {
	testEntryHelp := NewTestMemPoolEntry()
	tx1 := tx.NewTx(0, 0x02)
	tx1.Ins = make([]*txin.TxIn, 0)
	tx1.Outs = make([]*txout.TxOut, 2) // two descendants
	tx1.AddTxIn()
	tx1.Outs[0] = txout.NewTxOut(10*util.COIN, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	tx1.Outs[1] = txout.NewTxOut(10*util.COIN, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	tx1.Hash = tx1.TxHash()
	txEntry1 := testEntryHelp.SetTime(1).SetFee(amount.Amount(2 * util.COIN)).FromTxToEntry(tx1)

	tx2 := tx.NewTx(0, 0x02)
	tx2.AddTxIn()
	tx2.Ins = make([]*txin.TxIn, 1)
	tx2.Outs = make([]*txout.TxOut, 1)
	tx2.Outs[0] = txout.NewTxOut(5*util.COIN, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	// reference relation(tx2 -> tx1)
	tx2.Ins[0] = txin.NewTxIn(outpoint.NewOutPoint(tx1.Hash, 0), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}), 0xffffffff)
	tx2.Hash = tx2.TxHash()
	txEntry2 := testEntryHelp.SetTime(1).SetFee(amount.Amount(5 * util.COIN)).FromTxToEntry(tx2)

	//  modify tx3's content to avoid to get the same hash with tx2
	tx3 := tx.NewTx(0, 0x02)
	tx3.Ins = make([]*txin.TxIn, 1)
	tx3.Outs = make([]*txout.TxOut, 1)
	tx3.Outs[0] = txout.NewTxOut(6*util.COIN, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	// reference relation(tx3 -> tx1)
	tx3.Ins[0] = txin.NewTxIn(outpoint.NewOutPoint(tx1.Hash, 1), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}), 0xffffffff)
	tx3.Hash = tx3.TxHash()
	txEntry3 := testEntryHelp.SetTime(1).SetFee(amount.Amount(4 * util.COIN)).FromTxToEntry(tx3)

	tx4 := tx.NewTx(0, 0x02)
	tx4.Ins = make([]*txin.TxIn, 1)
	tx4.Outs = make([]*txout.TxOut, 1)
	tx4.Outs[0] = txout.NewTxOut(4*util.COIN, script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}))
	// reference relation(tx4 -> tx3 -> tx1)
	tx4.Ins[0] = txin.NewTxIn(outpoint.NewOutPoint(tx3.Hash, 0), script.NewScriptRaw([]byte{opcodes.OP_11, opcodes.OP_EQUAL}), 0xffffffff)
	tx4.Hash = tx4.TxHash()
	txEntry4 := testEntryHelp.SetTime(1).SetFee(amount.Amount(2 * util.COIN)).FromTxToEntry(tx4)

	t := make([]*mempool.TxEntry, 4)
	t[0] = txEntry1
	t[1] = txEntry2
	t[2] = txEntry3
	t[3] = txEntry4
	return t
}

//func TestCreateNewBlockByFee(t *testing.T) {
//	// clear mempool data
//	pool := blockchain.GMemPool
//	pool.PoolData = make(map[util.Hash]*mempool.TxEntry)
//	// clean mempool data
//	defer func() {
//		pool.PoolData = make(map[util.Hash]*mempool.TxEntry)
//	}()
//
//	txSet := createTx()
//	noLimit := uint64(math.MaxUint64)
//	for _, entry := range txSet {
//		pool.AddTx(entry, noLimit, noLimit, noLimit, noLimit, true)
//	}
//	if len(pool.PoolData) != 4 {
//		t.Error("add txEntry to mempool error")
//	}
//
//	ba := NewBlockAssembler(msg.ActiveNetParams)
//	strategy = sortByFee
//	ba.CreateNewBlock()
//
//	if len(ba.bt.Block.Txs) != 5 {
//		t.Error("some transactions are inserted to block error")
//	}
//
//	if ba.bt.Block.Txs[4].Hash != txSet[1].Tx.Hash {
//		t.Error("error sort by tx fee")
//	}
//}

//func TestCreateNewBlockByFeeRate(t *testing.T) {
//	// clear mempool data
//	pool := blockchain.GMemPool
//	pool.PoolData = make(map[util.Hash]*mempool.TxEntry)
//
//	txSet := createTx()
//	noLimit := uint64(math.MaxUint64)
//	for _, entry := range txSet {
//		pool.AddTx(entry, noLimit, noLimit, noLimit, noLimit, true)
//	}
//	if len(pool.PoolData) != 4 {
//		t.Error("add txEntry to mempool error")
//	}
//
//	ba := NewBlockAssembler(msg.ActiveNetParams)
//	strategy = sortByFeeRate
//	ba.CreateNewBlock()
//	if len(ba.bt.Block.Txs) != 5 {
//		t.Error("some transactions are inserted to block error")
//	}
//
//	if ba.bt.Block.Txs[1].Hash != txSet[0].Tx.Hash {
//		t.Error("error sort by tx feerate")
//	}
//
//	if ba.bt.Block.Txs[2].Hash != txSet[1].Tx.Hash {
//		t.Error("error sort by tx feerate")
//	}
//
//	if ba.bt.Block.Txs[3].Hash != txSet[2].Tx.Hash {
//		t.Error("error sort by tx feerate")
//	}
//
//	if ba.bt.Block.Txs[4].Hash != txSet[3].Tx.Hash {
//		t.Error("error sort by tx feerate")
//	}
//}
