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
	buckets *algorithm.Vector

	// Map of bucket upper-bound to index into all vectors by bucket
	bucketMap beegoUtils.BeeMap

	// For each bucket X:
	// Count the total # of txs in each bucket
	// Track the historical moving average of this total over blocks
	txCtAvg *algorithm.Vector

	// and calculate the total for the current block to update the moving
	// average
	curBlockTxCt *algorithm.Vector

	// Count the total # of txs confirmed within Y blocks in each bucket
	// Track the historical moving average of theses totals over blocks
	// confAvg[Y][X]
	confAvg *algorithm.Vector

	// and calculate the totals for the current block to update the moving
	// averages
	// curBlockConf[Y][X]
	curBlockConf *algorithm.Vector

	// Sum the total feerate of all tx's in each bucket
	// Track the historical moving average of this total over blocks
	avg *algorithm.Vector

	// Sum the total feerate of all tx's in each bucket
	// Track the historical moving average of this total over blocks

	curBlockVal *algorithm.Vector

	// Combine the conf counts with tx counts to calculate the confirmation %
	// for each Y,X. Combine the total value with the tx counts to calculate the
	// avg feerate per bucket
	decay float64

	// Mempool counts of outstanding transactions
	// For each bucket X, track the number of transactions in the mempool that
	// are unconfirmed for each possible confirmation value Y
	// unconfTxs[Y][X]
	unconfTxs *algorithm.Vector

	// transactions still unconfirmed after MAX_CONFIRMS for each bucket
	oldUnconfTxs *algorithm.Vector
}

func NewTxConfirmStats(defaultBuckets algorithm.Vector, maxConfirms int, decay float64) *TxConfirmStats {
	txConfirmStats := TxConfirmStats{}
	txConfirmStats.decay = decay
	txConfirmStats.buckets = algorithm.NewVector()
	for i := 0; i < defaultBuckets.Size(); i++ {
		bucket, _ := defaultBuckets.At(i)
		txConfirmStats.buckets.PushBack(bucket)
		txConfirmStats.bucketMap.Set(bucket, i)
	}
	txConfirmStats.confAvg = algorithm.NewVector()
	txConfirmStats.curBlockConf = algorithm.NewVector()
	txConfirmStats.unconfTxs = algorithm.NewVector()
	for j := 0; j < maxConfirms; j++ {
		txConfirmStats.confAvg.PushBack(algorithm.NewVector())
		txConfirmStats.curBlockConf.PushBack(algorithm.NewVector())
		txConfirmStats.unconfTxs.PushBack(algorithm.NewVector())
	}

	txConfirmStats.oldUnconfTxs = algorithm.NewVector()
	txConfirmStats.curBlockTxCt = algorithm.NewVector()
	txConfirmStats.txCtAvg = algorithm.NewVector()
	txConfirmStats.curBlockVal = algorithm.NewVector()
	txConfirmStats.avg = algorithm.NewVector()
	return &txConfirmStats
}

func (txCOnfirmStats *TxConfirmStats) ClearCurrent(blockHeight uint) {
	for j := 0; j < txCOnfirmStats.buckets.Size(); j++ {

		oldUnconfTxNum := txCOnfirmStats.oldUnconfTxs.Array[j].(int)
		unconfTxVec := txCOnfirmStats.unconfTxs.Array[int(blockHeight)%txCOnfirmStats.unconfTxs.Size()].(*algorithm.Vector)
		unconfNum := unconfTxVec.Array[j].(int)
		txCOnfirmStats.oldUnconfTxs.SetValueByIndex(j, oldUnconfTxNum+unconfNum)
		unconfTxVec.SetValueByIndex(j, 0)

		for i := 0; i < txCOnfirmStats.curBlockConf.Size(); i++ {
			curBlockTmp := txCOnfirmStats.curBlockConf.Array[i].(*algorithm.Vector)
			curBlockTmp.SetValueByIndex(j, 0)
		}

		txCOnfirmStats.curBlockVal.SetValueByIndex(j, 0)
		txCOnfirmStats.curBlockTxCt.SetValueByIndex(j, 0)
	}
}

func (txCOnfirmStats *TxConfirmStats) Record(blocksToConfirm int, val float64) {
	if blocksToConfirm < 1 {
		return
	}

	bucketindex := txCOnfirmStats.bucketMap.Get(val).(uint)
	for i := blocksToConfirm; i <= txCOnfirmStats.curBlockConf.Size(); i++ {
		curBlockConfTmp := txCOnfirmStats.curBlockConf.Array[i-1].(*algorithm.Vector)
		num := curBlockConfTmp.Array[bucketindex].(int)
		curBlockConfTmp.SetValueByIndex(int(bucketindex), num+1)
	}

	curTxCt := txCOnfirmStats.curBlockTxCt.Array[bucketindex].(int)
	txCOnfirmStats.curBlockTxCt.SetValueByIndex(int(bucketindex), curTxCt+1)
	curVal := txCOnfirmStats.curBlockVal.Array[bucketindex].(float64)
	txCOnfirmStats.curBlockTxCt.SetValueByIndex(int(bucketindex), curVal+val)
}

