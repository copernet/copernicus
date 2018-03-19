package policy

import (
	"encoding/binary"
	"io"

	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
)

/**
 * We will instantiate an instance of this class to track transactions that were
 * included in a block. We will lump transactions into a bucket according to
 * their approximate feeRate and then track how long it took for those txs to be
 * included in a block.
 *
 * The tracking of unconfirmed (memPool) transactions is completely independent
 * of the historical tracking of transactions that have been confirmed in a
 * block.
 */

type TxConfirmStats struct {
	// Define the buckets we will group transactions into
	// The upper-bound of the range for the bucket (inclusive)
	buckets *container.Vector

	// Map of bucket upper-bound to index into all vectors by bucket
	bucketMap container.BeeMap

	// For each bucket X:
	// Count the total # of txs in each bucket
	// Track the historical moving average of this total over blocks
	txCtAvg *container.Vector

	// and calculate the total for the current block to update the moving
	// average
	curBlockTxCt *container.Vector

	// Count the total # of txs confirmed within Y blocks in each bucket
	// Track the historical moving average of theses totals over blocks
	// confAvg[Y][X]
	confAvg *container.Vector

	// and calculate the totals for the current block to update the moving
	// averages
	// curBlockConf[Y][X]
	curBlockConf *container.Vector

	// Sum the total feerate of all tx's in each bucket
	// Track the historical moving average of this total over blocks
	avg *container.Vector

	// Sum the total feerate of all tx's in each bucket
	// Track the historical moving average of this total over blocks

	curBlockVal *container.Vector

	// Combine the conf counts with tx counts to calculate the confirmation %
	// for each Y,X. Combine the total value with the tx counts to calculate the
	// avg feerate per bucket
	decay float64

	// Mempool counts of outstanding transactions
	// For each bucket X, track the number of transactions in the mempool that
	// are unconfirmed for each possible confirmation value Y
	// unconfTxs[Y][X]
	unconfTxs *container.Vector

	// transactions still unconfirmed after MAX_CONFIRMS for each bucket
	oldUnconfTxs *container.Vector
}

const (
	FLOAT64TYPE = iota
	INTTYPE
	VECTORPTRTYPE
)

/*NewTxConfirmStats Initialize the data structures. This is called by BlockPolicyEstimator's
 * constructor with default values.
 * @param defaultBuckets contains the upper limits for the bucket boundaries
 * @param maxConfirms max number of confirms to track
 * @param decay how much to decay the historical moving average per block
 */
func NewTxConfirmStats(defaultBuckets *container.Vector, maxConfirms uint, decay float64) *TxConfirmStats {
	txConfirmStats := TxConfirmStats{}
	txConfirmStats.decay = decay
	txConfirmStats.buckets = container.NewVector()
	txConfirmStats.bucketMap = *container.NewBeeMap()

	for i := 0; i < defaultBuckets.Size(); i++ {
		bucket, _ := defaultBuckets.At(i)
		txConfirmStats.buckets.PushBack(bucket)
		txConfirmStats.bucketMap.Set(bucket, i)
	}
	txConfirmStats.confAvg = container.NewVectorWithSize(maxConfirms)
	txConfirmStats.curBlockConf = container.NewVectorWithSize(maxConfirms)
	txConfirmStats.unconfTxs = container.NewVectorWithSize(maxConfirms)
	for j := uint(0); j < maxConfirms; j++ {
		txConfirmStats.confAvg.SetValueByIndex(int(j), container.NewVectorWithSize(uint(txConfirmStats.buckets.Size())))
		setValue(txConfirmStats.confAvg.Array[j].(*container.Vector), FLOAT64TYPE)
		txConfirmStats.curBlockConf.SetValueByIndex(int(j), container.NewVectorWithSize(uint(txConfirmStats.buckets.Size())))
		setValue(txConfirmStats.curBlockConf.Array[j].(*container.Vector), INTTYPE)
		txConfirmStats.unconfTxs.SetValueByIndex(int(j), container.NewVectorWithSize(uint(txConfirmStats.buckets.Size())))
		setValue(txConfirmStats.unconfTxs.Array[j].(*container.Vector), INTTYPE)
	}

	txConfirmStats.oldUnconfTxs = container.NewVectorWithSize(uint(txConfirmStats.buckets.Size()))
	setValue(txConfirmStats.oldUnconfTxs, INTTYPE)
	txConfirmStats.curBlockTxCt = container.NewVectorWithSize(uint(txConfirmStats.buckets.Size()))
	setValue(txConfirmStats.curBlockTxCt, INTTYPE)
	txConfirmStats.txCtAvg = container.NewVectorWithSize(uint(txConfirmStats.buckets.Size()))
	setValue(txConfirmStats.txCtAvg, FLOAT64TYPE)
	txConfirmStats.curBlockVal = container.NewVectorWithSize(uint(txConfirmStats.buckets.Size()))
	setValue(txConfirmStats.curBlockVal, FLOAT64TYPE)
	txConfirmStats.avg = container.NewVectorWithSize(uint(txConfirmStats.buckets.Size()))
	setValue(txConfirmStats.avg, FLOAT64TYPE)

	return &txConfirmStats
}

