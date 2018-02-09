package blockchain

import (
	"fmt"
	"sync"

	"github.com/btcboost/copernicus/msg"
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
	Condition(index *BlockIndex, params *msg.BitcoinParams) bool
	BeginTime(params *msg.BitcoinParams) int64
	EndTime(params *msg.BitcoinParams) int64
	Period(params *msg.BitcoinParams) int
	Threshold(params *msg.BitcoinParams) int
}

type VersionBitsCache struct {
	sync.RWMutex
	cache [msg.MAX_VERSION_BITS_DEPLOYMENTS]ThresholdConditionCache
}

func newVersionBitsCache() *VersionBitsCache {
	var cache [msg.MAX_VERSION_BITS_DEPLOYMENTS]ThresholdConditionCache
	for i := 0; i < int(msg.MAX_VERSION_BITS_DEPLOYMENTS); i++ {
		cache[i] = make(ThresholdConditionCache)
	}
	return &VersionBitsCache{cache: cache}
}

func (vbc *VersionBitsCache) Clear() {
	for i := 0; i < int(msg.MAX_VERSION_BITS_DEPLOYMENTS); i++ {
		vbc.cache[i] = make(ThresholdConditionCache)
	}
}

func VersionBitsState(indexPrev *BlockIndex, params *msg.BitcoinParams, pos msg.DeploymentPos, vbc *VersionBitsCache) ThresholdState {
	vc := &VersionBitsConditionChecker{id: pos}
	return GetStateFor(vc, indexPrev, params, vbc.cache[pos])
}

func VersionBitsStateSinceHeight(indexPrev *BlockIndex, params *msg.BitcoinParams, pos msg.DeploymentPos, vbc *VersionBitsCache) int {
	vc := &VersionBitsConditionChecker{id: pos}
	return GetStateSinceHeightFor(vc, indexPrev, params, vbc.cache[pos])
}

func VersionBitsMask(params *msg.BitcoinParams, pos msg.DeploymentPos) uint32 {
	vc := VersionBitsConditionChecker{id: pos}
	return uint32(vc.Mask(params))
}

type VersionBitsConditionChecker struct {
	id msg.DeploymentPos
}

func (vc *VersionBitsConditionChecker) BeginTime(params *msg.BitcoinParams) int64 {
	return params.Deployments[vc.id].StartTime
}

func (vc *VersionBitsConditionChecker) EndTime(params *msg.BitcoinParams) int64 {
	return params.Deployments[vc.id].Timeout
}

func (vc *VersionBitsConditionChecker) Period(params *msg.BitcoinParams) int {
	return int(params.MinerConfirmationWindow)
}

func (vc *VersionBitsConditionChecker) Threshold(params *msg.BitcoinParams) int {
	return int(params.RuleChangeActivationThreshold)
}

func (vc *VersionBitsConditionChecker) Condition(index *BlockIndex, params *msg.BitcoinParams) bool {
	return ((int(index.Version) & VERSIONBITS_TOP_MASK) == VERSIONBITS_TOP_BITS) && (index.Version&vc.Mask(params)) != 0
}

func (vc *VersionBitsConditionChecker) Mask(params *msg.BitcoinParams) int32 {
	return int32(1) << uint(params.Deployments[vc.id].Bit)
}

func GetStateFor(vc AbstractThresholdConditionChecker, indexPrev *BlockIndex, params *msg.BitcoinParams, cache ThresholdConditionCache) ThresholdState {
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
	for {
		if _, ok := cache[indexPrev]; !ok {
			if indexPrev == nil {
				// The genesis block is by definition defined.
				cache[indexPrev] = THRESHOLD_DEFINED
				break
			}
			if indexPrev.GetMedianTimePast() < nTimeStart {
				// Optimization: don't recompute down further, as we know every
				// earlier block will be before the start time
				cache[indexPrev] = THRESHOLD_DEFINED
				break
			}
			toCompute = append(toCompute, indexPrev)
			indexPrev = indexPrev.GetAncestor(indexPrev.Height - nPeriod)
		} else {
			break
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
					fmt.Println("********* height : ", indexPrev.Height)
					//panic("jjjjjjj")
					stateNext = THRESHOLD_FAILED
					break
				}
				if indexPrev.Height == 2999 {
					fmt.Println("GetStateFor time : ", indexPrev.GetMedianTimePast())
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

func GetStateSinceHeightFor(vc AbstractThresholdConditionChecker, indexPrev *BlockIndex, params *msg.BitcoinParams, cache ThresholdConditionCache) int {
	initialState := GetStateFor(vc, indexPrev, params, cache)
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
	if indexPrev.Height == 2999 {
		fmt.Println("initialState : ", initialState)
	}
	for previousPeriodParent != nil && GetStateFor(vc, previousPeriodParent, params, cache) == initialState {
		indexPrev = previousPeriodParent
		previousPeriodParent = indexPrev.GetAncestor(indexPrev.Height - nPeriod)
	}

	// Adjust the result because right now we point to the parent block.
	return indexPrev.Height + 1
}
