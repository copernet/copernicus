package mining

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
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
	"github.com/copernet/copernicus/util/cashaddr"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"testing"
)

func initTestEnv(t *testing.T, initScriptVerify bool) (dirpath string, err error) {
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

	// Init UTXO DB
	utxoDbCfg := &db.DBOption{
		FilePath:  conf.DataDir + "/chainstate",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
	utxo.InitUtxoLruTip(&utxoConfig)

	// Init blocktree DB
	blkDbCfg := &db.DBOption{
		FilePath:  conf.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	chain.InitGlobalChain(blkdb.GetInstance())

	persist.InitPersistGlobal(blkdb.GetInstance())

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

type SignBox struct {
	signKey   crypto.PrivateKey
	payAddr   cashaddr.Address
	payScript []byte
	keys      []*crypto.PrivateKey
}

func newSignBox(chainParams *model.BitcoinParams) (signBox *SignBox, err error) {
	// Use a hard coded key pair for deterministic results.
	keyBytes, err := hex.DecodeString("700868df1838811ffbdf918fb482c1f7e" +
		"ad62db4b97bd7012c23e726485e577d")
	if err != nil {
		return nil, err
	}
	//signKey, signPub := btcec.PrivKeyFromBytes(btcec.S256(), keyBytes)
	signKey, signPub := generateKeys(keyBytes)

	// Generate associated pay-to-script-hash address and resulting payment
	// script.
	//pubKeyBytes := signPub.SerializeCompressed()
	//payPubKeyAddr, err := btcutil.NewAddressPubKey(pubKeyBytes, chainParams)
	var payAddr cashaddr.Address
	payAddr, err = cashaddr.NewCashAddressPubKeyHash(signPub.ToHash160(), chainParams)
	if err != nil {
		return nil, err
	}
	// payAddr := payPubKeyAddr.AddressPubKeyHash()
	// pkScript, err := txscript.PayToAddrScript(payAddr)
	pkScript, err := cashaddr.CashPayToAddrScript(payAddr)
	if err != nil {
		return nil, err
	}

	signBox = &SignBox{
		signKey:   signKey,
		payAddr:   payAddr,
		payScript: pkScript,
		keys:      []*crypto.PrivateKey{&signKey},
	}

	return signBox, nil
}

func getKeyStore(pkeys []*crypto.PrivateKey) *crypto.KeyStore {
	keyStore := crypto.NewKeyStore()
	for _, pkey := range pkeys {
		keyStore.AddKey(pkey)
	}
	return keyStore
}

func generateKeys(keyBytes []byte) (crypto.PrivateKey, crypto.PublicKey) {
	// var keyBytes []byte
	// for i := 0; i < 32; i++ {
	// 	keyBytes = append(keyBytes, byte(rand.Uint32()%256))
	// }
	privKey := *crypto.PrivateKeyFromBytes(keyBytes)
	return privKey, *privKey.PubKey()
}

func createTx(tt *testing.T, baseTx *tx.Tx, pubKey *script.Script) []*mempool.TxEntry {

	//keyStore := getKeyStore(signBox.keys)
	//coinsMap := utxo.NewEmptyCoinsMap()

	//outpointK := &outpoint.OutPoint{Hash: baseTx.GetHash(), Index: 0}
	//coinsMap.AddCoin(outpointK, utxo.NewFreshCoin(baseTx.GetTxOut(0), 0, true), false)

	testEntryHelp := NewTestMemPoolEntry()

	//pubKey := script.NewScriptRaw(signBox.payScript)
	tx1 := tx.NewTx(0, 0x02)
	tx1.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(baseTx.GetHash(), 0), script.NewEmptyScript(), math.MaxUint32-1))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(22*util.COIN), pubKey))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(22*util.COIN), pubKey))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx1.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))

	//outpointK = &outpoint.OutPoint{Hash: tx1.GetHash(), Index: 0}
	//coinsMap.AddCoin(outpointK, utxo.NewFreshCoin(tx1.GetTxOut(0), 0, false), false)
	//outpointK = &outpoint.OutPoint{Hash: tx1.GetHash(), Index: 1}
	//coinsMap.AddCoin(outpointK, utxo.NewFreshCoin(tx1.GetTxOut(1), 1, false), false)

	//ltx.SignRawTransaction([]*tx.Tx{tx1}, nil, keyStore, coinsMap, crypto.SigHashAll|crypto.SigHashForkID)
	txEntry1 := testEntryHelp.SetTime(util.GetTimeSec()).SetFee(amount.Amount(1 * util.COIN)).FromTxToEntry(tx1)

	tx2 := tx.NewTx(0, 0x02)
	// reference relation(tx2 -> tx1)
	tx2.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(tx1.GetHash(), 0), script.NewEmptyScript(), math.MaxUint32-1))

	tx2.AddTxOut(txout.NewTxOut(amount.Amount(16*util.COIN), pubKey))
	tx2.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx2.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx2.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx2.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx2.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))

	//outpointK = &outpoint.OutPoint{Hash: tx2.GetHash(), Index: 0}
	//coinsMap.AddCoin(outpointK, utxo.NewFreshCoin(tx2.GetTxOut(0), 0, false), false)
	//ltx.SignRawTransaction([]*tx.Tx{tx2}, nil, keyStore, coinsMap, crypto.SigHashAll|crypto.SigHashForkID)
	txEntry2 := testEntryHelp.SetTime(util.GetTimeSec()).SetFee(amount.Amount(1 * util.COIN)).FromTxToEntry(tx2)
	txEntry2.ParentTx[txEntry1] = struct{}{}

	//  modify tx3's content to avoid to get the same hash with tx2
	tx3 := tx.NewTx(0, 0x02)
	// reference relation(tx3 -> tx1)
	tx3.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(tx1.GetHash(), 1), script.NewEmptyScript(), math.MaxUint32-1))
	tx3.AddTxOut(txout.NewTxOut(amount.Amount(15*util.COIN), pubKey))
	tx3.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx3.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx3.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx3.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))

	//outpointK = &outpoint.OutPoint{Hash: tx3.GetHash(), Index: 0}
	//coinsMap.AddCoin(outpointK, utxo.NewFreshCoin(tx3.GetTxOut(0), 0, false), false)
	//ltx.SignRawTransaction([]*tx.Tx{tx3}, nil, keyStore, coinsMap, crypto.SigHashAll|crypto.SigHashForkID)
	txEntry3 := testEntryHelp.SetTime(util.GetTimeSec()).SetFee(amount.Amount(2 * util.COIN)).FromTxToEntry(tx3)
	txEntry3.ParentTx[txEntry1] = struct{}{}

	tx4 := tx.NewTx(0, 0x02)
	// reference relation(tx4 -> tx3 -> tx1)
	tx4.AddTxIn(txin.NewTxIn(outpoint.NewOutPoint(tx3.GetHash(), 0), script.NewEmptyScript(), math.MaxUint32-1))
	tx4.AddTxOut(txout.NewTxOut(amount.Amount(10*util.COIN), pubKey))
	tx4.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx4.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx4.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))
	tx4.AddTxOut(txout.NewTxOut(amount.Amount(1*util.COIN), pubKey))

	//outpointK = &outpoint.OutPoint{Hash: tx4.GetHash(), Index: 0}
	//coinsMap.AddCoin(outpointK, utxo.NewFreshCoin(tx4.GetTxOut(0), 0, false), false)
	//ltx.SignRawTransaction([]*tx.Tx{tx4}, nil, keyStore, coinsMap, crypto.SigHashAll|crypto.SigHashForkID)
	txEntry4 := testEntryHelp.SetTime(util.GetTimeSec()).SetFee(amount.Amount(1 * util.COIN)).FromTxToEntry(tx4)

	txEntry4.ParentTx[txEntry1] = struct{}{}
	txEntry4.ParentTx[txEntry3] = struct{}{}

	t := make([]*mempool.TxEntry, 4)
	t[0] = txEntry1
	t[1] = txEntry2
	t[2] = txEntry3
	t[3] = txEntry4
	return t
}

