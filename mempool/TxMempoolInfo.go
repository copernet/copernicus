package mempool

import (
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type TxMempoolInfo struct {
	Tx       *model.Tx
	Time     int64
	FeeRate  utils.FeeRate
	FeeDelta int64
}
