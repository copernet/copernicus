package mempool

import (
	"github.com/btcboost/copernicus/blockchain"
)

type LockPoints struct {
	// Will be set to the blockchain height and median time past values that
	// would be necessary to satisfy all relative locktime constraints (BIP68)
	// of this tx given our view of block chain history
	Height int
	Time   int64
	// As long as the current chain descends from the highest height block
	// containing one of the inputs used in the calculation, then the cached
	// values are still valid even after a reorg.
	maxInputBlock *blockchain.BlockIndex
}
