package mempool

import (
	"encoding/binary"
	"io"

	"github.com/astaxie/beego/logs"
	beegoUtils "github.com/astaxie/beego/utils"
	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/policy"
	"github.com/btcboost/copernicus/utils"
)

/*
 * BlockPolicyEstimator We want to be able to estimate feerates that are needed on tx's to be
 * included in a certain number of blocks.  Every time a block is added to the
 * best chain, this class records stats on the transactions included in that
 * block.
 */

type BlockPolicyEstimator struct {
	minTrackedFee  utils.FeeRate
	bestSeenHeight uint
	txStatsInfo    policy.TxStatsInfo

	/** Classes to track historical data on transaction confirmations*/
	mapMemPoolTxs *beegoUtils.BeeMap // map[utils.Hash]TxStatsInfo
	feeStats      policy.TxConfirmStats
	trackedTxs    uint
	untranckedTxs uint
}

func (blockPolicyEstimator *BlockPolicyEstimator) ProcessTransaction(entry *TxMempoolEntry, validFeeEstimate bool) {
	txHeight := entry.EntryHeight
	txID := entry.TxRef.Hash
	if has := blockPolicyEstimator.mapMemPoolTxs.Get(txID); has != nil {
		logs.Debug("estimatefee Blockpolicy error mempool tx %s already being tracked\n", txID.ToString())
		return
	}
	if txHeight != blockPolicyEstimator.bestSeenHeight {
		// Ignore side chains and re-orgs; assuming they are random they don't
		// affect the estimate. We'll potentially double count transactions in
		// 1-block reorgs. Ignore txs if BlockPolicyEstimator is not in sync
		// with chainActive.Tip(). It will be synced next time a block is
		// processed.
		return
	}
	// Only want to be updating estimates when our blockchain is synced,
	// otherwise we'll miscalculate how many blocks its taking to get included.
	if !validFeeEstimate {
		blockPolicyEstimator.untranckedTxs++
		return
	}
	blockPolicyEstimator.trackedTxs++
	// Feerates are stored and reported as BCC-per-kb:
	feeRate := utils.NewFeeRateWithSize(int64(entry.Fee), entry.TxSize)
	bucketIndex := blockPolicyEstimator.feeStats.NewTx(txHeight, float64(feeRate.GetFeePerK()))
	txStatsInfo := policy.TxStatsInfo{BlockHeight: txHeight, BucketIndex: bucketIndex}
	blockPolicyEstimator.mapMemPoolTxs.Set(txID, txStatsInfo)
}

func (blockPolicyEstimator *BlockPolicyEstimator) ProcessBlockTx(blockHeight uint, entry *TxMempoolEntry) bool {

	if !blockPolicyEstimator.RemoveTx(entry.TxRef.Hash) {
		// This transaction wasn't being tracked for fee estimation；
		return false
	}

	// How many blocks did it take for miners to include this transaction?
	// blocksToConfirm is 1-based, so a transaction included in the earliest
	// possible block has confirmation count of 1
	blocksToConfirm := blockHeight - entry.EntryHeight
	if blocksToConfirm <= 0 {
		logs.Error("estimatefee Blockpolicy error Transaction had negative blocksToConfirm\n")
		return false
	}

	// Feerates are stored and reported as BCC-per-kb:
	feeRate := utils.NewFeeRateWithSize(int64(entry.Fee), entry.TxSize)
	blockPolicyEstimator.feeStats.Record(int(blocksToConfirm), float64(feeRate.GetFeePerK()))

	return true
}

func (blockPolicyEstimator *BlockPolicyEstimator) ProcessBlock(blockHeight uint, entry []*TxMempoolEntry) {

	if blockHeight <= blockPolicyEstimator.bestSeenHeight {
		// Ignore side chains and re-orgs; assuming they are random they don't
		// affect the estimate. And if an attacker can re-org the chain at will,
		// then you've got much bigger problems than "attacker can influence
		// transaction fees."
		return
	}

	// Must update nBestSeenHeight in sync with ClearCurrent so that calls to
	// removeTx (via processBlockTx) correctly calculate age of unconfirmed txs
	// to remove from tracking.
	blockPolicyEstimator.bestSeenHeight = blockHeight
	// Clear the current block state and update unconfirmed circular buffer
	blockPolicyEstimator.feeStats.ClearCurrent(blockHeight)

	countedTxs := uint(0)
	// Repopulate the current block states
	for i := 0; i < len(entry); i++ {
		if blockPolicyEstimator.ProcessBlockTx(blockHeight, entry[i]) {
			countedTxs++
		}
	}

	blockPolicyEstimator.feeStats.UpdateMovingAverages()

	logs.Trace("estimatefee Blockpolicy after updating estimates for %u of %u txs in block, since last"+
		" block %u of %u tracked, new mempool map size %u", countedTxs, len(entry), blockPolicyEstimator.trackedTxs,
		blockPolicyEstimator.trackedTxs+blockPolicyEstimator.untranckedTxs, blockPolicyEstimator.mapMemPoolTxs.Count())

	blockPolicyEstimator.trackedTxs = 0
	blockPolicyEstimator.untranckedTxs = 0
}

func (blockPolicyEstimator *BlockPolicyEstimator) EstimateFee(confTarget int) utils.FeeRate {
	feeRate := utils.FeeRate{SataoshisPerK: 0}
	// Return failure if trying to analyze a target we're not tracking
	// It's not possible to get reasonable estimates for confTarget of 1
	if confTarget <= 1 || uint(confTarget) > blockPolicyEstimator.feeStats.GetMaxConfirms() {
		return feeRate
	}

	median := blockPolicyEstimator.feeStats.EstimateMedianVal(confTarget, utils.SufficientFeeTxs,
		utils.MinSuccessPct, true, blockPolicyEstimator.bestSeenHeight)
	if median < 0 {
		return feeRate
	}

	return utils.FeeRate{SataoshisPerK: int64(median)}
}

