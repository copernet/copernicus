package mempool

import "github.com/btcboost/copernicus/utils"

type PriorityFeeDelta struct {
	PriorityDelta float64
	Fee           utils.Amount
}
