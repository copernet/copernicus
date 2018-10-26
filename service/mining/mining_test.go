package mining

import (
	"errors"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/logic/lmerkleroot"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/service"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"testing"
)

func initTestEnv(t *testing.T) (dirpath string, err error) {
	args := []string{"--regtest"}
	conf.Cfg = conf.InitConfig(args)

	unitTestDataDirPath, err := conf.SetUnitTestDataDir(conf.Cfg)
	t.Logf("test in temp dir: %s", unitTestDataDirPath)
	if err != nil {
		return "", err
	}

	if conf.Cfg.P2PNet.TestNet {
		model.SetTestNetParams()
	} else if conf.Cfg.P2PNet.RegTest {
		model.SetRegTestParams()
	}

	persist.InitPersistGlobal()

	// Init UTXO DB
	utxoDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/chainstate",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
	utxo.InitUtxoLruTip(&utxoConfig)

	// Init blocktree DB
	blkDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	chain.InitGlobalChain()

	// Load blockindex DB
	lblockindex.LoadBlockIndexDB()

	lchain.InitGenesisChain()

	mempool.InitMempool()

	crypto.InitSecp256()

	ltx.ScriptVerifyInit()

	return unitTestDataDirPath, nil
}

const nInnerLoopCount = 0x100000

func generateBlocks(scriptPubKey *script.Script, generate int, maxTries uint64) (interface{}, error) {
	heightStart := chain.GetInstance().Height()
	heightEnd := heightStart + int32(generate)
	height := heightStart
	params := model.ActiveNetParams

	ret := make([]string, 0)
	var extraNonce uint
	for height < heightEnd {
		ba := NewBlockAssembler(params)
		bt := ba.CreateNewBlock(scriptPubKey, CoinbaseScriptSig(extraNonce))
		if bt == nil {
			return nil, errors.New("create block error")
		}

		bt.Block.Header.MerkleRoot = lmerkleroot.BlockMerkleRoot(bt.Block.Txs, nil)

		powCheck := pow.Pow{}
		bits := bt.Block.Header.Bits
		for maxTries > 0 && bt.Block.Header.Nonce < nInnerLoopCount {
			maxTries--
			bt.Block.Header.Nonce++
			hash := bt.Block.GetHash()
			if powCheck.CheckProofOfWork(&hash, bits, params) {
				break
			}
		}

		if maxTries == 0 {
			break
		}

		if bt.Block.Header.Nonce == nInnerLoopCount {
			extraNonce++
			continue
		}

		fNewBlock := false
		if service.ProcessNewBlock(bt.Block, true, &fNewBlock) != nil {
			return nil, errors.New("ProcessNewBlock, block not accepted")
		}

		height++
		extraNonce = 0

		blkHash := bt.Block.GetHash()
		ret = append(ret, blkHash.String())
	}

	return ret, nil
}

type TestMemPoolEntry struct {
	Fee            amount.Amount
	Time           int64
	Priority       float64
	Height         int32
	SpendsCoinbase bool
	SigOpCost      int
	lp             *mempool.LockPoints
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

func (t *TestMemPoolEntry) SetHeight(height int32) *TestMemPoolEntry {
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
	lp := mempool.LockPoints{}
	if t.lp != nil {
		lp = *(t.lp)
	}
	entry := mempool.NewTxentry(transaction, int64(t.Fee), t.Time, t.Height, lp, int(t.SigOpCost), t.SpendsCoinbase)
	return entry
}

func createTx(tt *testing.T, baseTx *tx.Tx, pubKey *script.Script) []*mempool.TxEntry {

	testEntryHelp := NewTestMemPoolEntry()
	tx1 := tx.NewTx(0, 0x02)
	//tx1.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(util.HashZero, math.MaxUint32), script.NewEmptyScript(), 0xffffffff))
	tx1.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(baseTx.GetHash(), 0), pubKey, math.MaxUint32-1))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(20*util.COIN), pubKey))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(15*util.COIN), pubKey))
	txEntry1 := testEntryHelp.SetTime(1).SetFee(amount.Amount(15 * util.COIN)).FromTxToEntry(tx1)

	tx2 := tx.NewTx(0, 0x02)
	// reference relation(tx2 -> tx1)
	tx2.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(tx1.GetHash(), 0), pubKey, math.MaxUint32-1))
	tx2.AddTxOut(txout.NewTxOut(amount.Amount(12*util.COIN), pubKey))
	txEntry2 := testEntryHelp.SetTime(1).SetFee(amount.Amount(8 * util.COIN)).FromTxToEntry(tx2)
	txEntry2.ParentTx[txEntry1] = struct{}{}

	//  modify tx3's content to avoid to get the same hash with tx2
	tx3 := tx.NewTx(0, 0x02)
	// reference relation(tx3 -> tx1)
	tx3.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(tx1.GetHash(), 1), pubKey, math.MaxUint32-1))
	tx3.AddTxOut(txout.NewTxOut(amount.Amount(9*util.COIN), pubKey))
	txEntry3 := testEntryHelp.SetTime(1).SetFee(amount.Amount(6 * util.COIN)).FromTxToEntry(tx3)
	txEntry3.ParentTx[txEntry1] = struct{}{}

	tx4 := tx.NewTx(0, 0x02)
	// reference relation(tx4 -> tx3 -> tx1)
	tx4.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(tx3.GetHash(), 0), pubKey, math.MaxUint32-1))
	tx4.AddTxOut(txout.NewTxOut(amount.Amount(6*util.COIN), pubKey))
	txEntry4 := testEntryHelp.SetTime(1).SetFee(amount.Amount(3 * util.COIN)).FromTxToEntry(tx4)
	txEntry4.ParentTx[txEntry1] = struct{}{}
	txEntry4.ParentTx[txEntry3] = struct{}{}

	t := make([]*mempool.TxEntry, 4)
	t[0] = txEntry1
	t[1] = txEntry2
	t[2] = txEntry3
	t[3] = txEntry4
	return t
}

