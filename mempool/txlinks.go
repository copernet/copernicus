package mempool

import (
	"github.com/btcboost/copernicus/container"
)

type TxLinks struct {
	Parents  *container.Set // The set element type : *TxMempoolEntry
	Children *container.Set // The set element type : *TxMempoolEntry
}