func (txCOnfirmStats *TxConfirmStats) UpdateMovingAverages() {
	for j := 0; j < txCOnfirmStats.buckets.Size(); j++ {
		for i := 0; i < txCOnfirmStats.confAvg.Size(); i++ {
			confAvgVecTmp := txCOnfirmStats.confAvg.Array[i].(*algorithm.Vector)
			confAvgNum := confAvgVecTmp.Array[j].(float64)
			curConfVecTmp := txCOnfirmStats.curBlockConf.Array[i].(*algorithm.Vector)
			curConfNum := curConfVecTmp.Array[j].(int)

			confAvgVecTmp.SetValueByIndex(j, confAvgNum*txCOnfirmStats.decay+float64(curConfNum))
		}
		curValNum := txCOnfirmStats.curBlockTxCt.Array[j].(int)
		avgNum := txCOnfirmStats.avg.Array[j].(float64)
		txCOnfirmStats.avg.SetValueByIndex(j, avgNum*txCOnfirmStats.decay+float64(curValNum))

		curTxCtNum := txCOnfirmStats.curBlockTxCt.Array[j].(int)
		txCtAvgNum := txCOnfirmStats.txCtAvg.Array[j].(float64)
		txCOnfirmStats.txCtAvg.SetValueByIndex(j, txCtAvgNum*txCOnfirmStats.decay+float64(curTxCtNum))
	}
}

func (txCOnfirmStats *TxConfirmStats) GetMaxConfirms() int {
	return txCOnfirmStats.confAvg.Size()
}

//EstimateMedianVal returns -1 on error conditions
func (txCOnfirmStats *TxConfirmStats) EstimateMedianVal(confTarget int, sufficientTxVal,
	successBreakPoint float64, requireGreater bool, nBlockHeight uint) float64 {

	// Counters for a bucket (or range of buckets)，Number of tx's confirmed within the confTarget
	nConf := 0.0
	// Total number of tx's that were ever confirmed
	totalNum := 0.0
	// Number of tx's still in mempool for confTarget or longer
	extraNum := 0
	maxBucketInex := txCOnfirmStats.buckets.Size() - 1
	startBucket := uint(0)
	step := 1

	// We'll combine buckets until we have enough samples.
	// The near and far variables will define the range we've combined
	// The best variables are the last range we saw which still had a high
	// enough confirmation rate to count as success.
	// The cur variables are the current range we're counting.
	curNearBucket := startBucket
	bestNearBucket := startBucket
	curFarBucket := startBucket
	bestFarBucket := startBucket
	foundAnswer := false
	bins := uint(txCOnfirmStats.unconfTxs.Size())

	// requireGreater means we are looking for the lowest feerate such that all
	// higher values pass, so we start at maxbucketindex (highest feerate) and
	// look at successively smaller buckets until we reach failure. Otherwise,
	// we are looking for the highest feerate such that all lower values fail,
	// and we go in the opposite direction.
	if requireGreater {
		startBucket = uint(maxBucketInex)
		step = -1
	}

	for bucket := startBucket; bucket >= 0 && bucket <= uint(maxBucketInex); bucket += uint(step) {
		curFarBucket = bucket
		confAvgVecTmp := txCOnfirmStats.confAvg.Array[confTarget-1].(*algorithm.Vector)
		nConf += confAvgVecTmp.Array[bucket].(float64)
		totalNum += txCOnfirmStats.txCtAvg.Array[bucket].(float64)

		confct := uint(confTarget)
		for ; confct < uint(txCOnfirmStats.GetMaxConfirms()); confct++ {
			unconfTxsVecTmp := txCOnfirmStats.unconfTxs.Array[(nBlockHeight-confct)%bins].(*algorithm.Vector)
			extraNum += unconfTxsVecTmp.Array[bucket].(int)
		}
		extraNum += txCOnfirmStats.oldUnconfTxs.Array[bucket].(int)

		// If we have enough transaction data points in this range of buckets,
		// we can test for success (Only count the confirmed data points, so
		// that each confirmation count will be looking at the same amount of
		// data and same bucket breaks)
		if totalNum >= sufficientTxVal/(1-txCOnfirmStats.decay) {
			curPct := nConf / (totalNum + float64(extraNum))

			// Check to see if we are no longer getting confirmed at the success rate
			if requireGreater && curPct < successBreakPoint {
				break
			}
			if !requireGreater && curPct > successBreakPoint {
				break
			}

			// Otherwise update the cumulative stats, and the bucket variables
			// and reset the counters
			foundAnswer = true
			nConf = 0
			totalNum = 0
			extraNum = 0
			bestFarBucket = curFarBucket
			bestNearBucket = curNearBucket
			curNearBucket = bucket + uint(step)
		}
	}

	median := -1.0
	txSum := 0.0

	// Calculate the "average" feerate of the best bucket range that met success
	// conditions. Find the bucket with the median transaction and then report
	// the average feerate from that bucket. This is a compromise between
	// finding the median which we can't since we don't save all tx's and
	// reporting the average which is less accurate
	minBucket := bestFarBucket
	maxBucket := bestFarBucket

	if bestNearBucket > bestFarBucket {
		minBucket = bestNearBucket
		maxBucket = bestNearBucket
	}

	for i := minBucket; i <= maxBucket; i++ {
		txSum += txCOnfirmStats.txCtAvg.Array[i].(float64)
	}

	if foundAnswer && txSum != 0 {
		txSum = txSum / 2
		for j := minBucket; j <= maxBucket; j++ {
			if txCOnfirmStats.txCtAvg.Array[j].(float64) < txSum {
				txSum -= txCOnfirmStats.txCtAvg.Array[j].(float64)
			} else {
				median = txCOnfirmStats.avg.Array[j].(float64) / txCOnfirmStats.txCtAvg.Array[j].(float64)
				break
			}
		}
	}

	return median
}
