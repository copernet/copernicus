package blockchain

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/utils"
)

var paramsDummy = consensus.Params{}

func testTime(Height int) int64 {
	return int64(1415926536 + 600*Height)
}

type testConditionChecker struct {
	cache ThresholdConditionCache
}

func (tc *testConditionChecker) BeginTime(params *consensus.Params) int64 {
	return testTime(10000)
}

func (tc *testConditionChecker) EndTime(params *consensus.Params) int64 {
	return testTime(20000)
}
func (tc *testConditionChecker) Period(params *consensus.Params) int {
	return 1000
}
func (tc *testConditionChecker) Threshold(params *consensus.Params) int {
	return 900
}
func (tc *testConditionChecker) Condition(index *BlockIndex, params *consensus.Params) bool {
	return index.Version&0x100 != 0
}

func (tc *testConditionChecker) GetStateFor(indexPrev *BlockIndex, params *consensus.Params, cache ThresholdConditionCache) ThresholdState {
	nPeriod := tc.Period(params)
	nThreshold := tc.Threshold(params)
	nTimeStart := tc.BeginTime(params)
	nTimeTimeout := tc.EndTime(params)

	// A block's state is always the same as that of the first of its period, so
	// it is computed based on a pindexPretcwhose height equals a multiple of
	// nPeriod - 1.
	if indexPrev != nil {
		indexPrev = indexPrev.GetAncestor(indexPrev.Height - (indexPrev.Height+1)%nPeriod)
	}

	// Walk backwards in steps of nPeriod to find a pindexPretcwhose information
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
					if tc.Condition(indexCount, params) {
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

func (tc *testConditionChecker) GetStateSinceHeightFor(indexPrev *BlockIndex, params *consensus.Params, cache ThresholdConditionCache) int {
	initialState := tc.GetStateFor(indexPrev, params, cache)
	// BIP 9 about state DEFINED: "The genesis block is by definition in this
	// state for each deployment."
	if initialState == THRESHOLD_DEFINED {
		return 0
	}

	nPeriod := tc.Period(params)
	// A block's state is always the same as that of the first of its period, so
	// it is computed based on a pindexPretcwhose height equals a multiple of
	// nPeriod - 1. To ease understanding of the following height calculation,
	// it helps to remember that right now pindexPretcpoints to the block prior
	// to the block that we are computing for, thus: if we are computing for the
	// last block of a period, then pindexPretcpoints to the second to last
	// block of the period, and if we are computing for the first block of a
	// period, then pindexPretcpoints to the last block of the previous period.
	// The parent of the genesis block is represented by nullptr.
	indexPrev = indexPrev.GetAncestor(indexPrev.Height - ((indexPrev.Height + 1) % nPeriod))
	previousPeriodParent := indexPrev.GetAncestor(indexPrev.Height - nPeriod)

	for previousPeriodParent != nil && tc.GetStateFor(previousPeriodParent, params, cache) == initialState {
		indexPrev = previousPeriodParent
		previousPeriodParent = indexPrev.GetAncestor(indexPrev.Height - nPeriod)
	}

	// Adjust the result because right now we point to the parent block.
	return indexPrev.Height + 1
}

const CHECKERS = 6

type VersionBitsTester struct {
	num     int
	block   []*BlockIndex
	checker [CHECKERS]testConditionChecker
}

func (versionBitsTester *VersionBitsTester) Tip() *BlockIndex {
	if len(versionBitsTester.block) == 0 {
		return nil
	}
	return versionBitsTester.block[len(versionBitsTester.block)-1]
}

func (versionBitsTester *VersionBitsTester) Reset() *VersionBitsTester {
	versionBitsTester.block = make([]*BlockIndex, 0)
	for i := 0; i < CHECKERS; i++ {
		versionBitsTester.checker[i] = testConditionChecker{}
	}
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) Mine(height int, nTime int32, nVersion int32) *VersionBitsTester {
	for len(versionBitsTester.block) < height {
		index := &BlockIndex{}
		index.SetNull()
		index.Height = len(versionBitsTester.block)

		if len(versionBitsTester.block) > 0 {
			index.PPrev = versionBitsTester.block[len(versionBitsTester.block)-1]
		} else {
			index.PPrev = nil
		}
		index.Time = uint32(nTime)
		index.Version = nVersion
		index.BuildSkip()
		versionBitsTester.block = append(versionBitsTester.block, index)
	}
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestStateSinceHeight(height int) *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
			var tmpHeight int
			if len(versionBitsTester.block) == 0 {
				tmpHeight = versionBitsTester.checker[i].GetStateSinceHeightFor(nil, &paramsDummy, ThresholdConditionCache{})
			} else {
				tmpHeight = versionBitsTester.checker[i].GetStateSinceHeightFor(versionBitsTester.block[len(versionBitsTester.block)-1], &paramsDummy, ThresholdConditionCache{})
			}

			if tmpHeight == height {
				fmt.Printf("Test %d for StateSinceHeight", versionBitsTester.num)
			} else {
				panic("height should be equal each other")
			}

		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestDefined() *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil, &paramsDummy, ThresholdConditionCache{})
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1], &paramsDummy, ThresholdConditionCache{})
			}

			if tmpThreshold == THRESHOLD_DEFINED {
				fmt.Printf("Test %d for DEFINED", versionBitsTester.num)
			} else {
				panic("threshold should be equal each other")
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestStarted() *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil, &paramsDummy, ThresholdConditionCache{})
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1], &paramsDummy, ThresholdConditionCache{})
			}

			if tmpThreshold == THRESHOLD_STARTED {
				fmt.Printf("Test %d for STARTED", versionBitsTester.num)
			} else {
				panic("threshold should be equal each other")
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestLockedIn() *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil, &paramsDummy, ThresholdConditionCache{})
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1], &paramsDummy, ThresholdConditionCache{})
			}

			if tmpThreshold == THRESHOLD_LOCKED_IN {
				fmt.Printf("Test %d for LOCKED_IN", versionBitsTester.num)
			} else {
				panic("threshold should be equal each other")
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestActive() *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil, &paramsDummy, ThresholdConditionCache{})
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1], &paramsDummy, ThresholdConditionCache{})
			}

			if tmpThreshold == THRESHOLD_ACTIVE {
				fmt.Printf("Test %d for ACTIVE", versionBitsTester.num)
			} else {
				panic("threshold should be equal each other")
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestFailed() *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil, &paramsDummy, ThresholdConditionCache{})
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1], &paramsDummy, ThresholdConditionCache{})
			}

			if tmpThreshold == THRESHOLD_FAILED {
				fmt.Printf("Test %d for ACTIVE", versionBitsTester.num)
			} else {
				panic("threshold should be equal each other")
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

// new a insecure rand creator from crypto/rand seed
func newInsecureRand() []byte {
	randByte := make([]byte, 32)
	_, err := rand.Read(randByte)
	if err != nil {
		panic("init rand number creator failed...")
	}
	return randByte
}

// GetRandHash create a random Hash(utils.Hash)
func GetRandHash() *utils.Hash {
	tmpStr := hex.EncodeToString(newInsecureRand())
	return utils.HashFromString(tmpStr)
}

// InsecureRandRange create a random number in [0, limit]
func InsecureRandRange(limit uint64) uint64 {
	if limit == 0 {
		fmt.Println("param 0 will be insignificant")
		return 0
	}
	r := newInsecureRand()
	return binary.LittleEndian.Uint64(r) % (limit + 1)
}

// InsecureRand32 create a random number in [0 math.MaxUint32]
func InsecureRand32() uint32 {
	r := newInsecureRand()
	return binary.LittleEndian.Uint32(r)
}

// InsecureRandBits create a random number following  specified bit count
func InsecureRandBits(bit uint8) uint64 {
	r := newInsecureRand()
	maxNum := uint64(((1<<(bit-1))-1)*2 + 1 + 1)
	return binary.LittleEndian.Uint64(r) % maxNum
}

// InsecureRandBool create true or false randomly
func InsecureRandBool() bool {
	r := newInsecureRand()
	remainder := binary.LittleEndian.Uint16(r) % 2
	return remainder == 1
}
