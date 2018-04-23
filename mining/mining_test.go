package mining

import "testing"

func TestGetSubVersionEB(t *testing.T) {
	str1 := GetSubVersionEB(0)
	str2 := GetSubVersionEB(2e6)
	str3 := GetSubVersionEB(2e7)
	str4 := GetSubVersionEB(2e8)

	if str1 != "0.0" {
		t.Error("convert error when value equal to zero")
	}

	if str2 != "0.2" {
		t.Error("convert error when value less than 1")
	}

	if str3 != "2.0" {
		t.Error("convert error when value between 1 and 10")
	}

	if str4 != "20.0" {
		t.Error("convert error when value more than 10")
	}
}

//
//var blockMinFeeRate = utils.NewFeeRate(int64(policy.DefaultBlockMinTxFee))
//
//type blockInfo struct {
//	extranonce uint8
//	nonce      uint
//}
//
//var binfo = []blockInfo{
//	{4, 0xa4a3e223}, {2, 0x15c32f9e}, {1, 0x0375b547}, {1, 0x7004a8a5},
//	{2, 0xce440296}, {2, 0x52cfe198}, {1, 0x77a72cd0}, {2, 0xbb5d6f84},
//	{2, 0x83f30c2c}, {1, 0x48a73d5b}, {1, 0xef7dcd01}, {2, 0x6809c6c4},
//	{2, 0x0883ab3c}, {1, 0x087bbbe2}, {2, 0x2104a814}, {2, 0xdffb6daa},
//	{1, 0xee8a0a08}, {2, 0xba4237c1}, {1, 0xa70349dc}, {1, 0x344722bb},
//	{3, 0xd6294733}, {2, 0xec9f5c94}, {2, 0xca2fbc28}, {1, 0x6ba4f406},
//	{2, 0x015d4532}, {1, 0x6e119b7c}, {2, 0x43e8f314}, {2, 0x27962f38},
//	{2, 0xb571b51b}, {2, 0xb36bee23}, {2, 0xd17924a8}, {2, 0x6bc212d9},
//	{1, 0x630d4948}, {2, 0x9a4c4ebb}, {2, 0x554be537}, {1, 0xd63ddfc7},
//	{2, 0xa10acc11}, {1, 0x759a8363}, {2, 0xfb73090d}, {1, 0xe82c6a34},
//	{1, 0xe33e92d7}, {3, 0x658ef5cb}, {2, 0xba32ff22}, {5, 0x0227a10c},
//	{1, 0xa9a70155}, {5, 0xd096d809}, {1, 0x37176174}, {1, 0x830b8d0f},
//	{1, 0xc6e3910e}, {2, 0x823f3ca8}, {1, 0x99850849}, {1, 0x7521fb81},
//	{1, 0xaacaabab}, {1, 0xd645a2eb}, {5, 0x7aea1781}, {5, 0x9d6e4b78},
//	{1, 0x4ce90fd8}, {1, 0xabdc832d}, {6, 0x4a34f32a}, {2, 0xf2524c1c},
//	{2, 0x1bbeb08a}, {1, 0xad47f480}, {1, 0x9f026aeb}, {1, 0x15a95049},
//	{2, 0xd1cb95b2}, {2, 0xf84bbda5}, {1, 0x0fa62cd1}, {1, 0xe05f9169},
//	{1, 0x78d194a9}, {5, 0x3e38147b}, {5, 0x737ba0d4}, {1, 0x63378e10},
//	{1, 0x6d5f91cf}, {2, 0x88612eb8}, {2, 0xe9639484}, {1, 0xb7fabc9d},
//	{2, 0x19b01592}, {1, 0x5a90dd31}, {2, 0x5bd7e028}, {2, 0x94d00323},
//	{1, 0xa9b9c01a}, {1, 0x3a40de61}, {1, 0x56e7eec7}, {5, 0x859f7ef6},
//	{1, 0xfd8e5630}, {1, 0x2b0c9f7f}, {1, 0xba700e26}, {1, 0x7170a408},
//	{1, 0x70de86a8}, {1, 0x74d64cd5}, {1, 0x49e738a1}, {2, 0x6910b602},
//	{0, 0x643c565f}, {1, 0x54264b3f}, {2, 0x97ea6396}, {2, 0x55174459},
//	{2, 0x03e8779a}, {1, 0x98f34d8f}, {1, 0xc07b2b07}, {1, 0xdfe29668},
//	{1, 0x3141c7c1}, {1, 0xb3b595f4}, {1, 0x735abf08}, {5, 0x623bfbce},
//	{2, 0xd351e722}, {1, 0xf4ca48c9}, {1, 0x5b19c670}, {1, 0xa164bf0e},
//	{2, 0xbbbeb305}, {2, 0xfe1c810a},
//}
//
//func CreateBlockIndex(height int) *core.BlockIndex {
//	return &core.BlockIndex{
//		Height: height,
//		Prev:   blockchain.GChainActive.Tip(),
//	}
//}
//
//func testSequenceLocks(tx *core.Tx, flags int) bool {
//	blockchain.GMemPool.Lock()
//	defer blockchain.GMemPool.Unlock()
//	return blockchain.CheckSequenceLocks(tx, flags, nil, false)
//}
//
//type testMempoolEntryHelper struct {
//	fee            utils.Amount
//	timestamp      int64
//	height         uint
//	spendsCoinbase bool
//	sigOpCost      uint
//	lp             core.LockPoints
//}
//
//func newTestMempoolEntryHelper() *testMempoolEntryHelper {
//	return &testMempoolEntryHelper{
//		height:    1,
//		sigOpCost: 4,
//	}
//}
//
//// Change the default value
//func (t *testMempoolEntryHelper) setFee(fee utils.Amount) *testMempoolEntryHelper {
//	t.fee = fee
//	return t
//}
//
//func (t *testMempoolEntryHelper) setHeight(height uint) *testMempoolEntryHelper {
//	t.height = height
//	return t
//}
//
//func (t *testMempoolEntryHelper) setTime(timestamp int64) *testMempoolEntryHelper {
//	t.timestamp = timestamp
//	return t
//}
//
//func (t *testMempoolEntryHelper) setSpendCoinbase(flag bool) *testMempoolEntryHelper {
//	t.spendsCoinbase = flag
//	return t
//}
//
//func (t *testMempoolEntryHelper) setSigOpsCost(sigopsCost uint) *testMempoolEntryHelper {
//	t.sigOpCost = sigopsCost
//	return t
//}
//
//// todo TxMempoolEntry discarded
//func (t *testMempoolEntryHelper) FromTx(tx *core.Tx, mpool *mempool.TxMempool) mempool.TxMempoolEntry {
//	// Hack to assume either it's completely dependent on other mempool txs or
//	// not at all.
//	var inChainValue utils.Amount
//	if mpool != nil && mpool.HasNoInputsOf(tx) {
//		inChainValue = utils.Amount(tx.GetValueOut())
//	}
//	return mempool.TxMempoolEntry{
//		TxRef:             tx,
//		Fee:               t.fee,
//		Time:              t.timestamp,
//		EntryHeight:       t.height,
//		SpendsCoinbase:    t.spendsCoinbase,
//		SigOpCount:        int64(t.sigOpCost),
//		LockPoints:        &t.lp,
//		InChainInputValue: inChainValue,
//	}
//}
//
//// Test suite for ancestor feerate transaction selection.
//// Implemented as an additional function, rather than a separate test case, to
//// allow reusing the blockchain created in CreateNewBlock_validity.
//// Note that this test assumes blockprioritypercentage is 0.
//func testPackageSelection(params *msg.BitcoinParams, pubScript core.Script, txFirst []*core.Tx) {
//	// Test the ancestor feerate transaction selection.
//	entry := newTestMempoolEntryHelper()
//	// Test that a medium fee transaction will be selected after a higher fee
//	// rate package with a low fee rate parent.
//	tx := core.NewTx()
//	tx.Ins = make([]*core.TxIn, 1)
//	sig := core.Script{}
//	sig.PushOpCode(core.OP_1)
//	tx.Ins[0].Script = &sig
//	tx.Ins[0].PreviousOutPoint.Hash = txFirst[0].TxHash()
//	tx.Ins[0].PreviousOutPoint.Index = 0
//
//	tx.Outs = make([]*core.TxOut, 1)
//	tx.Outs[0].Value = 5000000000 - 1000
//
//	// This tx has a low fee: 1000 satoshis.
//	// Save this txid for later use.
//	hashParentTx := tx.TxHash()
//	blockchain.GMemPool.
//}
