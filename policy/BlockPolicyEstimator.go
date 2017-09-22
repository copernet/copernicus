package policy

import (
	"io"

	beegoUtils "github.com/astaxie/beego/utils"
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/utils"
)

type BlockPolicyEstimator struct {
	minTrackedFee  utils.FeeRate
	bestSeenHeight uint
	txStatsInfo    TxStatsInfo
	/** Classes to track historical data on transaction confirmations*/
	mapMemPoolTxs *beegoUtils.BeeMap
	feeStats      TxConfirmStats
	trackedTxs    uint
	untranckedTxs uint
}

func NewBlockPolicyEstmator(rate utils.FeeRate) *BlockPolicyEstimator {
	blockPolicyEstimator := BlockPolicyEstimator{}

	if utils.MIN_FEERATE < 0 {
		panic("Min feerate must be nonzero")
	}
	blockPolicyEstimator.bestSeenHeight = 0
	blockPolicyEstimator.trackedTxs = 0
	blockPolicyEstimator.untranckedTxs = 0
	if rate.SataoshisPerK < utils.MIN_FEERATE {
		blockPolicyEstimator.minTrackedFee.SataoshisPerK = utils.MIN_FEERATE
	}
	blockPolicyEstimator.minTrackedFee.SataoshisPerK = rate.SataoshisPerK
	vfeeList := algorithm.NewVector()
	for bucketBoundary := float64(blockPolicyEstimator.minTrackedFee.GetFeePerK()); bucketBoundary <= float64(utils.MAX_FEERATE); bucketBoundary *= utils.FEE_SPACING {
		vfeeList.PushBack(bucketBoundary)
	}
	vfeeList.PushBack(float64(utils.INF_FEERATE))
	blockPolicyEstimator.feeStats = *NewTxConfirmStats(*vfeeList, utils.MAX_BLOCK_CONFIRMS, utils.DEFAULT_DECAY)

	return &blockPolicyEstimator
}

func (blockPolicyEstimator *BlockPolicyEstimator) ProcessTransaction(entry *mempool.TxMempoolEntry, validFeeEstimate bool) {

}

func (blockPolicyEstimator *BlockPolicyEstimator) ProcessBlockTx(blockHeight uint, entry *mempool.TxMempoolEntry) bool {
	return true
}

func (blockPolicyEstimator *BlockPolicyEstimator) ProcessBlock(blockHeight uint, entry *algorithm.Vector) {

}

func (blockPolicyEstimator *BlockPolicyEstimator) EstimateFee(confTarget int) utils.FeeRate {
	feeRate := utils.FeeRate{}
	return feeRate
}

func (blockPolicyEstimator *BlockPolicyEstimator) EstimateSmartFee(confTarget uint, answerFoundAtTarget *int, pool *mempool.Mempool) utils.FeeRate {
	feeRate := utils.FeeRate{}
	return feeRate
}

func (blockPolicyEstimator *BlockPolicyEstimator) EstimatePriority(confTarget int) float64 {
	return -1
}

func (blockPolicyEstimator *BlockPolicyEstimator) Serialize(writer io.Writer) error {
	return nil
}

func (blockPolicyEstimator *BlockPolicyEstimator) Deserialize(reader io.Reader) error {
	return nil
}
