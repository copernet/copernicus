package lchain_test

import (
	"encoding/json"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/logic/lmerkleroot"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
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
	"github.com/copernet/copernicus/model/versionbits"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/service"
	"github.com/copernet/copernicus/service/mining"
	"github.com/copernet/copernicus/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func coinbaseScriptSigWithHeight(extraNonce uint, height int32) *script.Script {
	coinbaseFlag := "copernicus.............................."
	scriptSig := script.NewEmptyScript()

	heightNum := script.NewScriptNum(int64(height))
	scriptSig.PushScriptNum(heightNum)

	extraNonceNum := script.NewScriptNum(int64(extraNonce))
	scriptSig.PushScriptNum(extraNonceNum)

	scriptSig.PushData([]byte(coinbaseFlag))

	return scriptSig
}

func generateDummyBlocks(scriptPubKey *script.Script, generate int, maxTries uint64,
	preHeight int32, txs []*tx.Tx) ([]util.Hash, error) {
	heightEnd := preHeight + int32(generate)
	height := preHeight
	nInnerLoopCount := uint32(0x100000)

	ret := make([]util.Hash, 0)
	var extraNonce uint
	blkHash := *chain.GetInstance().GetIndex(preHeight).GetBlockHash()
	for height < heightEnd {
		bk := block.NewBlock()
		bk = createDummyBlock(scriptPubKey, coinbaseScriptSigWithHeight(extraNonce, height+1), bk, blkHash)
		bk.Txs = append(bk.Txs, txs...)
		bk.Header.MerkleRoot = lmerkleroot.BlockMerkleRoot(bk.Txs, nil)

		powCheck := pow.Pow{}
		bits := bk.Header.Bits
		for maxTries > 0 && bk.Header.Nonce < nInnerLoopCount {
			maxTries--
			bk.Header.Nonce++
			hash := bk.GetHash()
			if powCheck.CheckProofOfWork(&hash, bits, model.ActiveNetParams) {
				break
			}
		}

		if maxTries == 0 {
			break
		}

		if bk.Header.Nonce == nInnerLoopCount {
			extraNonce++
			continue
		}

		fNewBlock := false
		if service.ProcessNewBlock(bk, true, &fNewBlock) != nil {
			return nil, errors.New("ProcessNewBlock, block not accepted")
		}

		height++
		extraNonce = 0

		blkHash = bk.GetHash()
		ret = append(ret, blkHash)
	}

	return ret, nil
}

func createDummyBlock(scriptPubKey, scriptSig *script.Script, bk *block.Block, prehash util.Hash) *block.Block {

	// add dummy coinbase tx as first transaction
	bk.Txs = make([]*tx.Tx, 0, 100000)
	bk.Txs = append(bk.Txs, tx.NewTx(0, tx.DefaultVersion))

	indexPrev := chain.GetInstance().FindBlockIndex(prehash)

	blkVersion := versionbits.ComputeBlockVersion(indexPrev, model.ActiveNetParams, versionbits.VBCache)
	bk.Header.Version = int32(blkVersion)

	bk.Header.Time = uint32(util.GetAdjustedTime())

	// Create coinbase transaction
	coinbaseTx := tx.NewTx(0, tx.DefaultVersion)

	outPoint := outpoint.OutPoint{Hash: util.HashZero, Index: 0xffffffff}

	coinbaseTx.AddTxIn(txin.NewTxIn(&outPoint, scriptSig, 0xffffffff))
	// value represents total reward(fee and block generate reward)
	coinbaseTx.AddTxOut(txout.NewTxOut(50, scriptPubKey))
	bk.Txs[0] = coinbaseTx

	// Fill in header.
	if indexPrev == nil {
		bk.Header.HashPrevBlock = util.HashZero
	} else {
		bk.Header.HashPrevBlock = *indexPrev.GetBlockHash()
	}
	mining.UpdateTime(bk, indexPrev)
	p := pow.Pow{}
	bk.Header.Bits = p.GetNextWorkRequired(indexPrev, &bk.Header, model.ActiveNetParams)
	bk.Header.Nonce = 0

	bk.SerializeSize()
	bk.Checked = true

	return bk
}

func initTestEnv(t *testing.T, args []string) (dirpath string, err error) {
	conf.Cfg = conf.InitConfig(args)

	conf.Cfg.Chain.UtxoHashStartHeight = 0
	conf.Cfg.Chain.UtxoHashEndHeight = 1000

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

	//init log
	logDir := filepath.Join(conf.DataDir, log.DefaultLogDirname)
	if !conf.FileExists(logDir) {
		err := os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	logConf := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
	}{
		FileName: logDir + "/" + conf.Cfg.Log.FileName + ".log",
		Level:    log.GetLevel(conf.Cfg.Log.Level),
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}
	log.Init(string(configuration))

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
	tchain := chain.GetInstance()
	*tchain = *chain.NewChain()

	// Load blockindex DB
	lblockindex.LoadBlockIndexDB()

	err = lchain.InitGenesisChain()
	assert.Nil(t, err)

	mempool.InitMempool()

	crypto.InitSecp256()

	ltx.ScriptVerifyInit()

	return unitTestDataDirPath, nil
}

func TestActivateBestChain(t *testing.T) {
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)
	defer os.RemoveAll(testDir)

	tChain := chain.GetInstance()

	pubKey := script.NewEmptyScript()
	pubKey.PushOpCode(opcodes.OP_TRUE)

	_, err = generateDummyBlocks(pubKey, 101, 1000000, 0, nil)
	assert.Nil(t, err)

	bIndex := tChain.GetIndex(1)
	assert.NotNil(t, bIndex)
	block1, ok := disk.ReadBlockFromDisk(bIndex, tChain.GetParams())
	assert.True(t, ok)

	var txs []*tx.Tx
	lockTime := uint32(0)
	transaction := tx.NewTx(lockTime, tx.DefaultVersion)
	preOut := outpoint.NewOutPoint(block1.Txs[0].GetHash(), 0)
	newScript := script.NewEmptyScript()
	txIn := txin.NewTxIn(preOut, newScript, math.MaxUint32-1)
	transaction.AddTxIn(txIn)

	for i := 0; i < 20; i++ {
		txOut := txout.NewTxOut(1, pubKey)
		transaction.AddTxOut(txOut)
	}

	txs = append(txs, transaction)
	_, err = generateDummyBlocks(pubKey, 1, 1000000, 101, txs)
	assert.Nil(t, err)
	height := tChain.TipHeight()
	assert.Equal(t, int32(102), height)

	_, err = generateDummyBlocks(pubKey, 103, 1000000, 0, nil)
	assert.Nil(t, err)
	height = tChain.TipHeight()
	assert.Equal(t, int32(103), height)
}

func TestGetUTXOStats(t *testing.T) {
	// set params, don't modify!
	model.SetRegTestParams()
	// clear chain data of last test case
	testDir, err := initTestEnv(t, []string{"--regtest"})
	assert.Nil(t, err)
	defer os.RemoveAll(testDir)

	pubKey := script.NewEmptyScript()
	pubKey.PushOpCode(opcodes.OP_TRUE)

	cdb := utxo.GetUtxoCacheInstance().(*utxo.CoinsLruCache).GetCoinsDB()

	_, err = generateDummyBlocks(pubKey, 100, 1000000, 0, nil)
	assert.Nil(t, err)

	_, err = lchain.GetUTXOStats(cdb)
	assert.Nil(t, err)
}
