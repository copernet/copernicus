package model

import "crypto"

type Checkpoint struct {
	Height int32
	Hash crypto.Hash
}
