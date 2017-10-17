package mempool

import (
	"github.com/btcboost/copernicus/utils"
)

type refOutPoint struct {
	Hash  utils.Hash
	Index uint32
}

func CompareByRefOutPoint(a, b interface{}) bool {
	comA := a.(refOutPoint)
	comB := b.(refOutPoint)
	cmp := comA.Hash.Cmp(&comB.Hash)
	if cmp < 0 || (cmp == 0 && comA.Index < comB.Index) {
		return true
	}
	return false
}
