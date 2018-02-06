package utxo

import (
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
)

type CoinsViewCursor struct {
	hashBlock utils.Hash
}

func (coinsViewCursor *CoinsViewCursor) Valid() bool {
	return true
}

func (coinsViewCursor *CoinsViewCursor) GetKey() *model.OutPoint {
	return nil
}

func (coinsViewCursor *CoinsViewCursor) GetValue() *Coin {
	return nil
}

func (coinsViewCursor *CoinsViewCursor) Next() {

}

func (coinsViewCursor *CoinsViewCursor) GetValueSize() int {
	return 0
}

func (coinsViewCursor *CoinsViewCursor) GetBestBlock() utils.Hash {
	return coinsViewCursor.hashBlock
}

func NewCoinsViewCursor(hash utils.Hash) *CoinsViewCursor {
	coinsViewCursor := new(CoinsViewCursor)
	coinsViewCursor.hashBlock = hash
	return coinsViewCursor
}
