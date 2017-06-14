package model

import "copernicus/crypto"

type Checkpoint struct {
	Height int32
	Hash *crypto.Hash
}
