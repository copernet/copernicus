package mempool

import (
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type Mempool struct {
	CheckFrequency int32
}

func GetTxFromMemPool(hash *utils.Hash) *model.Tx {
	return new(model.Tx)
}
