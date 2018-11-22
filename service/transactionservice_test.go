package service

import (
	"bytes"
	"encoding/hex"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/logic/lmerkleroot"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/bitcointime"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/service/mining"
	"github.com/copernet/copernicus/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"testing"
)

const nInnerLoopCount = 0x100000

func generateBlocks(scriptPubKey *script.Script, generate int, maxTries uint64) (interface{}, error) {
	heightStart := chain.GetInstance().Height()
	heightEnd := heightStart + int32(generate)
	height := heightStart
	params := model.ActiveNetParams

	ret := make([]string, 0)
	var extraNonce uint
	for height < heightEnd {
		ts := bitcointime.NewMedianTime()
		ba := mining.NewBlockAssembler(params, ts)
		bt := ba.CreateNewBlock(scriptPubKey, mining.CoinbaseScriptSig(extraNonce))
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
		if ProcessNewBlock(bt.Block, true, &fNewBlock) != nil {
			return nil, errors.New("ProcessNewBlock, block not accepted")
		}

		height++
		extraNonce = 0

		blkHash := bt.Block.GetHash()
		ret = append(ret, blkHash.String())
	}

	return ret, nil
}

func TestProcessTransaction(t *testing.T) {
	blockHex1260985 := "00000020dcc0f5f3fe3ec1cf174ead464c20741b74359761e9d6b933315a0f00000000009300e1b353c0f760e4bf8a02173231ab668ab873640b852a8c4a24a3ce20d072ca31bb5b0b692c1b4caa98700301000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4103b93d130e0d2f4249503130302f454233322f04ca31bb5b046d1a86250c7131bb5bd40000000000000016426974636f696e205854206174203738353066663633ffffffff025a1aa804000000001976a914158b5d181552c9f4f267c0de68aae4963043993988ac0000000000000000266a24aa21a9ed13bc96c3d459e6225b28d045b30a98f3ba03790daf04803d44972b93a5d23917000000000200000001bebf7bab9021fd3422231e13b39744ee788584bc42b63d15d163d26860d2ea5b010000006b4830450221009439545e50e255cc03d9685c9182415fa1c31bdae1c74fb41cf55ef371e9c5ac022070367d1685deec304bc92863d291788f4355ab3e1ab755afb1189cee4403a891412102413ce16bc4975dcc6945febd4b4786e0d0006c1ac4ff49cdbf71cc2cb5734b70feffffff030000000000000000456a4362636e73000001f4676f626162793200626368746573743a71716b35396a736870386c7934793667346c35686e6565766871663334347a647271756879306e657178008d340f00000000001976a9142d42ca1709fe4a9348afe979e72cb8131ad44d1888ac22020000000000001976a9142d42ca1709fe4a9348afe979e72cb8131ad44d1888acb83d13000200000001692c1265e5d05780ccec6e116c6432c78457ee4c2b2551c797f98b8780431d66010000006a473044022063ec6ce2bdd8c15dd7d53d1d230c840c39e6baa5edd34312a3e852a2184ca08502204bd7551b92b769b6aa668c3e3c7b8fa0bbe02190d7207036d52673c7120f868f412102413ce16bc4975dcc6945febd4b4786e0d0006c1ac4ff49cdbf71cc2cb5734b70feffffff030000000000000000416a3f62636e7300000213676f626162793200516d53707577656a55476a52456d6773766d386571335a647353376d5654484352505a6d4c6955713834533978380024310f00000000001976a9142d42ca1709fe4a9348afe979e72cb8131ad44d1888ac22020000000000001976a9142d42ca1709fe4a9348afe979e72cb8131ad44d1888acb83d1300"
	block1260985Bytes, _ := hex.DecodeString(blockHex1260985)
	block1260985 := block.NewBlock()
	err := block1260985.Unserialize(bytes.NewBuffer(block1260985Bytes))
	assert.Nil(t, err)

	txs := block1260985.Txs

	nodeID := int64(0)
	recentRejects := make(map[util.Hash]struct{})
	acceptedTxs, missTxHash, rejectTxHash, err := ProcessTransaction(txs[1], recentRejects, nodeID)
	assert.NotNil(t, err)
	assert.Empty(t, missTxHash)
	assert.Equal(t, txs[1].GetHash(), rejectTxHash[0])
	assert.Equal(t, 0, len(acceptedTxs))
}

