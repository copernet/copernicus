package utxo

import (
	//"bytes"
	//"encoding/binary"
	"errors"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

type Coin struct {
	txOut               core.TxOut
	heightAndIsCoinBase uint32
}

func (coin *Coin) GetHeight() uint32 {
	return coin.heightAndIsCoinBase >> 1
}

func (coin *Coin) IsCoinBase() bool {
	return coin.heightAndIsCoinBase&0x01 > 0
}

func (coin *Coin) IsSpent() bool {
	return coin.txOut.IsNull()
}

func (coin *Coin) Clear() {
	coin.txOut.SetNull()
	coin.heightAndIsCoinBase = 0
}

func (coin *Coin) GetTxOut() *core.TxOut {
	return &coin.txOut
}

func (coin *Coin) Serialize(w io.Writer) error {
	if coin.IsSpent() {
		return errors.New("already spent")
	}
	if err := utils.WriteVarInt(w, coin.heightAndIsCoinBase); err != nil {
		return err
	}
	tc := NewTxoutCompressor(&coin.txOut)
	return tc.Serialize(w)
}

func (coin *Coin) Unserialize(r io.Reader) error {
	hicb, err := utils.ReadVarInt(r)
	if err != nil {
		return err
	}
	coin.heightAndIsCoinBase = uint32(hicb)
	tc := NewTxoutCompressor(&coin.txOut)
	return tc.Unserialize(r)
}

func NewCoin(out *core.TxOut, height uint32, isCoinBase bool) *Coin {
	var bit uint32
	if isCoinBase {
		bit = 1
	}
	return &Coin{
		txOut:               out,
		heightAndIsCoinBase: (height << 1) | bit,
	}
}

func NewEmptyCoin() *Coin {
	return &Coin{
		heightAndIsCoinBase: 0,
		txOut:               core.NewTxOut(-1, []byte{}),
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

func DeepCopyCoin(coin *Coin) Coin {
	dst := Coin{
		txOut: &core.TxOut{
			Script: core.NewScriptRaw([]byte{}),
		},
	}

	dst.HeightAndIsCoinBase = coin.HeightAndIsCoinBase
	if coin.TxOut != nil {
		dst.TxOut.Value = coin.TxOut.Value
		dst.TxOut.SigOpCount = coin.TxOut.SigOpCount
		if coin.TxOut.Script != nil {
			tmp := coin.TxOut.Script.GetScriptByte()
			dst.TxOut.Script = core.NewScriptRaw(tmp)
		} else {
			dst.TxOut.Script = nil
		}
	} else {
		dst.TxOut = nil
	}

	return dst
}
