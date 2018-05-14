package utxo

import (
	"errors"
	"io"
	"encoding/binary"
	"fmt"
	"github.com/btcboost/copernicus/model/txout"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/persist/db"
	"github.com/btcboost/copernicus/util/amount"
)

type Coin struct {
	txOut               txout.TxOut
	height              uint32
	isCoinBase          bool
	dirty bool //是否修改过
	fresh bool //是否是新增
	isMempoolCoin        bool
}

func (coin *Coin) GetHeight() uint32 {
	return coin.height
}

func (coin *Coin) IsCoinBase() bool {
	return coin.isCoinBase
}

func (coin *Coin) GetIsMempoolCoin() bool {
	return coin.isMempoolCoin
}
//coinbase检查高度，锁定时间？
func (coin *Coin) IsSpendable() bool {
	fmt.Printf("isspend=======%#v",coin)
	return coin.txOut.IsNull()
}

func (coin *Coin) IsSpent() bool {
	fmt.Printf("isspend=======%#v",coin)
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

func (coin *Coin) GetAmount() amount.Amount {
	return amount.Amount(coin.txOut.GetValue())
}



func (coin *Coin) DynamicMemoryUsage() int64{
	return int64(binary.Size(coin))
}

func (coin *Coin) Serialize(w io.Writer) error {
	if coin.IsSpent() {
		return errors.New("already spent")
	}
	w.Write([]byte{db.DbCoin})
	var bit uint32
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

func (coin *Coin) Unserialize(r io.Reader)error {
	buf := make([]byte, 1)
	r.Read(buf) //read database.DbCoin
	hicb, err := util.ReadVarLenInt(r)
	if err != nil {
		return err
	}
	heightAndIsCoinBase := uint32(hicb)
	coin.height = heightAndIsCoinBase >> 1
	if (heightAndIsCoinBase & 1) == 1{
		coin.isCoinBase =  true
	}
	err = coin.txOut.Unserialize(r)
	return err
}

//生成一个确认的coin
func NewCoin(out *txout.TxOut, height uint32, isCoinBase bool) *Coin {

	return &Coin{
		txOut:               *out,
		height:              height,
		isCoinBase:          isCoinBase,
    }
}

//生成一个mempool中未确认的coin
func NewMempoolCoin(out *txout.TxOut)*Coin{
	return &Coin{
		txOut:               *out,
		isMempoolCoin:true,
	}
}

func NewEmptyCoin() *Coin {

	return &Coin{
		txOut:               *txout.NewTxOut(),
		height: 0,
		isCoinBase:false,
	}
}


