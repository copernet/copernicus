package mining

import (
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/utils"
)

// Container for tracking updates to ancestor feerate as we include (parent)
// transactions in a block
type txMemPoolModifiedEntry struct {
	iter                    *mempool.TxMempoolEntry
	sizeWithAncestors       uint64
	modFeesWithAncestors    utils.Amount
	sigOpCountWithAncestors int64
}

func newTxMemPoolModifiedEntry(entry *mempool.TxMempoolEntry) {
	mEntry := new(txMemPoolModifiedEntry)
	mEntry.iter = entry
	mEntry.sizeWithAncestors = entry.GetsizeWithAncestors()
	mEntry.modFeesWithAncestors = entry.ModFeesWithAncestors
}
