package utxo

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

type CoinsViewCursor struct {
	hashBlock utils.Hash
	keyTmp    KeyTmp
}

type KeyTmp struct {
	key      byte
	outPoint *core.OutPoint
}
