package lmerkleblock

import (
	"bytes"
	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/util"
	"gopkg.in/fatih/set.v0"
	"math"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	persist.InitPersistGlobal()
	conf.Cfg = conf.InitConfig([]string{})
	os.Exit(m.Run())
}

func TestMerkleBlock(t *testing.T) {

	bk := block.NewBlock()
	setTxIds := set.New()

	for i := 0; i < 100; i++ {
		lockTime := uint32(0)
		transaction := tx.NewTx(lockTime, tx.DefaultVersion)
		preOut := outpoint.NewOutPoint(*util.GetRandHash(), 0)
		newScript := script.NewEmptyScript()
		txIn := txin.NewTxIn(preOut, newScript, math.MaxUint32-1)
		transaction.AddTxIn(txIn)

		pubKey := script.NewEmptyScript()
		pubKey.PushOpCode(opcodes.OP_TRUE)
		txOut := txout.NewTxOut(10, pubKey)
		transaction.AddTxOut(txOut)

		bk.Txs = append(bk.Txs, transaction)
	}

	for i, tx := range bk.Txs {
		if i > 47 {
			break
		}
		setTxIds.Add(tx.GetHash())
	}

	mb := NewMerkleBlock(bk, setTxIds)
	buf := bytes.NewBuffer(nil)
	mb.Serialize(buf)
	mb.Unserialize(buf)

	matches := make([]util.Hash, 0)
	items := make([]int, 0)
	mb.Txn.ExtractMatches(&matches, &items)

}
