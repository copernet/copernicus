package utxo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"io"
)

type Coin struct {
	txOut         txout.TxOut
	height        int32
	isCoinBase    bool
	dirty         bool //whether modified
	fresh         bool //whether add new
	isMempoolCoin bool
}

func (coin *Coin) GetHeight() int32 {
	return coin.height
}

func (coin *Coin) IsCoinBase() bool {
	return coin.isCoinBase
}

func (coin *Coin) IsMempoolCoin() bool {
	return coin.isMempoolCoin
}

// todo check coinbase height，lock time？
func (coin *Coin) IsSpendable() bool {
	return coin.txOut.IsNull()
}

func (coin *Coin) IsSpent() bool {
	return coin.txOut.IsNull()
}

func (coin *Coin) Clear() {
	coin.txOut.SetNull()
	coin.height = 0
	coin.isCoinBase = false
}

func (coin *Coin) GetTxOut() txout.TxOut {
	return coin.txOut
}

func (coin *Coin) GetScriptPubKey() *script.Script {
	txOut := coin.txOut
	return txOut.GetScriptPubKey()
}

func (coin *Coin) GetAmount() amount.Amount {
	return amount.Amount(coin.txOut.GetValue())
}

func (coin *Coin) DeepCopy() *Coin {
	newCoin := Coin{height: coin.height, isCoinBase: coin.isCoinBase, dirty: coin.dirty, fresh: coin.fresh, isMempoolCoin: coin.isMempoolCoin}
	outScript := coin.txOut.GetScriptPubKey()
	if coin.txOut.GetScriptPubKey() != nil {
		newOutScript := script.NewScriptRaw(outScript.GetData())
		//newOutScript.ParsedOpCodes = outScript.ParsedOpCodes
		newOut := txout.NewTxOut(coin.txOut.GetValue(), newOutScript)
		newCoin.txOut = *newOut
	}
	return &newCoin
}

func (coin *Coin) DynamicMemoryUsage() int64 {
	return int64(binary.Size(coin))
}

func (coin *Coin) Serialize(w io.Writer) error {
	if coin.IsSpent() {
		log.Debug("already spent")
		return errors.New("already spent")
	}
	var bit int32
	if coin.isCoinBase {
		bit = 1
	}
	heightAndIsCoinBase := (coin.height << 1) | bit
	if err := util.WriteVarLenInt(w, uint64(heightAndIsCoinBase)); err != nil {
		return err
	}
	tc := coin.txOut
	return tc.Serialize(w)
}

func (coin *Coin) Unserialize(r io.Reader) error {

	hicb, err := util.ReadVarLenInt(r)
	if err != nil {
		return err
	}
	heightAndIsCoinBase := int32(hicb)
	coin.height = heightAndIsCoinBase >> 1
	if (heightAndIsCoinBase & 1) == 1 {
		coin.isCoinBase = true
	}
	fmt.Println("coin.Unserialize=====", err, coin.height, coin.isCoinBase)
	err = coin.txOut.Unserialize(r)
	return err
}

//new an confirmed coin
func NewCoin(out *txout.TxOut, height int32, isCoinBase bool) *Coin {

	return &Coin{
		txOut:      *out,
		height:     height,
		isCoinBase: isCoinBase,
	}
}

//new an unconfirmed coin for mempool
func NewMempoolCoin(out *txout.TxOut) *Coin {
	return &Coin{
		txOut:         *out,
		isMempoolCoin: true,
	}
}

func NewEmptyCoin() *Coin {

	return &Coin{
		txOut:      *txout.NewTxOut(0, nil),
		height:     0,
		isCoinBase: false,
	}
}