func (blockPolicyEstimator *BlockPolicyEstimator) EstimateSmartFee(confTarget int, answerFoundAtTarget *int, pool *Mempool) utils.FeeRate {

	if answerFoundAtTarget != nil {
		*answerFoundAtTarget = confTarget
	}

	// Return failure if trying to analyze a target we're not tracking
	if confTarget <= 0 || uint(confTarget) > blockPolicyEstimator.feeStats.GetMaxConfirms() {
		return utils.FeeRate{SataoshisPerK: 0}
	}

	// It's not possible to get reasonable estimates for confTarget of 1
	if confTarget == 1 {
		confTarget = 2
	}

	median := float64(-1)
	for median < 0 && uint(confTarget) <= blockPolicyEstimator.feeStats.GetMaxConfirms() {
		median = blockPolicyEstimator.feeStats.EstimateMedianVal(confTarget, utils.SufficientFeeTxs, utils.MinSuccessPct,
			true, blockPolicyEstimator.bestSeenHeight)
		confTarget++
	}

	if answerFoundAtTarget != nil {
		*answerFoundAtTarget = confTarget - 1
	}

	// If mempool is limiting txs , return at least the min feerate from the
	// mempool
	maxMempool := utils.GetArg("-maxmempool", int64(policy.DefaultMaxMemPoolSize)) * 1000000
	minPoolFeeTmp := pool.GetMinFee(maxMempool)
	minPoolFee := minPoolFeeTmp.GetFeePerK()

	if minPoolFee > 0 && float64(minPoolFee) > median {
		return utils.FeeRate{SataoshisPerK: minPoolFee}
	}
	if median < 0 {
		median = 0
	}
	return utils.FeeRate{SataoshisPerK: int64(median)}
}

func (blockPolicyEstimator *BlockPolicyEstimator) EstimatePriority(confTarget int) float64 {

	return -1
}

func (blockPolicyEstimator *BlockPolicyEstimator) EstimateSmartPriority(confTarget int, answerFoundAtTarget *int, pool *Mempool) float64 {
	if answerFoundAtTarget != nil {
		*answerFoundAtTarget = confTarget
	}

	// If memPool is limiting txs, no priority txs are allowed
	maxMempool := utils.GetArg("-maxmempool", int64(policy.DefaultMaxMemPoolSize)) * 1000000
	minPoolFeeTmp := pool.GetMinFee(maxMempool)
	minPoolFee := minPoolFeeTmp.GetFeePerK()

	if minPoolFee > 0 {
		return utils.InfPriority
	}

	return -1
}

func (blockPolicyEstimator *BlockPolicyEstimator) Serialize(writer io.Writer) error {
	err := binary.Write(writer, binary.LittleEndian, blockPolicyEstimator.bestSeenHeight)
	if err != nil {
		return err
	}
	err = blockPolicyEstimator.feeStats.Serialize(writer)
	return err
}

func (blockPolicyEstimator *BlockPolicyEstimator) Deserialize(reader io.Reader, fileVersion int) error {
	fileBestSeenHeight := uint(0)
	err := binary.Read(reader, binary.LittleEndian, &fileBestSeenHeight)
	if err != nil {
		return err
	}
	err = blockPolicyEstimator.feeStats.Deserialize(reader)
	if err != nil {
		return err
	}
	blockPolicyEstimator.bestSeenHeight = fileBestSeenHeight

	if fileVersion < 139900 {
		priStats := policy.TxConfirmStats{}
		err = priStats.Deserialize(reader)
	}
	return err
}

func (blockPolicyEstimator *BlockPolicyEstimator) RemoveTx(hash utils.Hash) bool {

	value := blockPolicyEstimator.mapMemPoolTxs.Get(hash)
	if value == nil {
		return false
	}
	txStatsInfo := value.(policy.TxStatsInfo)
	blockPolicyEstimator.feeStats.RemoveTx(txStatsInfo.BlockHeight, blockPolicyEstimator.bestSeenHeight, txStatsInfo.BucketIndex)
	blockPolicyEstimator.mapMemPoolTxs.Delete(hash)
	return true

}

func NewBlockPolicyEstmator(rate utils.FeeRate) *BlockPolicyEstimator {
	blockPolicyEstimator := BlockPolicyEstimator{}
	if utils.MinFeeRate < 0 {
		panic("Min feerate must be nonzero")
	}
	blockPolicyEstimator.mapMemPoolTxs = beegoUtils.NewBeeMap()
	blockPolicyEstimator.bestSeenHeight = 0
	blockPolicyEstimator.trackedTxs = 0
	blockPolicyEstimator.untranckedTxs = 0
	blockPolicyEstimator.minTrackedFee.SataoshisPerK = rate.SataoshisPerK
	if rate.SataoshisPerK < utils.MinFeeRate {
		blockPolicyEstimator.minTrackedFee.SataoshisPerK = utils.MinFeeRate
	}
	vfeeList := container.NewVector()
	for bucketBoundary := float64(blockPolicyEstimator.minTrackedFee.GetFeePerK()); bucketBoundary <= float64(utils.MaxFeeRate); bucketBoundary *= utils.FeeSpacing {
		vfeeList.PushBack(bucketBoundary)
	}
	vfeeList.PushBack(float64(utils.InfFeeRate))
	blockPolicyEstimator.feeStats = *policy.NewTxConfirmStats(vfeeList, utils.MaxBlockConfirms, utils.DefaultDecay)
	return &blockPolicyEstimator
}
