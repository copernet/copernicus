package utxo

import (
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type CoinView struct {
	hashBlock        utils.Hash
	cacheCoins       map[model.OutPoint]*CoinsCacheEntry
	cachedCoinsUsage int32
}
