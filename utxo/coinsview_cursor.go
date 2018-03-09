package utxo

import (
	"bytes"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/database/boltdb"
	"github.com/btcboost/copernicus/utils"
)

type CoinsViewCursor struct {
	hashBlock utils.Hash
	keyTmp    KeyTmp
	cursor    boltdb.Cursor
}

type KeyTmp struct {
	key      byte
	outPoint *core.OutPoint
}

func (coinsViewCursor *CoinsViewCursor) Valid() bool {
	return coinsViewCursor.keyTmp.key == database.DB_COIN
}

func (coinsViewCursor *CoinsViewCursor) GetKey() *core.OutPoint {
	if coinsViewCursor.keyTmp.key == database.DB_COIN {
		return coinsViewCursor.keyTmp.outPoint
	}
	return nil
}

func (coinsViewCursor *CoinsViewCursor) GetValue() *Coin {
	v := coinsViewCursor.cursor.Value()
	buf := bytes.NewBuffer(v)
	coin, err := DeserializeCoin(buf)
	if err != nil {
		return nil
	}
	return coin
}

func (coinsViewCursor *CoinsViewCursor) Next() {
	coinsViewCursor.cursor.Next()
	//todo CDBIterator logic
	coinEntry := NewCoinEntry(coinsViewCursor.keyTmp.outPoint)
	coinsViewCursor.keyTmp.key = coinEntry.key

}

func (coinsViewCursor *CoinsViewCursor) GetValueSize() int {
	return len(coinsViewCursor.cursor.Value())
}

func (coinsViewCursor *CoinsViewCursor) GetBestBlock() utils.Hash {
	return coinsViewCursor.hashBlock
}

func NewCoinsViewCursor(cursor boltdb.Cursor, hash utils.Hash) *CoinsViewCursor {
	coinsViewCursor := new(CoinsViewCursor)
	coinsViewCursor.hashBlock = hash
	return coinsViewCursor
}
