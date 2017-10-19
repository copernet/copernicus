package mempool

import (
	"github.com/btcboost/copernicus/algorithm"
)

type TxLinks struct {
	Parents  *algorithm.Set //The set element type : *TxMempoolEntry
	Children *algorithm.Set //The set element type : *TxMempoolEntry
}
