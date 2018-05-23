package utxo

import (
	"fmt"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/persist/db"
	"github.com/btcboost/copernicus/util"
	"testing"
)

func TestNewCoinsLruCache(t *testing.T) {
	config := UtxoConfig{do: &db.DBOption{CacheSize: 10000}}

	InitUtxoLruTip(&config)
	coinsmap := make(CoinsMap)
	coin := NewEmptyCoin()
	h := util.HashZero
	op := outpoint.NewOutPoint(h, 0)
	coinsmap.AddCoin(op, coin)
	utxoLruTip.UpdateCoins(&coinsmap, &util.HashZero)
	utxoLruTip.Flush()
	c := utxoLruTip.GetCoin(op)
	fmt.Println("c=======%#v", c)
}

func TestCoinsLruCache_GetCoin(t *testing.T) {
	config := UtxoConfig{do: &db.DBOption{CacheSize: 10000}}

	InitUtxoLruTip(&config)
	h := util.HashZero
	op := outpoint.NewOutPoint(h, 0)
	c := utxoLruTip.GetCoin(op)
	fmt.Println("c=======%#v", c)
}