func TestCTORAndSortByFee(t *testing.T) {
	tempDir, err := initTestEnv(t, true)
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	// clear mempool data
	mempool.InitMempool()
	pool := mempool.GetInstance()

	//chainParams := model.ActiveNetParams
	//signBox, err := newSignBox(chainParams)
	//if err != nil {
	//	t.Fatal(err)
	//}

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

	//test sort by fee
	tmpStrategy := getStrategy()
	*tmpStrategy = sortByFee
	sc := script.NewEmptyScript()
	sc.PushOpCode(opcodes.OP_TRUE)
	var extraNonce uint

	bt := ba.CreateNewBlock(sc, CoinbaseScriptSig(extraNonce))
	if bt == nil {
		t.Fatal("create new block failed")
	}

	if len(ba.bt.Block.Txs) != 5 {
		fmt.Println(len(ba.bt.Block.Txs))
		t.Fatal("some transactions are inserted to block error")
	}

	//test CTOR
	if ba.bt.Block.Txs[1].GetHash().String() > ba.bt.Block.Txs[2].GetHash().String() {
		t.Errorf("the CTOR failed,hash is:%s", ba.bt.Block.Txs[1].GetHash().String())
	}
	if ba.bt.Block.Txs[2].GetHash().String() > ba.bt.Block.Txs[3].GetHash().String() {
		t.Errorf("the CTOR failed,hash is:%s", ba.bt.Block.Txs[2].GetHash().String())
	}
	if ba.bt.Block.Txs[3].GetHash().String() > ba.bt.Block.Txs[4].GetHash().String() {
		t.Errorf("the CTOR failed,hash is:%s", ba.bt.Block.Txs[3].GetHash().String())
	}
	if ba.bt.Block.Txs[4].GetHash().String() < ba.bt.Block.Txs[1].GetHash().String() {
		t.Errorf("the CTOR failed,hash is:%s", ba.bt.Block.Txs[4].GetHash().String())
	}
}