func setValue(v *container.Vector, eleType int) {
	switch eleType {
	case FLOAT64TYPE:
		for i := 0; i < v.Size(); i++ {
			v.SetValueByIndex(i, 0.0)
		}
	case INTTYPE:
		for i := 0; i < v.Size(); i++ {
			v.SetValueByIndex(i, 0)
		}
	case VECTORPTRTYPE:
		for i := 0; i < v.Size(); i++ {
			v.SetValueByIndex(i, container.NewVector())
		}
	}
}

//ClearCurrent Clear the state of the curBlock variables to start counting for the new block.
func (txConfirmStats *TxConfirmStats) ClearCurrent(blockHeight uint) {
	for j := 0; j < txConfirmStats.buckets.Size(); j++ {
		oldUnconfTxNum := txConfirmStats.oldUnconfTxs.Array[j]
		unconfTxVec := txConfirmStats.unconfTxs.Array[int(blockHeight)%txConfirmStats.unconfTxs.Size()].(*container.Vector)
		unconfNum := unconfTxVec.Array[j].(int)
		txConfirmStats.oldUnconfTxs.SetValueByIndex(j, oldUnconfTxNum.(int)+unconfNum)
		unconfTxVec.SetValueByIndex(j, 0)

		for i := 0; i < txConfirmStats.curBlockConf.Size(); i++ {
			curBlockTmp := txConfirmStats.curBlockConf.Array[i].(*container.Vector)
			curBlockTmp.SetValueByIndex(j, 0)
		}

		txConfirmStats.curBlockVal.SetValueByIndex(j, 0)
		txConfirmStats.curBlockTxCt.SetValueByIndex(j, 0)
	}
}

//Record a new transaction data point in the current block stats
// @param blocksToConfirm the number of blocks it took this transaction to confirm
// @param val the feerate of the transaction
// @warning blocksToConfirm is 1-based and has to be >= 1
func (txConfirmStats *TxConfirmStats) Record(blocksToConfirm int, val float64) {
	// blocksToConfirm is 1-based
	if blocksToConfirm < 1 {
		return
	}

	bucketindex := txConfirmStats.bucketMap.GetLowerBoundByfloat64(val).(uint)
	for i := blocksToConfirm; i <= txConfirmStats.curBlockConf.Size(); i++ {
		curBlockConfTmp := txConfirmStats.curBlockConf.Array[i-1].(*container.Vector)
		num := curBlockConfTmp.Array[bucketindex].(int)
		curBlockConfTmp.SetValueByIndex(int(bucketindex), num+1)
	}

	curTxCt := txConfirmStats.curBlockTxCt.Array[bucketindex].(int)
	txConfirmStats.curBlockTxCt.SetValueByIndex(int(bucketindex), curTxCt+1)
	curVal := txConfirmStats.curBlockVal.Array[bucketindex].(float64)
	txConfirmStats.curBlockTxCt.SetValueByIndex(int(bucketindex), curVal+val)
}

