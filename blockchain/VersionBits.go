package blockchain

import (
	"github.com/btcboost/copernicus/consensus"
)

const (
	// VERSIONBITS_LAST_OLD_BLOCK_VERSION What block version to use for new blocks (pre versionbits)
	VERSIONBITS_LAST_OLD_BLOCK_VERSION = 4
	// VERSIONBITS_TOP_BITS What bits to set in version for versionbits blocks
	VERSIONBITS_TOP_BITS = 0x20000000
	// VERSIONBITS_TOP_MASK What bitmask determines whether versionbits is in use
	VERSIONBITS_TOP_MASK = 0xE0000000
	// VERSIONBITS_NUM_BITS Total bits available for versionbits
	VERSIONBITS_NUM_BITS = 29
)

type ThresholdState int

const (
	THRESHOLD_DEFINED ThresholdState = iota
	THRESHOLD_STARTED
	THRESHOLD_LOCKED_IN
	THRESHOLD_ACTIVE
	THRESHOLD_FAILED
)

type BIP9DeploymentInfo struct {
	name     string
	gbtForce bool
}

type ThresholdConditionCache map[*BlockIndex]ThresholdState

var VersionBitsDeploymentInfo = []BIP9DeploymentInfo{
	{
		name:     "testdummy",
		gbtForce: true,
	},
	{
		name:     "csv",
		gbtForce: true,
	},
}

type AbstractThresholdConditionChecker interface {
	Condition(index *BlockIndex, params *consensus.Params) bool
	BeginTime(params *consensus.Params) int64
	EndTime(params *consensus.Params) int64
	Period(params *consensus.Params) int
	Threshold(params *consensus.Params) int
	GetStateFor(indexPrev *BlockIndex, params *consensus.Params, cache ThresholdConditionCache) ThresholdState
	GetStateSinceHeightFor(indexPrev *BlockIndex, params *consensus.Params, cache ThresholdConditionCache) ThresholdState
}

type VersionBitsCache [consensus.MAX_VERSION_BITS_DEPLOYMENTS]ThresholdConditionCache

func VersionBitsState(indexPrev *BlockIndex, params *consensus.Params, pos consensus.DeploymentPos, cache *VersionBitsCache) ThresholdState {
	vc := VersionBitsConditionChecker{id: pos}
	return vc.GetStateFor(indexPrev, params, cache[pos])
}

func VersionBitsStateSinceHeight(indexPrev *BlockIndex, params *consensus.Params, pos consensus.DeploymentPos, cache *VersionBitsCache) int {
	vc := VersionBitsConditionChecker{id: pos}
	return vc.GetStateSinceHeightFor(indexPrev, params, cache[pos])
}

func VersionBitsMask(params *consensus.Params, pos consensus.DeploymentPos) uint32 {
	vc := VersionBitsConditionChecker{id: pos}
	return uint32(vc.Mask(params))
}

type VersionBitsConditionChecker struct {
	id consensus.DeploymentPos
}

func (vc *VersionBitsConditionChecker) BeginTime(params *consensus.Params) int64 {
	return params.Deployments[vc.id].StartTime
}

func (vc *VersionBitsConditionChecker) EndTime(params *consensus.Params) int64 {
	return params.Deployments[vc.id].Timeout
}

func (vc *VersionBitsConditionChecker) Period(params *consensus.Params) int {
	return int(params.MinerConfirmationWindow)
}

func (vc *VersionBitsConditionChecker) Threshold(params *consensus.Params) int {
	return int(params.RuleChangeActivationThreshold)
}

func (vc *VersionBitsConditionChecker) Condition(index *BlockIndex, params *consensus.Params) bool {
	return ((int(index.Version) & VERSIONBITS_TOP_MASK) == VERSIONBITS_TOP_BITS) && (index.Version&vc.Mask(params)) != 0
}

func (vc *VersionBitsConditionChecker) Mask(params *consensus.Params) int32 {
	return int32(1) << uint(params.Deployments[vc.id].Bit)
}

