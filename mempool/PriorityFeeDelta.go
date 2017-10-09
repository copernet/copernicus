package mempool

import "github.com/btcboost/copernicus/btcutil"

type PriorityFeeDelta struct {
	priorityDelta float64
	fee           btcutil.Amount
}
