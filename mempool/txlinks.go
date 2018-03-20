package mempool

import (
	"github.com/btcboost/copernicus/container"
)

type TxLinks struct {
	Parents  *container.Set //The set element type : *TxMemPoolEntry
	Children *container.Set //The set element type : *TxMemPoolEntry
}
