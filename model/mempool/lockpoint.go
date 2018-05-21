package mempool

import (
	"github.com/btcboost/copernicus/model/blockindex"
)

type LockPoints struct {
	// Height and Time will be set to the blockChain height and median time past values that
	// would be necessary to satisfy all relative lockTime constraints (BIP68)
	// of this tx given our view of block chain history
	Height int
	Time   int64
	// MaxInputBlock as long as the current chain descends from the highest height block
	// containing one of the inputs used in the calculation, then the cached
	// values are still valid even after a reOrg.
	MaxInputBlock *blockindex.BlockIndex
}

func NewLockPoints() *LockPoints {
	lockPoints := LockPoints{}
	return &lockPoints
}