func TestProcessTransactionNormal(t *testing.T) {
	var dataElement = []byte{203, 72, 18, 50, 41, 156, 213, 116, 49, 81, 172, 75, 45, 99, 174, 25, 142, 123, 176, 169}

	// clear chain data of last test case
	gChain := chain.GetInstance()
	// set params, don't modify!
	model.SetRegTestParams()
	*gChain = *chain.NewChain()

	testDir, err := initTestEnv(t, []string{"--regtest"}, false)
	assert.Nil(t, err)
	pubKey := script.NewEmptyScript()
	pubKey.PushOpCode(opcodes.OP_DUP)
	pubKey.PushOpCode(opcodes.OP_HASH160)
	pubKey.PushSingleData(dataElement)
	pubKey.PushOpCode(opcodes.OP_EQUALVERIFY)
	pubKey.PushOpCode(opcodes.OP_CHECKSIG)

	coinBaseScriptPubKey := script.NewEmptyScript()
	coinBaseScriptPubKey.PushOpCode(opcodes.OP_TRUE)
	_, err = generateBlocks(coinBaseScriptPubKey, 101, 1000000)
	assert.Nil(t, err)

	bl1Index := gChain.GetIndex(1)
	assert.NotNil(t, bl1Index)

	block1, ok := disk.ReadBlockFromDisk(bl1Index, gChain.GetParams())
	assert.True(t, ok)

	lockTime := uint32(0)
	transaction := tx.NewTx(lockTime, tx.DefaultVersion)
	preOut := outpoint.NewOutPoint(block1.Txs[0].GetHash(), 0)
	newScript := script.NewEmptyScript()
	txIn := txin.NewTxIn(preOut, newScript, math.MaxUint32-1)
	transaction.AddTxIn(txIn)

	txOut := txout.NewTxOut(10, pubKey)
	transaction.AddTxOut(txOut)
	transaction.AddTxOut(txOut)
	transaction.AddTxOut(txOut)
	transaction.AddTxOut(txOut)
	transaction.AddTxOut(txOut)

	nodeID := int64(0)
	recentRejects := make(map[util.Hash]struct{})
	acceptedTxs, missTxHash, rejectTxHash, err := ProcessTransaction(transaction, recentRejects, nodeID)
	assert.Nil(t, err)
	assert.Equal(t, transaction, acceptedTxs[0])
	assert.Empty(t, missTxHash)
	assert.Empty(t, rejectTxHash)

	// test Orphan transcation
	tscaOrphan := tx.NewTx(lockTime, tx.DefaultVersion)
	preOut = outpoint.NewOutPoint(util.Hash{1, 2, 3}, 0)
	OrphanScript := script.NewEmptyScript()
	txIn = txin.NewTxIn(preOut, OrphanScript, math.MaxUint32-1)
	tscaOrphan.AddTxIn(txIn)

	txOut = txout.NewTxOut(10, pubKey)
	tscaOrphan.AddTxOut(txOut)
	tscaOrphan.AddTxOut(txOut)
	tscaOrphan.AddTxOut(txOut)
	tscaOrphan.AddTxOut(txOut)
	tscaOrphan.AddTxOut(txOut)

	acceptedTxs, missTxHash, rejectTxHash, err = ProcessTransaction(tscaOrphan, recentRejects, nodeID)
	assert.Equal(t, errcode.New(errcode.TxErrNoPreviousOut), err)
	assert.Equal(t, 1, len(missTxHash))
	assert.Empty(t, acceptedTxs)
	assert.Empty(t, rejectTxHash)

	defer os.RemoveAll(testDir)
}
