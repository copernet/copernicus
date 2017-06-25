package model

import (
	"copernicus/utils"
)

type Checkpoint struct {
	Height int32
	Hash *utils.Hash
}
