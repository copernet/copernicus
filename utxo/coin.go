package utxo

import (
	"errors"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

type Coin struct {
	txOut               core.TxOut
	height              uint32
	isCoinBase          bool
}

func (coin *Coin) GetHeight() uint32 {
	return coin.height
}

func (coin *Coin) IsCoinBase() bool {
	return coin.isCoinBase
}

func (coin *Coin) IsSpent() bool {
	return coin.txOut.IsNull()
}

func (coin *Coin) Clear() {
	coin.txOut.SetNull()
	coin.height = 0
	coin.isCoinBase = false
}

func (coin *Coin) GetTxOut() *core.TxOut {
	return &coin.txOut
}

func (coin *Coin) Serialize(w io.Writer) error {
	if coin.IsSpent() {
		return errors.New("already spent")
	}
	var bit uint32
	if coin.isCoinBase {
		bit = 1
	}
	heightAndIsCoinBase := (coin.height << 1) | bit
	if err := utils.WriteVarLenInt(w, uint64(heightAndIsCoinBase)); err != nil {
		return err
	}
	tc := NewTxoutCompressor(&coin.txOut)
	return tc.Serialize(w)
}

func (coin *Coin) Unserialize(r io.Reader) error {
	hicb, err := utils.ReadVarLenInt(r)
	if err != nil {
		return err
	}
	heightAndIsCoinBase := uint32(hicb)
	coin.height = heightAndIsCoinBase >> 1
	if (heightAndIsCoinBase & 1) == 1{
		coin.isCoinBase =  true
	}
	tc := NewTxoutCompressor(&coin.txOut)
	return tc.Unserialize(r)
}

func NewCoin(out *core.TxOut, height uint32, isCoinBase bool) *Coin {

	return &Coin{
		txOut:               *out,
		height:height,
		isCoinBase:isCoinBase,
    }
}

type CoinView interface {
	GetCoin(core.OutPoint) (Coin, bool)
	HaveCoin(core.OutPoint) bool
	GetBestBlock() utils.Hash
	GetHeadBlocks() []utils.Hash
	BatchWrite(*coinMap, utils.Hash)
}

type CoinViewCursor interface {
	GetKey() (core.OutPoint, bool)
	GetVal() (Coin, bool)
	GetValSize() int
	Valid() bool
	Next()
}
