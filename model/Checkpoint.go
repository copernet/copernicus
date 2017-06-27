package model

import (
	"github.com/btccom/copernicus/utils"
)

type Checkpoint struct {
	Height int32
	Hash *utils.Hash
}
