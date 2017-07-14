package model

import (
	"github.com/btcboost/copernicus/utils"
)

type Checkpoint struct {
	Height int32
	Hash   *utils.Hash
}
