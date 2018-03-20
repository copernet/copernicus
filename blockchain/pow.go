package blockchain

import (
	"math/big"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/utils"
)

type Pow struct{}

func (pow *Pow) GetNextWorkRequired(indexPrev *core.BlockIndex, blHeader *core.BlockHeader, params *msg.BitcoinParams) uint32 {
	if indexPrev == nil {
		return BigToCompact(params.PowLimit)
	}

	// Special rule for regTest: we never retarget.
	if params.FPowNoRetargeting {
		return indexPrev.Bits
	}

	if indexPrev.GetMedianTimePast() >= params.CashHardForkActivationTime {
		return pow.getNextCashWorkRequired(indexPrev, blHeader, params)
	}

	return pow.getNextEDAWorkRequired(indexPrev, blHeader, params)
}

func (pow *Pow) calculateNextWorkRequired(indexPrev *core.BlockIndex, firstBlockTime int64, params *msg.BitcoinParams) uint32 {
	if params.FPowNoRetargeting {
		return indexPrev.Bits
	}

	//Limit adjustment step
	actualTimeSpan := indexPrev.GetBlockTime() - uint32(firstBlockTime)
	if actualTimeSpan < uint32(params.TargetTimespan/4) {
		actualTimeSpan = uint32(params.TargetTimespan / 4)
	}

	if actualTimeSpan > uint32(params.TargetTimespan*4) {
		actualTimeSpan = uint32(params.TargetTimespan * 4)
	}

	// Retarget
	bnNew := CompactToBig(indexPrev.Bits)
	bnNew.Mul(bnNew, big.NewInt(int64(actualTimeSpan)))
	bnNew.Div(bnNew, big.NewInt(int64(params.TargetTimespan)))
	if bnNew.Cmp(params.PowLimit) > 0 {
		bnNew = params.PowLimit
	}
	return BigToCompact(bnNew)
}

//GetNextCashWorkRequired Compute the next required proof of work using a weighted
// average of the estimated hashRate per block.
//
//Using a weighted average ensure that the timestamp parameter cancels out in
//most of the calculation - except for the timestamp of the first and last
//block. Because timestamps are the least trustworthy information we have as
//input, this ensures the algorithm is more resistant to malicious inputs.
func (pow *Pow) getNextCashWorkRequired(indexPrev *core.BlockIndex, blHeader *core.BlockHeader, params *msg.BitcoinParams) uint32 {
	if indexPrev == nil {
		panic("This cannot handle the genesis block and early blocks in general.")
	}

	// Special difficulty rule for testnet:
	// If the new block's timestamp is more than 2* 10 minutes then allow
	// mining of a min-difficulty block.
	if params.FPowAllowMinDifficultyBlocks && (blHeader.GetBlockTime() > indexPrev.GetBlockTime()+uint32(2*params.TargetTimePerBlock)) {
		return BigToCompact(params.PowLimit)
	}

	// Compute the difficulty based on the full adjustement interval.
	if int64(indexPrev.Height) < params.DifficultyAdjustmentInterval() {
		panic("this height should not less than params.DifficultyAdjustmentInterval()")
	}

	// Get the last suitable block of the difficulty interval.
	indexLast := pow.getSuitableBlock(indexPrev)
	if indexLast == nil {
		panic("the pindexLast value should not equal nil")
	}

	// Get the first suitable block of the difficulty interval.
	heightFirst := indexPrev.Height - 144
	indexFirst := pow.getSuitableBlock(indexPrev.GetAncestor(heightFirst))
	if indexFirst == nil {
		panic("the indexFirst should not equal nil")
	}

	// Compute the target based on time and work done during the interval.
	nextTarget := pow.computeTarget(indexFirst, indexLast, params)
	if nextTarget.Cmp(params.PowLimit) > 0 {
		return BigToCompact(params.PowLimit)
	}

	return BigToCompact(nextTarget)
}

