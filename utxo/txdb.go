package utxo

import (
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

const (
	DB_COIN    = 'C'
	DB_COINS   = 'c'
	DB_TXINDEX = 't'
)

func GetTxFromUTXO(hash *utils.Hash) *model.Tx {
	return new(model.Tx)
}
