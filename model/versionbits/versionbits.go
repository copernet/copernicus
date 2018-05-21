package blockchain

import (
	"math"
	"sync"

	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/net/msg"
)

const (
	// VersionBitsLastOldBlockVersion what block version to use for new blocks (pre versionBits)
	VersionBitsLastOldBlockVersion = 4
	// VersionBitsTopBits what bits to set in version for versionBits blocks
	VersionBitsTopBits = 0x20000000
	// VersionBitsTopMask What bitMask determines whether versionBits is in use
	VersionBitsTopMask int64 = 0xE0000000
	// VersionBitsNumBits Total bits available for versionBits
	VersionBitsNumBits = 29
)

type ThresholdState int

const (
	ThresholdDefined ThresholdState = iota
	ThresholdStarted
	ThresholdLockedIn
	ThresholdActive
	ThresholdFailed
)

type BIP9DeploymentInfo struct {
	Name     string
	GbtForce bool
}

type ThresholdConditionCache map[*core.BlockIndex]ThresholdState

var VersionBitsDeploymentInfo = []BIP9DeploymentInfo{
	{
		Name:     "testdummy",
		GbtForce: true,
	},
	{
		Name:     "csv",
		GbtForce: true,
	},
}

type AbstractThresholdConditionChecker interface {
	Condition(index *core.BlockIndex, params *msg.BitcoinParams) bool
	BeginTime(params *msg.BitcoinParams) int64
	EndTime(params *msg.BitcoinParams) int64
	Period(params *msg.BitcoinParams) int
	Threshold(params *msg.BitcoinParams) int
}

var VBCache *VersionBitsCache // todo waring: there is a global variable(used as cache)

type VersionBitsCache struct {
	sync.RWMutex
	cache [consensus.MaxVersionBitsDeployments]ThresholdConditionCache
}

func NewVersionBitsCache() *VersionBitsCache {
	var cache [consensus.MaxVersionBitsDeployments]ThresholdConditionCache
	for i := 0; i < int(consensus.MaxVersionBitsDeployments); i++ {
		cache[i] = make(ThresholdConditionCache)
	}
	return &VersionBitsCache{cache: cache}
}

func (vbc *VersionBitsCache) Clear() {
	for i := 0; i < int(consensus.MaxVersionBitsDeployments); i++ {
		vbc.cache[i] = make(ThresholdConditionCache)
	}
}

func NewWarnBitsCache(bitNum int) []ThresholdConditionCache {
	w := make([]ThresholdConditionCache, 0)
	for i := 0; i < bitNum; i++ {
		thres := make(ThresholdConditionCache)
		w = append(w, thres)
	}

	return w
}

func VersionBitsState(indexPrev *core.BlockIndex, params *msg.BitcoinParams, pos consensus.DeploymentPos, vbc *VersionBitsCache) ThresholdState {
	vc := &VersionBitsConditionChecker{id: pos}
	return GetStateFor(vc, indexPrev, params, vbc.cache[pos])
}

func VersionBitsStateSinceHeight(indexPrev *core.BlockIndex, params *msg.BitcoinParams, pos consensus.DeploymentPos, vbc *VersionBitsCache) int {
	vc := &VersionBitsConditionChecker{id: pos}
	return GetStateSinceHeightFor(vc, indexPrev, params, vbc.cache[pos])
}

func VersionBitsMask(params *msg.BitcoinParams, pos consensus.DeploymentPos) uint32 {
	vc := VersionBitsConditionChecker{id: pos}
	return uint32(vc.Mask(params))
}

