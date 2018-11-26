package versionbits

import (
	"github.com/magiconair/properties/assert"
	"testing"

	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/util"
)

var paramsDummy = model.BitcoinParams{}

func testTime(Height int) int64 {
	return 1415926536 + int64(600*Height)
}

type ConditionChecker struct {
	cache ThresholdConditionCache
}

var randomNum = util.InsecureRand32()

func (tc *ConditionChecker) BeginTime(params *model.BitcoinParams) int64 {
	return testTime(10000)
}

func (tc *ConditionChecker) EndTime(params *model.BitcoinParams) int64 {
	return testTime(20000)
}
func (tc *ConditionChecker) Period(params *model.BitcoinParams) int {
	return 1000
}
func (tc *ConditionChecker) Threshold(params *model.BitcoinParams) int {
	return 900
}
func (tc *ConditionChecker) Condition(index *blockindex.BlockIndex, params *model.BitcoinParams) bool {
	return index.Header.Version&0x100 != 0
}

func (tc *ConditionChecker) GetStateFor(indexPrev *blockindex.BlockIndex) ThresholdState {
	return GetStateFor(tc, indexPrev, &paramsDummy, tc.cache)
}

const CHECKERS = 6

type VersionBitsTester struct {
	// Test counter (to identify failures)
	num int
	// A fake blockchain
	block []*blockindex.BlockIndex
	// 6 independent checkers for the same bit.
	// The first one performs all checks, the second only 50%, the third only
	// 25%, etc...
	// This is to test whether lack of cached information leads to the same
	// results.
	checker [CHECKERS]ConditionChecker
}

func newVersionBitsTester() *VersionBitsTester {
	vt := VersionBitsTester{
		num:   0,
		block: make([]*blockindex.BlockIndex, 0),
	}

	var checker [CHECKERS]ConditionChecker
	for i := 0; i < CHECKERS; i++ {
		checker[i] = ConditionChecker{cache: make(ThresholdConditionCache)}
	}
	vt.checker = checker
	return &vt
}

func (versionBitsTester *VersionBitsTester) Tip() *blockindex.BlockIndex {
	if len(versionBitsTester.block) == 0 {
		return nil
	}
	return versionBitsTester.block[len(versionBitsTester.block)-1]
}

func (versionBitsTester *VersionBitsTester) Reset() *VersionBitsTester {
	versionBitsTester.block = make([]*blockindex.BlockIndex, 0)
	for i := 0; i < CHECKERS; i++ {
		versionBitsTester.checker[i] = ConditionChecker{}
		versionBitsTester.checker[i].cache = make(ThresholdConditionCache)
	}

	return versionBitsTester
}

// Mine the block, util the blockChain height equal height - 1.
func (versionBitsTester *VersionBitsTester) Mine(height int, nTime int64, nVersion int32) *VersionBitsTester {
	for len(versionBitsTester.block) < height {
		index := &blockindex.BlockIndex{}
		index.SetNull()
		index.Height = int32(len(versionBitsTester.block))
		index.Prev = nil
		if len(versionBitsTester.block) > 0 {
			index.Prev = versionBitsTester.block[len(versionBitsTester.block)-1]
		}
		index.Header.Time = uint32(nTime)
		index.Header.Version = nVersion
		//index.BuildSkip()
		versionBitsTester.block = append(versionBitsTester.block, index)
	}
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestDefined(t *testing.T) *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (randomNum & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != ThresholdDefined {
				t.Errorf("Test %d for DEFINED, actual state : %d, expect state : THRESHOLD_DEFINED\n",
					versionBitsTester.num, tmpThreshold)
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestStarted(t *testing.T) *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (randomNum & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != ThresholdStarted {
				t.Errorf("Test %d for STARTED, actual state : %d, expect state : THRESHOLD_STARTED\n",
					versionBitsTester.num, tmpThreshold)
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestLockedIn(t *testing.T) *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (randomNum & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != ThresholdLockedIn {
				t.Errorf("Test %d for LOCKED_IN, actual state : %d, expect state : THRESHOLD_LOCKED_IN\n",
					versionBitsTester.num, tmpThreshold)
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestActive(t *testing.T) *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (randomNum & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != ThresholdActive {
				t.Errorf("Test %d for ACTIVE, actual state : %d, expect state : THRESHOLD_ACTIVE\n", versionBitsTester.num, tmpThreshold)
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestFailed(t *testing.T) *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (randomNum & ((1 << uint(i)) - 1)) == 0 {
			var tmpThreshold ThresholdState
			if len(versionBitsTester.block) == 0 {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}
			if tmpThreshold != ThresholdFailed {
				t.Errorf("Test %d for FAILED, actual state : %d, expect state : THRESHOLD_FAILED\n", versionBitsTester.num, tmpThreshold)
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func TestVersionBitsComputeBlockVersion(t *testing.T) {
	assert.Equal(t, int32(VersionBitsTopBits), ComputeBlockVersion())
}
