package utxo

import (
	"bytes"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/utils"
)

type CoinViewDB struct {
	database.DBWrapper
	bucketKey string
}

func (coinViewDB *CoinViewDB) GetCoin(outpoint *core.OutPoint) (coin *Coin) {
	var v []byte

	buf := bytes.NewBuffer(v)
	coin, err := DeserializeCoin(buf)
	if err != nil {
		return nil
	}
	return coin
}

func (coinViewDB *CoinViewDB) HaveCoin(outpoint *core.OutPoint) bool {
	return false

}

func (coinViewDB *CoinViewDB) SetBestBlock(hash *utils.Hash) {
	//todo:not finish
}

func (coinViewDB *CoinViewDB) GetBestBlock() utils.Hash {
	var v []byte
	hash := utils.Hash{}
	//todo:not finish
	hash.SetBytes(v)
	return hash
}

func (coinViewDB *CoinViewDB) BatchWrite(mapCoins map[core.OutPoint]CoinsCacheEntry) (bool, error) {
	return true, nil
}

func (coinViewDB *CoinViewDB) EstimateSize() int {
	var size int
	//todo:not finish
	return size
}

//
//func (coinViewDB *CoinViewDB) Cursor() *CoinsViewCursor {
//
//}

func NewCoinViewDB() *CoinViewDB {
	coinViewDB := new(CoinViewDB)

	return coinViewDB
}