//UpdateMovingAverages Update our estimates by decaying our historical moving average and
//updating with the data gathered from the current block.
func (txConfirmStats *TxConfirmStats) UpdateMovingAverages() {
	for j := 0; j < txConfirmStats.buckets.Size(); j++ {
		for i := 0; i < txConfirmStats.confAvg.Size(); i++ {
			confAvgVecTmp := txConfirmStats.confAvg.Array[i].(*container.Vector)
			confAvgNum := confAvgVecTmp.Array[j].(float64)
			curConfVecTmp := txConfirmStats.curBlockConf.Array[i].(*container.Vector)
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

/*EstimateMedianVal Calculate a feerate estimate.  Find the lowest value bucket (or range of
 * buckets to make sure we have enough data points) whose transactions still
 * have sufficient likelihood of being confirmed within the target number of
 * confirmations
 * @param confTarget target number of confirmations
 * @param sufficientTxVal required average number of transactions per block
 * in a bucket range
 * @param minSuccess the success probability we require
 * @param requireGreater return the lowest feerate such that all higher
 * values pass minSuccess OR
 *        return the highest feerate such that all lower values fail
 * minSuccess
 * @param nBlockHeight the current block height
 * returns -1 on error conditions
 */
func (txConfirmStats *TxConfirmStats) EstimateMedianVal(confTarget int, sufficientTxVal,
	successBreakPoint float64, requireGreater bool, nBlockHeight uint) float64 {

	// Counters for a bucket (or range of buckets)，Number of tx's confirmed within the confTarget
	nConf := 0.0
	// Total number of tx's that were ever confirmed
	totalNum := 0.0
	// Number of tx's still in mempool for confTarget or longer
	extraNum := 0
	maxBucketInex := txConfirmStats.buckets.Size() - 1
	// requireGreater means we are looking for the lowest feerate such that all
	// higher values pass, so we start at maxbucketindex (highest feerate) and
	// look at successively smaller buckets until we reach failure. Otherwise,
	// we are looking for the highest feerate such that all lower values fail,
	// and we go in the opposite direction.
	startBucket := uint(0)
	step := 1
	if requireGreater {
		startBucket = uint(maxBucketInex)
		step = -1
	}

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

	for bucket := startBucket; bucket >= 0 && bucket <= uint(maxBucketInex); bucket += uint(step) {
		curFarBucket = bucket
		confAvgVecTmp := txConfirmStats.confAvg.Array[confTarget-1].(*container.Vector)
		nConf += confAvgVecTmp.Array[bucket].(float64)
		totalNum += txConfirmStats.txCtAvg.Array[bucket].(float64)

		for confct := uint(confTarget); confct < txConfirmStats.GetMaxConfirms(); confct++ {
			unconfTxsVecTmp := txConfirmStats.unconfTxs.Array[(nBlockHeight-confct)%bins].(*container.Vector)
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

	if bestNearBucket < bestFarBucket {
		minBucket = bestNearBucket
	}
	if bestNearBucket > bestFarBucket {
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

//NewTx Record a new transaction entering the mempool
func (txConfirmStats *TxConfirmStats) NewTx(nBlockHeight uint, val float64) uint {
	bucketIndex := txConfirmStats.bucketMap.GetLowerBoundByfloat64(val).(uint)
	blockIndex := nBlockHeight % uint(txConfirmStats.unconfTxs.Size())
	unconfxVecTmp := txConfirmStats.unconfTxs.Array[blockIndex].(*container.Vector)
	unconfxVecTmp.SetValueByIndex(int(bucketIndex), unconfxVecTmp.Array[bucketIndex].(int)+1)
	return bucketIndex

}

//RemoveTx Remove a transaction from mempool tracking stats
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
		unconfTxVecTmp := txConfirmStats.unconfTxs.Array[blockIndex].(*container.Vector)
		if unconfTxVecTmp.Array[bucketIndex].(int) > 0 {
			unconfTxVecTmp.Array[bucketIndex] = unconfTxVecTmp.Array[bucketIndex].(int) - 1
		}
	}
}

func writeForVector(writer io.Writer, vector *container.Vector) error {
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
		err = writeForVector(writer, v.(*container.Vector))
		if err != nil {
			return err
		}
	}
	return nil
}

func readForVector(reader io.Reader, vector *container.Vector) error {
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
	fileBuckets := container.NewVector()
	fileAvg := container.NewVector()
	fileTxCtAvg := container.NewVector()
	fileConfAvg := container.NewVector()

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
		tmpVector := container.NewVector()
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
		if fileConfAvg.Array[i].(*container.Vector).Size() != numBuckets {
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
	txConfirmStats.bucketMap = *container.NewBeeMap()

	// Resize the current block variables which aren't stored in the data file
	// to match the number of confirms and buckets
	txConfirmStats.curBlockConf = container.NewVector()
	txConfirmStats.unconfTxs = container.NewVector()
	for i := 0; i < maxConfirms; i++ {
		txConfirmStats.curBlockConf.PushBack(container.NewVector())
		txConfirmStats.unconfTxs.PushBack(container.NewVector())
	}
	txConfirmStats.curBlockTxCt = container.NewVector()
	txConfirmStats.curBlockVal = container.NewVector()
	txConfirmStats.oldUnconfTxs = container.NewVector()
	for i := 0; i < txConfirmStats.buckets.Size(); i++ {
		txConfirmStats.bucketMap.Set(txConfirmStats.buckets.Array[i], i)
	}

	return nil
}
