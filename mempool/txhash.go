package mempool

import "github.com/btcboost/copernicus/utils"

type TxHash struct {
	hash  utils.Hash
	entry *TxMempoolEntry
}
