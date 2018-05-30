package utxo

import (
	
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/util"
	
	"github.com/btcboost/copernicus/persist/db"
)

var utxoTip CacheView

type UtxoConfig struct {
	Do *db.DBOption
}

// func InitUtxoTip(uc *UtxoConfig) {
// 	fmt.Printf("initUtxo processing ....%v", uc)
//
// 	db := NewCoinsDB(uc.do)
// 	utxoTip = NewCoinCache(*db)
//
// }

func GetUtxoCacheInstance() CacheView {
	if utxoLruTip == nil {
		log.Error("utxoTip has not init!!")
	}
	return utxoLruTip
}

type CacheView interface {
	GetCoin(outpoint *outpoint.OutPoint) *Coin
	HaveCoin(point *outpoint.OutPoint) bool
	GetBestBlock() util.Hash
	SetBestBlock(hash util.Hash)
	UpdateCoins(tempCacheCoins *CoinsMap, hash *util.Hash) error
	DynamicMemoryUsage() int64
	GetCacheSize() int
	Flush() bool
}

