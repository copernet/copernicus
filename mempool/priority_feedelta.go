package mempool

import "github.com/btcboost/copernicus/btcutil"

type PriorityFeeDelta struct {
	PriorityDelta float64
	Fee           btcutil.Amount
}