type VersionBitsConditionChecker struct {
	id consensus.DeploymentPos
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

func (vc *VersionBitsConditionChecker) Condition(index *core.BlockIndex, params *msg.BitcoinParams) bool {
	return ((int64(index.Header.Version) & VersionBitsTopMask) == VersionBitsTopBits) &&
		(index.Header.Version&vc.Mask(params)) != 0
}

func (vc *VersionBitsConditionChecker) Mask(params *msg.BitcoinParams) int32 {
	return int32(1) << uint(params.Deployments[vc.id].Bit)
}

func GetStateFor(vc AbstractThresholdConditionChecker, indexPrev *core.BlockIndex,
	params *msg.BitcoinParams, cache ThresholdConditionCache) ThresholdState {

	nPeriod := vc.Period(params)
	nThreshold := vc.Threshold(params)
	nTimeStart := vc.BeginTime(params)
	nTimeTimeout := vc.EndTime(params)

	// A block's state is always the same as that of the first of its period, so
	// it is computed based on a indexPrev whose height equals a multiple of
	// nPeriod - 1.
	if indexPrev != nil {
		indexPrev = indexPrev.GetAncestor(indexPrev.Height - (indexPrev.Height+1)%nPeriod)
	}

	// Walk backwards in steps of nPeriod to find a indexPrev whose information
	// is known
	toCompute := make([]*core.BlockIndex, 0)
	for {
		if _, ok := cache[indexPrev]; !ok {
			if indexPrev == nil {
				// The genesis block is by definition defined.
				cache[indexPrev] = ThresholdDefined
				break
			}
			if indexPrev.GetMedianTimePast() < nTimeStart {
				// Optimization: don't recompute down further, as we know every
				// earlier block will be before the start time
				cache[indexPrev] = ThresholdDefined
				break
			}
			toCompute = append(toCompute, indexPrev)
			indexPrev = indexPrev.GetAncestor(indexPrev.Height - nPeriod)
		} else {
			break
		}
	}

	// At this point, cache[indexPrev] is known
	state, ok := cache[indexPrev]
	if !ok {
		panic("there should be a element in cache")
	}

	// Now walk forward and compute the state of descendants of indexPrev
	for n := 0; n < len(toCompute); n++ {
		stateNext := state
		indexPrev = toCompute[len(toCompute)-1]
		toCompute = toCompute[:(len(toCompute) - 1)]

		switch state {
		case ThresholdDefined:
			{
				if indexPrev.GetMedianTimePast() >= nTimeTimeout {
					stateNext = ThresholdFailed
				} else if indexPrev.GetMedianTimePast() >= nTimeStart {
					stateNext = ThresholdStarted
				}
			}
		case ThresholdStarted:
			{
				if indexPrev.GetMedianTimePast() >= nTimeTimeout {
					stateNext = ThresholdFailed
					break
				}
				// We need to count
				indexCount := indexPrev
				count := 0
				for i := 0; i < nPeriod; i++ {
					if vc.Condition(indexCount, params) {
						count++
					}
					indexCount = indexCount.Prev
				}
				if count >= nThreshold {
					stateNext = ThresholdLockedIn
				}
			}
		case ThresholdLockedIn:
			{
				// Always progresses into ACTIVE.
				stateNext = ThresholdActive
			}
		case ThresholdFailed:
		case ThresholdActive:
			{
				// Nothing happens, these are terminal states.
			}
		}
		state = stateNext
		cache[indexPrev] = state
	}
	return state
}

func GetStateSinceHeightFor(vc AbstractThresholdConditionChecker, indexPrev *core.BlockIndex, params *msg.BitcoinParams, cache ThresholdConditionCache) int {
	initialState := GetStateFor(vc, indexPrev, params, cache)
	// BIP 9 about state DEFINED: "The genesis block is by definition in this
	// state for each deployment."
	if initialState == ThresholdDefined {
		return 0
	}

	nPeriod := vc.Period(params)
	// A block's state is always the same as that of the first of its period, so
	// it is computed based on a indexPrev whose height equals a multiple of
	// nPeriod - 1. To ease understanding of the following height calculation,
	// it helps to remember that right now indexPrev points to the block prior
	// to the block that we are computing for, thus: if we are computing for the
	// last block of a period, then indexPrev points to the second to last
	// block of the period, and if we are computing for the first block of a
	// period, then indexPrev points to the last block of the previous period.
	// The parent of the genesis block is represented by nullptr.
	indexPrev = indexPrev.GetAncestor(indexPrev.Height - ((indexPrev.Height + 1) % nPeriod))
	previousPeriodParent := indexPrev.GetAncestor(indexPrev.Height - nPeriod)
	for previousPeriodParent != nil && GetStateFor(vc, previousPeriodParent, params, cache) == initialState {
		indexPrev = previousPeriodParent
		previousPeriodParent = indexPrev.GetAncestor(indexPrev.Height - nPeriod)
	}

	// Adjust the result because right now we point to the parent block.
	return indexPrev.Height + 1
}

type WarningBitsConditionChecker struct {
	bit int
}

func NewWarningBitsConChecker(bitIn int) *WarningBitsConditionChecker {
	w := new(WarningBitsConditionChecker)
	w.bit = bitIn
	return w
}

func (w *WarningBitsConditionChecker) BeginTime(params *msg.BitcoinParams) int64 {
	return 0
}

func (w *WarningBitsConditionChecker) EndTime(params *msg.BitcoinParams) int64 {
	return math.MaxInt64
}

func (w *WarningBitsConditionChecker) Period(params *msg.BitcoinParams) int {
	return int(params.MinerConfirmationWindow)
}

func (w *WarningBitsConditionChecker) Threshold(params *msg.BitcoinParams) int {
	return int(params.RuleChangeActivationThreshold)
}

func (w *WarningBitsConditionChecker) Condition(index *core.BlockIndex, params *msg.BitcoinParams) bool {

	return int64(index.Header.Version)&VersionBitsTopMask == VersionBitsTopBits &&
		((index.Header.Version)>>uint(w.bit))&1 != 0 &&
		(ComputeBlockVersion(index.Prev, params, VBCache)>>uint(w.bit))&1 == 0
}

func ComputeBlockVersion(indexPrev *core.BlockIndex, params *msg.BitcoinParams, t *VersionBitsCache) int {
	version := VersionBitsTopBits

	for i := 0; i < int(consensus.MaxVersionBitsDeployments); i++ {
		state := func() ThresholdState {
			t.Lock()
			defer t.Unlock()
			v := VersionBitsState(indexPrev, params, consensus.DeploymentPos(i), t)
			return v
		}()

		if state == ThresholdLockedIn || state == ThresholdStarted {
			version |= int(VersionBitsMask(params, consensus.DeploymentPos(i)))
		}
	}

	return version
}

func init() {
	VBCache = NewVersionBitsCache()
}