// getNextEDAWorkRequired Compute the next required proof of work using the
// legacy Bitcoin difficulty adjustement + Emergency Difficulty Adjustement (EDA).
func (pow *Pow) getNextEDAWorkRequired(indexPrev *core.BlockIndex, pblock *core.BlockHeader, params *msg.BitcoinParams) uint32 {

	// Only change once per difficulty adjustment interval
	nHeight := indexPrev.Height + 1
	if int64(nHeight)%params.DifficultyAdjustmentInterval() == 0 {
		// Go back by what we want to be 14 days worth of blocks
		if int64(nHeight) < params.DifficultyAdjustmentInterval() {
			panic("the current block height should not less than difficulty adjustment interval dural")
		}

		nHeightFirst := nHeight - int(params.DifficultyAdjustmentInterval())
		pindexFirst := indexPrev.GetAncestor(nHeightFirst)
		if pindexFirst == nil {
			panic("the blockIndex should not equal nil")
		}

		return pow.calculateNextWorkRequired(indexPrev, int64(pindexFirst.GetBlockTime()), params)
	}

	nProofOfWorkLimit := BigToCompact(params.PowLimit)
	if params.FPowAllowMinDifficultyBlocks {
		// Special difficulty rule for testnet:
		// If the new block's timestamp is more than 2* 10 minutes then allow
		// mining of a min-difficulty block.
		if pblock.GetBlockTime() > indexPrev.GetBlockTime()+2*uint32(params.TargetTimePerBlock) {
			return nProofOfWorkLimit
		}
		// Return the last non-special-min-difficulty-rules-block
		index := indexPrev
		for index.Prev != nil && int64(index.Height)%params.DifficultyAdjustmentInterval() != 0 &&
			index.Bits == nProofOfWorkLimit {
			index = index.Prev
		}

		return index.Bits
	}

	// We can't go bellow the minimum, so early bail.
	bits := indexPrev.Bits
	if bits == nProofOfWorkLimit {
		return nProofOfWorkLimit
	}

	// If producing the last 6 block took less than 12h, we keep the same
	// difficulty
	index6 := indexPrev.GetAncestor(nHeight - 7)
	if index6 == nil {
		panic("the block Index should not equal nil")
	}
	mtp6Blocks := indexPrev.GetMedianTimePast() - index6.GetMedianTimePast()
	if mtp6Blocks < 12*3600 {
		return bits
	}

	// If producing the last 6 block took more than 12h, increase the difficulty
	// target by 1/4 (which reduces the difficulty by 20%). This ensure the
	// chain do not get stuck in case we lose hashRate abruptly.
	nPow := CompactToBig(bits)
	nPow.Add(nPow, big.NewInt(0).Div(nPow, big.NewInt(4)))

	// Make sure we do not go bellow allowed values.
	if nPow.Cmp(params.PowLimit) > 0 {
		nPow = params.PowLimit
	}

	return BigToCompact(nPow)
}

// computeTarget Compute the a target based on the work done between 2 blocks and the time
// required to produce that work.
func (pow *Pow) computeTarget(indexFirst, indexLast *core.BlockIndex, params *msg.BitcoinParams) *big.Int {
	if indexLast.Height <= indexFirst.Height {
		panic("indexLast height should greater the indexFirst height ")
	}

	/**
	* From the total work done and the time it took to produce that much work,
	* we can deduce how much work we expect to be produced in the targeted time
	* between blocks.
	 */
	work := new(big.Int).Sub(&indexLast.ChainWork, &indexFirst.ChainWork)
	work.Mul(work, big.NewInt(int64(params.TargetTimePerBlock)))

	// In order to avoid difficulty cliffs, we bound the amplitude of the
	// adjustement we are going to do.
	if indexLast.Time <= indexFirst.Time {
		panic("indexLast time should greater than indexFirst time ")
	}
	actualTimeSpan := indexLast.Time - indexFirst.Time
	if actualTimeSpan > uint32(288*params.TargetTimePerBlock) {
		actualTimeSpan = uint32(288 * params.TargetTimePerBlock)
	} else if actualTimeSpan < uint32(72*params.TargetTimePerBlock) {
		actualTimeSpan = 72 * uint32(params.TargetTimePerBlock)
	}

	work.Div(work, big.NewInt(int64(actualTimeSpan)))
	/**
	 * We need to compute T = (2^256 / W) - 1 but 2^256 doesn't fit in 256 bits.
	 * By expressing 1 as W / W, we get (2^256 - W) / W, and we can compute
	 * 2^256 - W as the complement of W.
	 */
	return new(big.Int).Sub(new(big.Int).Div(oneLsh256, work), big.NewInt(1))
}

func (pow *Pow) getSuitableBlock(index *core.BlockIndex) *core.BlockIndex {
	if index.Height < 3 {
		panic("This block height should not less than 3")
	}

	//In order to avoid a block is a very skewed timestamp to have too much
	//influence, we select the median of the 3 top most blocks as a starting
	//point.
	blocks := make([]*core.BlockIndex, 3)
	blocks[2] = index
	blocks[1] = index.Prev
	blocks[0] = blocks[1].Prev

	// Sorting network.
	if blocks[0].Time > blocks[2].Time {
		blocks[0], blocks[2] = blocks[2], blocks[0]
	}

	if blocks[0].Time > blocks[1].Time {
		blocks[0], blocks[1] = blocks[1], blocks[0]
	}

	if blocks[1].Time > blocks[2].Time {
		blocks[1], blocks[2] = blocks[2], blocks[1]
	}

	// We should have our candidate in the middle now.
	return blocks[1]
}

func (pow *Pow) CheckProofOfWork(hash *utils.Hash, bits uint32, params *msg.BitcoinParams) bool {
	target := CompactToBig(bits)
	if target.Sign() <= 0 || target.Cmp(params.PowLimit) > 0 ||
		HashToBig(hash).Cmp(target) > 0 {
		return false
	}

	return true
}
