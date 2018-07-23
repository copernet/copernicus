package model

import "github.com/copernet/copernicus/util"

type Checkpoint struct {
	Height int32
	Hash   *util.Hash
}
