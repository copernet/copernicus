package utxo

import (
	"io"

	"encoding/binary"

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

func (coin *Coin) Deserialize(reader io.Reader) error {
	heightAndIsCoinBase, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	coin.HeightAndIsCoinBase = heightAndIsCoinBase
	if err != nil {
		return err
	}
	txOut := model.TxOut{}
	err = txOut.Deserialize(reader)
	coin.TxOut = &txOut
	return err
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
