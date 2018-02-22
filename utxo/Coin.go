package utxo

import (
	"bytes"
	"encoding/binary"
	"io"
	"unsafe"

	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type Coin struct {
	TxOut               *model.TxOut
	HeightAndIsCoinBase uint32
}

func (coin *Coin) GetHeight() uint32 {
	return coin.HeightAndIsCoinBase >> 1
}

func (coin *Coin) IsCoinBase() bool {
	return coin.HeightAndIsCoinBase&0x01 > 0
}

func (coin *Coin) IsSpent() bool {
	return coin.TxOut.IsNull()
}

func (coin *Coin) Clear() {
	coin.TxOut.SetNull()
	coin.HeightAndIsCoinBase = 0
}

func (coin *Coin) Serialize(writer io.Writer) error {
	err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, coin.HeightAndIsCoinBase)
	if err != nil {
		return err
	}
	return coin.TxOut.Serialize(writer)
}

func (coin *Coin) GetSerialize() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := coin.Serialize(buf)
	return buf.Bytes(), err
}

func (coin *Coin) DynamicMemoryUsage() int64 {
	return int64(unsafe.Sizeof(coin.TxOut.Script.ParsedOpCodes))
}

func DeserializeCoin(reader io.Reader) (*Coin, error) {
	coin := new(Coin)
	heightAndIsCoinBase, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	coin.HeightAndIsCoinBase = heightAndIsCoinBase
	if err != nil {
		return nil, err
	}
	txOut := model.TxOut{}
	err = txOut.Deserialize(reader)
	coin.TxOut = &txOut
	return coin, err
}

func NewCoin(out *model.TxOut, height uint32, isCoinBase bool) *Coin {
	var bit uint32
	if isCoinBase {
		bit = 1
	}
	return &Coin{
		TxOut:               out,
		HeightAndIsCoinBase: (height << 1) | bit,
	}
}

func NewEmptyCoin() *Coin {
	return &Coin{
		HeightAndIsCoinBase: 0,
		TxOut:               model.NewTxOut(-1, []byte{}),
	}
}

func DeepCopyCoin(coin *Coin) Coin {
	dst := Coin{
		TxOut: &model.TxOut{
			Script: model.NewScriptRaw([]byte{}),
		},
	}

	dst.HeightAndIsCoinBase = coin.HeightAndIsCoinBase
	if coin.TxOut != nil {
		dst.TxOut.Value = coin.TxOut.Value
		dst.TxOut.SigOpCount = coin.TxOut.SigOpCount
		if coin.TxOut.Script != nil {
			tmp := coin.TxOut.Script.GetScriptByte()
			dst.TxOut.Script = model.NewScriptRaw(tmp)
		} else {
			dst.TxOut.Script = nil
		}
	} else {
		dst.TxOut = nil
	}

	return dst
}
