package blockchain

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"
	//"testing"

	"github.com/btcboost/copernicus/msg"
	"github.com/btcboost/copernicus/utils"
)

var paramsDummy = msg.BitcoinParams{}

func testTime(Height int) int64 {
	return int64(1415926536 + 600*Height)
}

type ConditionChecker struct {
	cache ThresholdConditionCache
}

var tcc = ConditionChecker{cache: make(ThresholdConditionCache)}

func (tc *ConditionChecker) BeginTime(params *msg.BitcoinParams) int64 {
	return testTime(10000)
}

func (tc *ConditionChecker) EndTime(params *msg.BitcoinParams) int64 {
	return testTime(20000)
}
func (tc *ConditionChecker) Period(params *msg.BitcoinParams) int {
	return 1000
}
func (tc *ConditionChecker) Threshold(params *msg.BitcoinParams) int {
	return 900
}
func (tc *ConditionChecker) Condition(index *BlockIndex, params *msg.BitcoinParams) bool {
	return index.Version&0x100 != 0
}

func (tc *ConditionChecker) GetStateFor(indexPrev *BlockIndex) ThresholdState {
	return GetStateFor(tc, indexPrev, &paramsDummy, tc.cache)
}

func (tc *ConditionChecker) GetStateSinceHeightFor(indexPrev *BlockIndex) int {
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
	checker [CHECKERS]ConditionChecker
}

