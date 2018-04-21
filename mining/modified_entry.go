package mining

import (
	"github.com/btcboost/copernicus/mempool"
)

// Container for tracking updates to ancestor feerate as we include (parent)
// transactions in a block
type txMemPoolModifiedEntry struct {
	iter                    *mempool.TxEntry
	sizeWithAncestors       int64
	modFeesWithAncestors    int64
	sigOpCountWithAncestors int64
}

func newTxMemPoolModifiedEntry(entry *mempool.TxEntry) *txMemPoolModifiedEntry {
	mEntry := new(txMemPoolModifiedEntry)
	mEntry.iter = entry
	mEntry.sizeWithAncestors = entry.SumSizeWitAncestors
	mEntry.modFeesWithAncestors = entry.SumFeeWithAncestors
	mEntry.sigOpCountWithAncestors = entry.SumSigOpCountWithAncestors
	return mEntry
}
