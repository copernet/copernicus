package utxo

import (
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/util"

	"github.com/copernet/copernicus/persist/db"
)

var utxoTip CacheView

type UtxoConfig struct {
	Do *db.DBOption
}

func GetUtxoCacheInstance() CacheView {
	if utxoTip == nil {
		panic("utxoTip has not init!!")
	}
	return utxoTip
}

// Close FIXME: this is only for test. We must do it in a graceful way
func Close() {
	utxoTip = nil
}

type CacheView interface {
	GetCoin(outpoint *outpoint.OutPoint) *Coin
	HaveCoin(point *outpoint.OutPoint) bool
	GetBestBlock() (util.Hash, error)
	UpdateCoins(tempCacheCoins *CoinsMap, hash *util.Hash) error
	DynamicMemoryUsage() int64
	GetCacheSize() int
	RemoveCoins(outpoint *outpoint.OutPoint)
	Flush() bool
}