func TestCreateNewBlockByFee(t *testing.T) {
	tempDir, err := initTestEnv(t)
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	// clear mempool data
	mempool.InitMempool()
	pool := mempool.GetInstance()

	pubKey := script.NewEmptyScript()
	pubKey.PushOpCode(opcodes.OP_TRUE)

	_, err = generateBlocks(pubKey, 101, 1000000)
	assert.Nil(t, err)

	gChain := chain.GetInstance()
	bl1Index := gChain.GetIndex(1)
	assert.NotNil(t, bl1Index)

	block1, ok := disk.ReadBlockFromDisk(bl1Index, gChain.GetParams())
	assert.True(t, ok)

	txSet := createTx(t, block1.Txs[0], pubKey)

	for _, entry := range txSet {
		err := pool.AddTx(entry, entry.ParentTx)
		if err != nil {
			t.Fatal(err)
		}
	}
	if pool.Size() != 4 {
		t.Fatal("add txEntry to mempool error")
	}

	ba := NewBlockAssembler(model.ActiveNetParams)
	assert.NotNil(t, ba)

	tmpStrategy := getStrategy()
	*tmpStrategy = sortByFee
	sc := script.NewEmptyScript()
	ba.CreateNewBlock(sc, BasicScriptSig())

	if len(ba.bt.Block.Txs) != 5 {
		t.Fatal("some transactions are inserted to block error")
	}

	if ba.bt.Block.Txs[4].GetHash() != txSet[1].Tx.GetHash() {
		t.Error("error sort by tx fee")
	}
}

//func TestCreateNewBlockByFeeRate(t *testing.T) {
//	mempool.InitMempool()
//
//	txSet := createTx()
//
//	pool := mempool.GetInstance()
//	for _, entry := range txSet {
//		pool.AddTx(entry, entry.ParentTx)
//	}
//
//	if pool.Size() != 4 {
//		t.Error("add txEntry to mempool error")
//	}
//
//	ba := NewBlockAssembler(model.ActiveNetParams)
//	tmpStrategy := getStrategy()
//	*tmpStrategy = sortByFeeRate
//
//	sc := script.NewScriptRaw([]byte{opcodes.OP_2DIV})
//	ba.CreateNewBlock(sc, BasicScriptSig())
//	if len(ba.bt.Block.Txs) != 5 {
//		t.Error("some transactions are inserted to block error")
//	}
//	// todo  test failed
//
//	//if ba.bt.Block.Txs[1].GetHash() != txSet[0].Tx.GetHash() {
//	//	t.Error("error sort by tx feerate")
//	//}
//	//
//	//if ba.bt.Block.Txs[2].GetHash() != txSet[1].Tx.GetHash() {
//	//	t.Error("error sort by tx feerate")
//	//}
//	//
//	//if ba.bt.Block.Txs[3].GetHash() != txSet[2].Tx.GetHash() {
//	//	t.Error("error sort by tx feerate")
//  	//}
//  	//
//  	//if ba.bt.Block.Txs[4].GetHash() != txSet[3].Tx.GetHash() {
//  	//	t.Error("error sort by tx feerate")
//  	//}
//}