func (vc *VersionBitsConditionChecker) GetStateFor(indexPrev *BlockIndex, params *consensus.Params, cache ThresholdConditionCache) ThresholdState {
	nPeriod := vc.Period(params)
	nThreshold := vc.Threshold(params)
	nTimeStart := vc.BeginTime(params)
	nTimeTimeout := vc.EndTime(params)

	// A block's state is always the same as that of the first of its period, so
	// it is computed based on a pindexPrev whose height equals a multiple of
	// nPeriod - 1.
	if indexPrev != nil {
		indexPrev = indexPrev.GetAncestor(indexPrev.Height - (indexPrev.Height+1)%nPeriod)
	}

	// Walk backwards in steps of nPeriod to find a pindexPrev whose information
	// is known
	toCompute := make([]*BlockIndex, 0)
	_, ok := cache[indexPrev]
	if !ok {
		switch {
		case indexPrev == nil:
			cache[indexPrev] = THRESHOLD_DEFINED
		case indexPrev.GetMedianTimePast() < nTimeStart:
			// Optimization: don't recompute down further, as we know every
			// earlier block will be before the start time
			cache[indexPrev] = THRESHOLD_DEFINED
		default:
			toCompute = append(toCompute, indexPrev)
			indexPrev = indexPrev.GetAncestor(indexPrev.Height - nPeriod)
		}
	}

	// At this point, cache[pindexPrev] is known
	state, ok := cache[indexPrev]
	if !ok {
		panic("there should be a element in cache")
	}

	// Now walk forward and compute the state of descendants of pindexPrev
	for n := 0; n < len(toCompute); n++ {
		stateNext := state
		indexPrev = toCompute[len(toCompute)-1]
		toCompute = toCompute[:(len(toCompute) - 1)]

		switch state {
		case THRESHOLD_DEFINED:
			{
				if indexPrev.GetMedianTimePast() >= nTimeTimeout {
					stateNext = THRESHOLD_FAILED
				} else if indexPrev.GetMedianTimePast() >= nTimeStart {
					stateNext = THRESHOLD_STARTED
				}
			}
		case THRESHOLD_STARTED:
			{
				if indexPrev.GetMedianTimePast() >= nTimeTimeout {
					stateNext = THRESHOLD_FAILED
				}

				// We need to count
				indexCount := indexPrev
				count := 0
				for i := 0; i < nPeriod; i++ {
					if vc.Condition(indexCount, params) {
						count++
					}
					indexCount = indexCount.PPrev
				}
				if count >= nThreshold {
					stateNext = THRESHOLD_LOCKED_IN
				}
			}
		case THRESHOLD_LOCKED_IN:
			{
				// Always progresses into ACTIVE.
				stateNext = THRESHOLD_ACTIVE
			}
		case THRESHOLD_FAILED:
		case THRESHOLD_ACTIVE:
			{
				// Nothing happens, these are terminal states.
			}
		}
		state = stateNext
		cache[indexPrev] = state
	}
	return state
}

func (vc *VersionBitsConditionChecker) GetStateSinceHeightFor(indexPrev *BlockIndex, params *consensus.Params, cache ThresholdConditionCache) int {
	initialState := vc.GetStateFor(indexPrev, params, cache)
	// BIP 9 about state DEFINED: "The genesis block is by definition in this
	// state for each deployment."
	if initialState == THRESHOLD_DEFINED {
		return 0
	}

	nPeriod := vc.Period(params)
	// A block's state is always the same as that of the first of its period, so
	// it is computed based on a pindexPrev whose height equals a multiple of
	// nPeriod - 1. To ease understanding of the following height calculation,
	// it helps to remember that right now pindexPrev points to the block prior
	// to the block that we are computing for, thus: if we are computing for the
	// last block of a period, then pindexPrev points to the second to last
	// block of the period, and if we are computing for the first block of a
	// period, then pindexPrev points to the last block of the previous period.
	// The parent of the genesis block is represented by nullptr.
	indexPrev = indexPrev.GetAncestor(indexPrev.Height - ((indexPrev.Height + 1) % nPeriod))
	previousPeriodParent := indexPrev.GetAncestor(indexPrev.Height - nPeriod)

	for previousPeriodParent != nil && vc.GetStateFor(previousPeriodParent, params, cache) == initialState {
		indexPrev = previousPeriodParent
		previousPeriodParent = indexPrev.GetAncestor(indexPrev.Height - nPeriod)
	}

	// Adjust the result because right now we point to the parent block.
	return indexPrev.Height + 1
}
