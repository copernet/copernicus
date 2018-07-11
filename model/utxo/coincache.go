package utxo

import (
	"github.com/copernet/copernicus/log"
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
		log.Error("utxoTip has not init!!")
	}
	return utxoTip
}

type CacheView interface {
	GetCoin(outpoint *outpoint.OutPoint) *Coin
	HaveCoin(point *outpoint.OutPoint) bool
	GetBestBlock() (util.Hash, error)
	SetBestBlock(hash util.Hash)
	UpdateCoins(tempCacheCoins *CoinsMap, hash *util.Hash) error
	DynamicMemoryUsage() int64
	GetCacheSize() int
	Flush() bool
}
