package mempool

import "github.com/btcboost/copernicus/algorithm"

type TxLinks struct {
	Parents  *algorithm.Vector
	Children *algorithm.Vector
}
