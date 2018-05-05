package mining

import (
	"math"
	"testing"

	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/utils"
)

type TestMemPoolEntry struct {
	Fee            utils.Amount
	Time           int64
	Priority       float64
	Height         int
	SpendsCoinbase bool
	SigOpCost      int
	lp             *core.LockPoints
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

func (t *TestMemPoolEntry) SetFee(fee utils.Amount) *TestMemPoolEntry {
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

func (t *TestMemPoolEntry) FromTxToEntry(tx *core.Tx) *mempool.TxEntry {
	lp := core.LockPoints{}
	if t.lp != nil {
		lp = *(t.lp)
	}
	entry := mempool.NewTxentry(tx, int64(t.Fee), t.Time, int(t.Height), lp, int(t.SigOpCost), t.SpendsCoinbase)
	return entry
}

func createTx() []*mempool.TxEntry {
	testEntryHelp := NewTestMemPoolEntry()
	tx1 := core.NewTx()
	tx1.Ins = make([]*core.TxIn, 0)
	tx1.Outs = make([]*core.TxOut, 2) // two descendants
	tx1.Outs[0] = core.NewTxOut(10*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	tx1.Outs[1] = core.NewTxOut(10*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	tx1.Hash = tx1.TxHash()
	txEntry1 := testEntryHelp.SetTime(1).SetFee(utils.Amount(2 * utils.COIN)).FromTxToEntry(tx1)

	tx2 := core.NewTx()
	tx2.Ins = make([]*core.TxIn, 1)
	tx2.Outs = make([]*core.TxOut, 1)
	tx2.Outs[0] = core.NewTxOut(5*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	// reference relation(tx2 -> tx1)
	tx2.Ins[0] = core.NewTxIn(core.NewOutPoint(tx1.Hash, 0), []byte{core.OP_11, core.OP_EQUAL})
	tx2.Hash = tx2.TxHash()
	txEntry2 := testEntryHelp.SetTime(1).SetFee(utils.Amount(5 * utils.COIN)).FromTxToEntry(tx2)

	//  modify tx3's content to avoid to get the same hash with tx2
	tx3 := core.NewTx()
	tx3.Ins = make([]*core.TxIn, 1)
	tx3.Outs = make([]*core.TxOut, 1)
	tx3.Outs[0] = core.NewTxOut(6*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	// reference relation(tx3 -> tx1)
	tx3.Ins[0] = core.NewTxIn(core.NewOutPoint(tx1.Hash, 1), []byte{core.OP_11, core.OP_EQUAL})
	tx3.Hash = tx3.TxHash()
	txEntry3 := testEntryHelp.SetTime(1).SetFee(utils.Amount(4 * utils.COIN)).FromTxToEntry(tx3)

	tx4 := core.NewTx()
	tx4.Ins = make([]*core.TxIn, 1)
	tx4.Outs = make([]*core.TxOut, 1)
	tx4.Outs[0] = core.NewTxOut(4*utils.COIN, []byte{core.OP_11, core.OP_EQUAL})
	// reference relation(tx4 -> tx3 -> tx1)
	tx4.Ins[0] = core.NewTxIn(core.NewOutPoint(tx3.Hash, 0), []byte{core.OP_11, core.OP_EQUAL})
	tx4.Hash = tx4.TxHash()
	txEntry4 := testEntryHelp.SetTime(1).SetFee(utils.Amount(2 * utils.COIN)).FromTxToEntry(tx4)

	t := make([]*mempool.TxEntry, 4)
	t[0] = txEntry1
	t[1] = txEntry2
	t[2] = txEntry3
	t[3] = txEntry4
	return t
}

func TestCreateNewBlockByFee(t *testing.T) {
	// clear mempool data
	blockchain.GMemPool = mempool.NewTxMempool()
	pool := blockchain.GMemPool

	txSet := createTx()
	noLimit := uint64(math.MaxUint64)
	for _, entry := range txSet {
		pool.AddTx(entry, noLimit, noLimit, noLimit, noLimit, true)
	}
	if pool.Size() != 4 {
		t.Error("add txEntry to mempool error")
	}

	ba := NewBlockAssembler(msg.ActiveNetParams)
	strategy = sortByFee
	ba.CreateNewBlock()

	if len(ba.bt.Block.Txs) != 5 {
		t.Error("some transactions are inserted to block error")
	}

	if ba.bt.Block.Txs[4].Hash != txSet[1].Tx.Hash {
		t.Error("error sort by tx fee")
	}
}

//func TestCreateNewBlockByFeeRate(t *testing.T) {
//	// clear mempool data
//	pool := blockchain.GMemPool
//	pool.PoolData = make(map[utils.Hash]*mempool.TxEntry)
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

