package model

import "github.com/btcboost/copernicus/util"

type Checkpoint struct {
	Height int32
	Hash   *util.Hash
}