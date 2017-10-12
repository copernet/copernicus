package utxo

import "github.com/btcboost/copernicus/model"

type Coin struct {
	TxOut               *model.TxOut
	HeightAndIsCoinBase uint32
}