func newVersionBitsTester() *VersionBitsTester {
	vt := VersionBitsTester{
		num:   0,
		block: make([]*BlockIndex, 0),
	}

	var checker [CHECKERS]ConditionChecker
	for i := 0; i < CHECKERS; i++ {
		checker[i] = ConditionChecker{cache: make(ThresholdConditionCache)}
	}
	vt.checker = checker
	return &vt
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
		versionBitsTester.checker[i] = ConditionChecker{}
		versionBitsTester.checker[i].cache = make(ThresholdConditionCache)
	}

	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) Mine(height int, nTime int64, nVersion int32) *VersionBitsTester {
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

func TestVersionBits(t *testing.T) {
	for i := 0; i < 64; i++ {
		// DEFINED -> FAILED
		vt := newVersionBitsTester()
		vt.
			TestDefined().
			TestStateSinceHeight(0).Mine(1, testTime(1), 0x100).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(11, testTime(11), 0x100).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(989, testTime(989), 0x100).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(999, testTime(20000), 0x100).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(1000, testTime(20000), 0x100).
			TestFailed().
			TestStateSinceHeight(1000).
			Mine(1999, testTime(30001), 0x100).
			TestFailed().
			TestStateSinceHeight(1000).
			Mine(2000, testTime(30002), 0x100).
			TestFailed().
			TestStateSinceHeight(1000).
			Mine(2001, testTime(30003), 0x100).
			TestFailed().
			TestStateSinceHeight(1000).
			Mine(2999, testTime(30004), 0x100).
			TestFailed().
			TestStateSinceHeight(1000).
			Mine(3000, testTime(30005), 0x100).
			TestFailed().
			TestStateSinceHeight(1000).

			// DEFINED -> STARTED -> FAILED
			Reset().
			TestDefined().
			TestStateSinceHeight(0).
			Mine(1, testTime(1), 0).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(1000, testTime(10000)-1, 0x100).
			TestDefined().
			// One second more and it would be defined
			TestStateSinceHeight(0).
			Mine(2000, testTime(10000), 0x100).
			TestStarted().
			// So that's what happens the next period
			TestStateSinceHeight(2000).
			Mine(2051, testTime(10010), 0).
			TestStarted().
			// 51 old blocks
			TestStateSinceHeight(2000).
			Mine(2950, testTime(10020), 0x100).
			TestStarted().
			// 899 new blocks
			TestStateSinceHeight(2000).
			Mine(3000, testTime(20000), 0).
			TestFailed().
			// 50 old blocks (so 899 out of the past 1000)
			TestStateSinceHeight(3000).
			Mine(4000, testTime(20010), 0x100).
			TestFailed().
			TestStateSinceHeight(3000).

			// DEFINED -> STARTED -> FAILED while threshold reached
			Reset().
			TestDefined().
			TestStateSinceHeight(0).
			Mine(1, testTime(1), 0).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(1000, testTime(10000)-1, 0x101).
			TestDefined().
			// One second more and it would be defined
			TestStateSinceHeight(0).
			Mine(2000, testTime(10000), 0x101).
			TestStarted().
			// So that's what happens the next period
			TestStateSinceHeight(2000).
			Mine(2999, testTime(30000), 0x100).
			TestStarted().
			// 999 new blocks
			TestStateSinceHeight(2000).
			Mine(3000, testTime(30000), 0x100).
			TestFailed().
			// 1 new block (so 1000 out of the past 1000 are new)
			TestStateSinceHeight(3000).
			Mine(3999, testTime(30001), 0).
			TestFailed().
			TestStateSinceHeight(3000).
			Mine(4000, testTime(30002), 0).
			TestFailed().
			TestStateSinceHeight(3000).
			Mine(14333, testTime(30003), 0).
			TestFailed().
			TestStateSinceHeight(3000).
			Mine(24000, testTime(40000), 0).
			TestFailed().
			TestStateSinceHeight(3000).

			// DEFINED -> STARTED -> LOCKEDIN at the last minute -> ACTIVE
			Reset().
			TestDefined().
			Mine(1, testTime(1), 0).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(1000, testTime(10000)-1, 0x101).
			TestDefined().
			// One second more and it would be defined
			TestStateSinceHeight(0).
			Mine(2000, testTime(10000), 0x101).
			TestStarted().
			// So that's what happens the next period
			TestStateSinceHeight(2000).
			Mine(2050, testTime(10010), 0x200).
			TestStarted().
			// 50 old blocks
			TestStateSinceHeight(2000).
			Mine(2950, testTime(10020), 0x100).
			TestStarted().
			// 900 new blocks
			TestStateSinceHeight(2000).
			Mine(2999, testTime(19999), 0x200).
			TestStarted().
			// 49 old blocks
			TestStateSinceHeight(2000).
			Mine(3000, testTime(29999), 0x200).
			TestLockedIn().
			// 1 old block (so 900 out of the past 1000)
			TestStateSinceHeight(3000).
			Mine(3999, testTime(30001), 0).
			TestLockedIn().
			TestStateSinceHeight(3000).
			Mine(4000, testTime(30002), 0).
			TestActive().
			TestStateSinceHeight(4000).
			Mine(14333, testTime(30003), 0).
			TestActive().
			TestStateSinceHeight(4000).
			Mine(24000, testTime(40000), 0).
			TestActive().
			TestStateSinceHeight(4000).

			// DEFINED multiple periods -> STARTED multiple periods -> FAILED
			Reset().
			TestDefined().
			TestStateSinceHeight(0).
			Mine(999, testTime(999), 0).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(1000, testTime(1000), 0).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(2000, testTime(2000), 0).
			TestDefined().
			TestStateSinceHeight(0).
			Mine(3000, testTime(10000), 0).
			TestStarted().
			TestStateSinceHeight(3000).
			Mine(4000, testTime(10000), 0).
			TestStarted().
			TestStateSinceHeight(3000).
			Mine(5000, testTime(10000), 0).
			TestStarted().
			TestStateSinceHeight(3000).
			Mine(6000, testTime(20000), 0).
			TestFailed().
			TestStateSinceHeight(6000).
			Mine(7000, testTime(20000), 0x100).
			TestFailed().
			TestStateSinceHeight(6000)
	}

	// Sanity checks of version bit deployments
	mainnetParams := msg.MainNetParams
	for i := 0; i < int(msg.MAX_VERSION_BITS_DEPLOYMENTS); i++ {
		bitmask := VersionBitsMask(&mainnetParams, msg.DeploymentPos(i))
		// Make sure that no deployment tries to set an invalid bit.
		if int64(bitmask)&int64(^VERSIONBITS_TOP_MASK) != int64(bitmask) {
			t.Error("there is am invalid bit to be set")
		}

		// Verify that the deployment windows of different deployment using the
		// same bit are disjoint. This test may need modification at such time
		// as a new deployment is proposed that reuses the bit of an activated
		// soft fork, before the end time of that soft fork.  (Alternatively,
		// the end time of that activated soft fork could be later changed to be
		// earlier to avoid overlap.)
		for j := i + 1; j < int(msg.MAX_VERSION_BITS_DEPLOYMENTS); j++ {
			if VersionBitsMask(&mainnetParams, msg.DeploymentPos(j)) == bitmask {
				if !(mainnetParams.Deployments[j].StartTime > mainnetParams.Deployments[i].Timeout || mainnetParams.Deployments[i].StartTime > mainnetParams.Deployments[j].Timeout) {
					t.Error("logic error")
				}
			}
		}
	}
}
