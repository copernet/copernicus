package utxo

import "github.com/btcboost/copernicus/utils"

type CoinsViewCursor struct {
	hashBlock utils.Hash
}

func NewCoinsViewCursor(hash utils.Hash) *CoinsViewCursor {
	coinsViewCursor := new(CoinsViewCursor)
	coinsViewCursor.hashBlock = hash
	return coinsViewCursor
}
