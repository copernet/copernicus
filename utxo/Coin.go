package utxo

import (
	"github.com/btcboost/copernicus/model"
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
