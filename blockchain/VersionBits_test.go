package blockchain

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	//"testing"

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

func (tc *testConditionChecker) GetStateFor(indexPrev *BlockIndex) ThresholdState {
	return GetStateFor(tc, indexPrev, &paramsDummy, tc.cache)
}

func (tc *testConditionChecker) GetStateSinceHeightFor(indexPrev *BlockIndex) int {
	return GetStateSinceHeightFor(tc, indexPrev, &paramsDummy, tc.cache)
}

const CHECKERS = 6

type VersionBitsTester struct {
	// Test counter (to identify failures)
	num int
	// A fake blockchain
	block []*BlockIndex
	// 6 independent checkers for the same bit.
	// The first one performs all checks, the second only 50%, the third only
	// 25%, etc...
	// This is to test whether lack of cached information leads to the same
	// results.
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
		versionBitsTester.checker[i].cache = make(ThresholdConditionCache)
	}

	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) Mine(height int, nTime int32, nVersion int32) *VersionBitsTester {
	for len(versionBitsTester.block) < height {
		index := &BlockIndex{}
		index.SetNull()
		index.Height = len(versionBitsTester.block)
		index.PPrev = nil
		if len(versionBitsTester.block) > 0 {
			index.PPrev = versionBitsTester.block[len(versionBitsTester.block)-1]
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
				tmpHeight = versionBitsTester.checker[i].GetStateSinceHeightFor(nil)
			} else {
				tmpHeight = versionBitsTester.checker[i].GetStateSinceHeightFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpHeight == height {
				fmt.Printf("Test %d for StateSinceHeight", versionBitsTester.num)
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
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != THRESHOLD_DEFINED {
				fmt.Printf("Test %d for DEFINED", versionBitsTester.num)
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
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != THRESHOLD_STARTED {
				fmt.Printf("Test %d for STARTED", versionBitsTester.num)
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
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != THRESHOLD_LOCKED_IN {
				fmt.Printf("Test %d for LOCKED_IN", versionBitsTester.num)
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
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != THRESHOLD_ACTIVE {
				fmt.Printf("Test %d for ACTIVE", versionBitsTester.num)
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
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(nil)
			} else {
				tmpThreshold = versionBitsTester.checker[i].GetStateFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpThreshold != THRESHOLD_FAILED {
				fmt.Printf("Test %d for ACTIVE", versionBitsTester.num)
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

/*
func TestVersionBitsState(t *testing.T) {
	for i:= 0; i < 64; i++{
		v := VersionBitsTester{num:0}
		v.TestDefined().
			TestStateSinceHeight(0).
				Mine(1, int32(testTime(1)), 0x100).
					TestDefined().TestStateSinceHeight(0).
						Mine(11, int32(testTime(11)), 0x100).
							TestDefined().TestStateSinceHeight(0).
								Mine(989, int32(testTime(989)),0x100).
									TestDefined().TestStateSinceHeight(0).
										Mine(999, int32(testTime(20000)), 0x100).
											TestDefined().TestStateSinceHeight(0).
												Mine(1000, int32(testTime(20000)), 0x100).
													TestFailed().TestStateSinceHeight(1000).
														Mine(1999, int32(testTime(30001)), 0x100).
															TestFailed().TestStateSinceHeight(1000).
																Mine(2000, int32(testTime(30002)), 0x100).
																	TestFailed().TestStateSinceHeight(1000).
																		Mine(2001, int32(testTime(30003)), 0x100).
																			TestFailed().TestStateSinceHeight(1000).
																				Mine(2999, int32(testTime(30004)), 0x100).
																					TestFailed().TestStateSinceHeight(1000).
																						Mine(3000, int32(testTime(30005)), 0x100).
																							TestFailed().TestStateSinceHeight(1000)

	}

}
*/
