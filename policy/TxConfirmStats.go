package policy

import (
	"encoding/binary"
	"io"

	beegoUtils "github.com/astaxie/beego/utils"
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
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

func NewTxConfirmStats(defaultBuckets algorithm.Vector, maxConfirms uint, decay float64) *TxConfirmStats {
	txConfirmStats := TxConfirmStats{}
	txConfirmStats.decay = decay
	txConfirmStats.buckets = algorithm.NewVector()
	txConfirmStats.bucketMap = *beegoUtils.NewBeeMap()
	for i := 0; i < defaultBuckets.Size(); i++ {
		bucket, _ := defaultBuckets.At(i)
		txConfirmStats.buckets.PushBack(bucket)
		txConfirmStats.bucketMap.Set(bucket, i)
	}
	txConfirmStats.confAvg = algorithm.NewVector()
	txConfirmStats.curBlockConf = algorithm.NewVector()
	txConfirmStats.unconfTxs = algorithm.NewVector()
	for j := uint(0); j < maxConfirms; j++ {
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

func (txConfirmStats *TxConfirmStats) ClearCurrent(blockHeight uint) {
	for j := 0; j < txConfirmStats.buckets.Size(); j++ {

		oldUnconfTxNum := txConfirmStats.oldUnconfTxs.Array[j].(int)
		unconfTxVec := txConfirmStats.unconfTxs.Array[int(blockHeight)%txConfirmStats.unconfTxs.Size()].(*algorithm.Vector)
		unconfNum := unconfTxVec.Array[j].(int)
		txConfirmStats.oldUnconfTxs.SetValueByIndex(j, oldUnconfTxNum+unconfNum)
		unconfTxVec.SetValueByIndex(j, 0)

		for i := 0; i < txConfirmStats.curBlockConf.Size(); i++ {
			curBlockTmp := txConfirmStats.curBlockConf.Array[i].(*algorithm.Vector)
			curBlockTmp.SetValueByIndex(j, 0)
		}

		txConfirmStats.curBlockVal.SetValueByIndex(j, 0)
		txConfirmStats.curBlockTxCt.SetValueByIndex(j, 0)
	}
}

func (txConfirmStats *TxConfirmStats) Record(blocksToConfirm int, val float64) {
	if blocksToConfirm < 1 {
		return
	}

	bucketindex := txConfirmStats.bucketMap.Get(val).(uint)
	for i := blocksToConfirm; i <= txConfirmStats.curBlockConf.Size(); i++ {
		curBlockConfTmp := txConfirmStats.curBlockConf.Array[i-1].(*algorithm.Vector)
		num := curBlockConfTmp.Array[bucketindex].(int)
		curBlockConfTmp.SetValueByIndex(int(bucketindex), num+1)
	}

	curTxCt := txConfirmStats.curBlockTxCt.Array[bucketindex].(int)
	txConfirmStats.curBlockTxCt.SetValueByIndex(int(bucketindex), curTxCt+1)
	curVal := txConfirmStats.curBlockVal.Array[bucketindex].(float64)
	txConfirmStats.curBlockTxCt.SetValueByIndex(int(bucketindex), curVal+val)
}

func (txConfirmStats *TxConfirmStats) UpdateMovingAverages() {
	for j := 0; j < txConfirmStats.buckets.Size(); j++ {
		for i := 0; i < txConfirmStats.confAvg.Size(); i++ {
			confAvgVecTmp := txConfirmStats.confAvg.Array[i].(*algorithm.Vector)
			confAvgNum := confAvgVecTmp.Array[j].(float64)
			curConfVecTmp := txConfirmStats.curBlockConf.Array[i].(*algorithm.Vector)
			curConfNum := curConfVecTmp.Array[j].(int)

			confAvgVecTmp.SetValueByIndex(j, confAvgNum*txConfirmStats.decay+float64(curConfNum))
		}
		curValNum := txConfirmStats.curBlockTxCt.Array[j].(int)
		avgNum := txConfirmStats.avg.Array[j].(float64)
		txConfirmStats.avg.SetValueByIndex(j, avgNum*txConfirmStats.decay+float64(curValNum))

		curTxCtNum := txConfirmStats.curBlockTxCt.Array[j].(int)
		txCtAvgNum := txConfirmStats.txCtAvg.Array[j].(float64)
		txConfirmStats.txCtAvg.SetValueByIndex(j, txCtAvgNum*txConfirmStats.decay+float64(curTxCtNum))
	}
}

func (txConfirmStats *TxConfirmStats) GetMaxConfirms() uint {
	return uint(txConfirmStats.confAvg.Size())
}

//EstimateMedianVal returns -1 on error conditions
func (txConfirmStats *TxConfirmStats) EstimateMedianVal(confTarget int, sufficientTxVal,
	successBreakPoint float64, requireGreater bool, nBlockHeight uint) float64 {

	// Counters for a bucket (or range of buckets)，Number of tx's confirmed within the confTarget
	nConf := 0.0
	// Total number of tx's that were ever confirmed
	totalNum := 0.0
	// Number of tx's still in mempool for confTarget or longer
	extraNum := 0
	maxBucketInex := txConfirmStats.buckets.Size() - 1
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
	bins := uint(txConfirmStats.unconfTxs.Size())

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
		confAvgVecTmp := txConfirmStats.confAvg.Array[confTarget-1].(*algorithm.Vector)
		nConf += confAvgVecTmp.Array[bucket].(float64)
		totalNum += txConfirmStats.txCtAvg.Array[bucket].(float64)

		confct := uint(confTarget)
		for ; confct < txConfirmStats.GetMaxConfirms(); confct++ {
			unconfTxsVecTmp := txConfirmStats.unconfTxs.Array[(nBlockHeight-confct)%bins].(*algorithm.Vector)
			extraNum += unconfTxsVecTmp.Array[bucket].(int)
		}
		extraNum += txConfirmStats.oldUnconfTxs.Array[bucket].(int)

		// If we have enough transaction data points in this range of buckets,
		// we can test for success (Only count the confirmed data points, so
		// that each confirmation count will be looking at the same amount of
		// data and same bucket breaks)
		if totalNum >= sufficientTxVal/(1-txConfirmStats.decay) {
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
		txSum += txConfirmStats.txCtAvg.Array[i].(float64)
	}

	if foundAnswer && txSum != 0 {
		txSum = txSum / 2
		for j := minBucket; j <= maxBucket; j++ {
			if txConfirmStats.txCtAvg.Array[j].(float64) < txSum {
				txSum -= txConfirmStats.txCtAvg.Array[j].(float64)
			} else {
				median = txConfirmStats.avg.Array[j].(float64) / txConfirmStats.txCtAvg.Array[j].(float64)
				break
			}
		}
	}

	return median
}

func (txConfirmStats *TxConfirmStats) NewTx(nBlockHeight uint, val float64) uint {
	bucketIndex := txConfirmStats.bucketMap.Get(val).(uint)
	blockINdex := nBlockHeight % uint(txConfirmStats.unconfTxs.Size())
	unconfxVecTmp := txConfirmStats.unconfTxs.Array[blockINdex].(*algorithm.Vector)
	unconfxVecTmp.SetValueByIndex(int(bucketIndex), unconfxVecTmp.Array[bucketIndex].(int)+1)
	return bucketIndex

}

func (txConfirmStats *TxConfirmStats) RemoveTx(entryHeight, nBestSeenHeight, bucketIndex uint) {
	// nBestSeenHeight is not updated yet for the new block
	blocksAgo := int(nBestSeenHeight - entryHeight)
	if nBestSeenHeight == 0 {
		blocksAgo = 0
	}
	if blocksAgo < 0 {
		return
	}

	if blocksAgo >= txConfirmStats.unconfTxs.Size() {
		if txConfirmStats.oldUnconfTxs.Array[bucketIndex].(int) > 0 {
			txConfirmStats.oldUnconfTxs.Array[bucketIndex] = txConfirmStats.oldUnconfTxs.Array[bucketIndex].(int) - 1
		}
	} else {
		blockIndex := entryHeight % uint(txConfirmStats.unconfTxs.Size())
		unconfTxVecTmp := txConfirmStats.unconfTxs.Array[blockIndex].(*algorithm.Vector)
		if unconfTxVecTmp.Array[bucketIndex].(int) > 0 {
			unconfTxVecTmp.Array[bucketIndex] = unconfTxVecTmp.Array[bucketIndex].(int) - 1
		}
	}
}

func writeForVector(writer io.Writer, vector *algorithm.Vector) error {
	utils.WriteVarInt(writer, uint64(vector.Size()))
	for _, v := range vector.Array {
		err := binary.Write(writer, binary.LittleEndian, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (txConfirmStats *TxConfirmStats) Serialize(writer io.Writer) error {
	err := binary.Write(writer, binary.LittleEndian, txConfirmStats.decay)
	if err != nil {
		return err
	}

	err = writeForVector(writer, txConfirmStats.buckets)
	if err != nil {
		return err
	}
	err = writeForVector(writer, txConfirmStats.avg)
	if err != nil {
		return err
	}
	err = writeForVector(writer, txConfirmStats.txCtAvg)
	if err != nil {
		return err
	}

	utils.WriteVarInt(writer, uint64(txConfirmStats.confAvg.Size()))
	for _, v := range txConfirmStats.confAvg.Array {
		err = writeForVector(writer, v.(*algorithm.Vector))
		if err != nil {
			return err
		}
	}
	return nil
}

func readForVector(reader io.Reader, vector *algorithm.Vector) error {
	size, err := utils.ReadVarInt(reader)
	if err != nil {
		return err
	}
	var element float64
	for i := uint64(0); i < size; i++ {
		err = binary.Read(reader, binary.LittleEndian, &element)
		if err != nil {
			return err
		}
		vector.PushBack(element)
	}
	return nil
}

func (txConfirmStats *TxConfirmStats) Deserialize(reader io.Reader) error {
	var fileDecay float64
	fileBuckets := algorithm.NewVector()
	fileAvg := algorithm.NewVector()
	fileTxCtAvg := algorithm.NewVector()
	fileConfAvg := algorithm.NewVector()

	// Read data file into temporary variables and do some very basic sanity
	// checking
	err := binary.Read(reader, binary.LittleEndian, &fileDecay)
	if err != nil {
		return err
	}
	if fileDecay <= 0 || fileDecay >= 1 {
		return errors.New("Corrupt estimates file. Decay must be between 0 and 1 (non-inclusive)")
	}

	err = readForVector(reader, fileBuckets)
	if err != nil {
		return err
	}
	numBuckets := fileBuckets.Size()
	if numBuckets <= 1 || numBuckets > 1000 {
		return errors.New("Corrupt estimates file. Must have between 2 and 1000 feerate buckets")
	}

	err = readForVector(reader, fileAvg)
	if err != nil {
		return err
	}
	if fileAvg.Size() != numBuckets {
		return errors.New("Corrupt estimates file. Mismatch in feerate average bucket count")
	}

	err = readForVector(reader, fileTxCtAvg)
	if err != nil {
		return err
	}
	if fileTxCtAvg.Size() != numBuckets {
		return errors.New("Corrupt estimates file. Mismatch in tx count bucket count")
	}

	size, err := utils.ReadVarInt(reader)
	if err != nil {
		return err
	}
	for i := uint64(0); i < size; i++ {
		tmpVector := algorithm.NewVector()
		err = readForVector(reader, tmpVector)
		if err != nil {
			return err
		}
		fileConfAvg.PushBack(tmpVector)
	}
	maxConfirms := fileConfAvg.Size()
	if maxConfirms <= 0 || maxConfirms > 6*24*7 {
		return errors.New("Corrupt estimates file.  Must maintain" +
			"estimates for between 1 and 1008 (one week) confirms")
	}

	for i := 0; i < maxConfirms; i++ {
		if fileConfAvg.Array[i].(*algorithm.Vector).Size() != numBuckets {
			return errors.New("Corrupt estimates file. Mismatch in feerate conf average bucket count")
		}
	}

	// Now that we've processed the entire feerate estimate data file and not
	// thrown any errors, we can copy it to our data structures
	txConfirmStats.decay = fileDecay
	txConfirmStats.buckets = fileBuckets
	txConfirmStats.avg = fileAvg
	txConfirmStats.confAvg = fileConfAvg
	txConfirmStats.txCtAvg = fileTxCtAvg
	txConfirmStats.bucketMap = *beegoUtils.NewBeeMap()

	// Resize the current block variables which aren't stored in the data file
	// to match the number of confirms and buckets
	txConfirmStats.curBlockConf = algorithm.NewVector()
	txConfirmStats.unconfTxs = algorithm.NewVector()
	for i := 0; i < maxConfirms; i++ {
		txConfirmStats.curBlockConf.PushBack(algorithm.NewVector())
		txConfirmStats.unconfTxs.PushBack(algorithm.NewVector())
	}
	txConfirmStats.curBlockTxCt = algorithm.NewVector()
	txConfirmStats.curBlockVal = algorithm.NewVector()
	txConfirmStats.oldUnconfTxs = algorithm.NewVector()
	for i := 0; i < txConfirmStats.buckets.Size(); i++ {
		txConfirmStats.bucketMap.Set(txConfirmStats.buckets.Array[i], i)
	}

	return nil
}