func TestCreateNewBlockByFeeRate(t *testing.T) {
	// clear chain data of last test case
	gChain := chain.GetInstance()
	*gChain = *chain.NewChain()

	tempDir, err := initTestEnv(t, false)
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	// clear mempool data
	mempool.InitMempool()
	pool := mempool.GetInstance()

	//chainParams := model.ActiveNetParams
	//signBox, err := newSignBox(chainParams)
	//if err != nil {
	//	t.Fatal(err)
	//}

	pubKey := script.NewEmptyScript()
	pubKey.PushOpCode(opcodes.OP_TRUE)

	_, err = generateBlocks(pubKey, 101, 1000000)
	assert.Nil(t, err)

	bl1Index := gChain.GetIndex(1)
	assert.NotNil(t, bl1Index)

	block1, ok := disk.ReadBlockFromDisk(bl1Index, gChain.GetParams())
	assert.True(t, ok)

	txSet := createTx(t, block1.Txs[0], pubKey)

	for _, entry := range txSet {
		pool.AddTx(entry, entry.ParentTx)
	}

	if pool.Size() != 4 {
		t.Error("add txEntry to mempool error")
	}

	ba := NewBlockAssembler(model.ActiveNetParams)
	tmpStrategy := getStrategy()
	*tmpStrategy = sortByFeeRate

	sc := script.NewEmptyScript()
	sc.PushOpCode(opcodes.OP_TRUE)
	var extraNonce uint
	bt := ba.CreateNewBlock(sc, CoinbaseScriptSig(extraNonce))
	if bt == nil {
		t.Fatal("create new block failed")
	}
	if len(ba.bt.Block.Txs) != 5 {
		t.Error("some transactions are inserted to block error")
	}
}
