package policy

import (
	beegoUtils "github.com/astaxie/beego/utils"
	"github.com/btcboost/copernicus/algorithm"
)

/**
 * We will instantiate an instance of this class to track transactions that were
 * included in a block. We will lump transactions into a bucket according to
 * their approximate feerate and then track how long it took for those txs to be
 * included in a block.
 *
 * The tracking of unconfirmed (mempool) transactions is completely independent
 * of the historical tracking of transactions that have been confirmed in a
 * block.
 */

type TxConfirmStats struct {
	// Define the buckets we will group transactions into
	// The upper-bound of the range for the bucket (inclusive)
	buckets algorithm.Vector

	// Map of bucket upper-bound to index into all vectors by bucket
	bucketMap beegoUtils.BeeMap

	// For each bucket X:
	// Count the total # of txs in each bucket
	// Track the historical moving average of this total over blocks
	txCtAvg algorithm.Vector

	// and calculate the total for the current block to update the moving
	// average
	curBlockTxCt algorithm.Vector

	// Count the total # of txs confirmed within Y blocks in each bucket
	// Track the historical moving average of theses totals over blocks
	// confAvg[Y][X]
	confAvg algorithm.Vector

	// and calculate the totals for the current block to update the moving
	// averages
	// curBlockConf[Y][X]
	curBlockConf algorithm.Vector

	// Sum the total feerate of all tx's in each bucket
	// Track the historical moving average of this total over blocks

	curBlockVal algorithm.Vector

	// Combine the conf counts with tx counts to calculate the confirmation %
	// for each Y,X. Combine the total value with the tx counts to calculate the
	// avg feerate per bucket
	decay float64

	// Mempool counts of outstanding transactions
	// For each bucket X, track the number of transactions in the mempool that
	// are unconfirmed for each possible confirmation value Y
	// unconfTxs[Y][X]
	unconfTxs algorithm.Vector

	// transactions still unconfirmed after MAX_CONFIRMS for each bucket
	oldUnconfTxs algorithm.Vector
}
