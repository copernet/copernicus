package mempool

import (
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

/**
 * Fake height value used in Coins to signify they are only in the memory
 * pool(since 0.8)
 */
const (
	MEMPOOL_HEIGHT = 0x7FFFFFFF
)

type Mempool struct {
	CheckFrequency int32
}

func GetTxFromMemPool(hash *utils.Hash) *model.Tx {
	return new(model.Tx)
}

func AllowFreeThreshold() float64 {
	return (float64(utils.COIN) * 144) / 250

}

func AllowFee(priority float64) bool {
	// Large (in bytes) low-priority (new, small-coin) transactions need a fee.
	return priority > AllowFreeThreshold()
}
