package utxo

import (
	"fmt"
	"testing"
	
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/persist/db"
	"github.com/btcboost/copernicus/util"
)

func TestMain(m *testing.M){
	config := UtxoConfig{do: &db.DBOption{CacheSize: 10000}}
	InitUtxoLruTip(&config)
	m.Run()
}
func TestNewCoinsLruCache(t *testing.T) {
	
	coinsmap := NewEmptyCoinsMap()
	coin := NewEmptyCoin()
	coin.isCoinBase = true
	coin.height = 100
	h := util.HashZero
	op := outpoint.NewOutPoint(h, 0)
	coinsmap.AddCoin(op, coin)
	utxoLruTip.UpdateCoins(coinsmap, &util.HashZero)
	utxoLruTip.Flush()
	c := utxoLruTip.GetCoin(op)
	fmt.Println("c==TestNewCoinsLruCache=====%#v", c)
}

func TestCoinsLruCache_GetCoin(t *testing.T) {
	h := util.HashZero
	op := outpoint.NewOutPoint(h, 0)
	c := utxoLruTip.GetCoin(op)
	fmt.Println("c===TestCoinsLruCache_GetCoin====%#v", c)
}
