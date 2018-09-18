package versionbits

import (
	"testing"

	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chainparams"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/util"
)

var paramsDummy = chainparams.BitcoinParams{}

func testTime(Height int) int64 {
	return 1415926536 + int64(600*Height)
}

type ConditionChecker struct {
	cache ThresholdConditionCache
}

// var tcc = ConditionChecker{cache: make(ThresholdConditionCache)}

func (tc *ConditionChecker) BeginTime(params *chainparams.BitcoinParams) int64 {
	return testTime(10000)
}

func (tc *ConditionChecker) EndTime(params *chainparams.BitcoinParams) int64 {
	return testTime(20000)
}
func (tc *ConditionChecker) Period(params *chainparams.BitcoinParams) int {
	return 1000
}
func (tc *ConditionChecker) Threshold(params *chainparams.BitcoinParams) int {
	return 900
}
func (tc *ConditionChecker) Condition(index *blockindex.BlockIndex, params *chainparams.BitcoinParams) bool {
	return index.Header.Version&0x100 != 0
}

func (tc *ConditionChecker) GetStateFor(indexPrev *blockindex.BlockIndex) ThresholdState {
	return GetStateFor(tc, indexPrev, &paramsDummy, tc.cache)
}

func (tc *ConditionChecker) GetStateSinceHeightFor(indexPrev *blockindex.BlockIndex) int {
	return GetStateSinceHeightFor(tc, indexPrev, &paramsDummy, tc.cache)
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

func (versionBitsTester *VersionBitsTester) TestStateSinceHeight(height int, t *testing.T) *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (util.InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
			var tmpHeight int
			if len(versionBitsTester.block) == 0 {
				tmpHeight = versionBitsTester.checker[i].GetStateSinceHeightFor(nil)
			} else {
				tmpHeight = versionBitsTester.checker[i].GetStateSinceHeightFor(versionBitsTester.block[len(versionBitsTester.block)-1])
			}

			if tmpHeight != height {
				t.Errorf("Test %d for StateSinceHeight, actual height : %d, expect height : %d\n",
					versionBitsTester.num, tmpHeight, height)
			}
		}
	}
	versionBitsTester.num++
	return versionBitsTester
}

func (versionBitsTester *VersionBitsTester) TestDefined(t *testing.T) *VersionBitsTester {
	for i := 0; i < CHECKERS; i++ {
		if (util.InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
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
		if (util.InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
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
		if (util.InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
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
		if (util.InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
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
		if (util.InsecureRand32() & ((1 << uint(i)) - 1)) == 0 {
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

func TestVersionBits(t *testing.T) {
	for i := 0; i < 10; i++ {
		// DEFINED -> FAILED
		vt := newVersionBitsTester()
		vt.TestDefined(t).
			TestStateSinceHeight(0, t).Mine(1, testTime(1), 0x100).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(11, testTime(11), 0x100).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(989, testTime(989), 0x100).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(999, testTime(20000), 0x100).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(1000, testTime(20000), 0x100).
			TestFailed(t).
			TestStateSinceHeight(1000, t).
			Mine(1999, testTime(30001), 0x100).
			TestFailed(t).
			TestStateSinceHeight(1000, t).
			Mine(2000, testTime(30002), 0x100).
			TestFailed(t).
			TestStateSinceHeight(1000, t).
			Mine(2001, testTime(30003), 0x100).
			TestFailed(t).
			TestStateSinceHeight(1000, t).
			Mine(2999, testTime(30004), 0x100).
			TestFailed(t).
			TestStateSinceHeight(1000, t).
			Mine(3000, testTime(30005), 0x100).
			TestFailed(t).
			TestStateSinceHeight(1000, t).

			// DEFINED -> STARTED -> FAILED
			Reset().
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(1, testTime(1), 0).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(1000, testTime(10000)-1, 0x100).
			TestDefined(t).
			// One second more and it would be defined
			TestStateSinceHeight(0, t).
			Mine(2000, testTime(10000), 0x100).
			TestStarted(t).
			// So that's what happens the next period
			TestStateSinceHeight(2000, t).
			Mine(2051, testTime(10010), 0).
			TestStarted(t).
			// 51 old blocks
			TestStateSinceHeight(2000, t).
			Mine(2950, testTime(10020), 0x100).
			TestStarted(t).
			// 899 new blocks
			TestStateSinceHeight(2000, t).
			Mine(3000, testTime(20000), 0).
			//TestFailed(t).
			//// 50 old blocks (so 899 out of the past 1000)
			//TestStateSinceHeight(3000, t).
			//
			//Mine(4000, testTime(20010), 0x100).
			//TestFailed(t).
			//TestStateSinceHeight(3000, t).

			// DEFINED -> STARTED -> FAILED while threshold reached
			Reset().
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(1, testTime(1), 0).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(1000, testTime(10000)-1, 0x101).
			TestDefined(t).
			// One second more and it would be defined
			TestStateSinceHeight(0, t).
			Mine(2000, testTime(10000), 0x101).
			TestStarted(t).
			// So that's what happens the next period
			TestStateSinceHeight(2000, t).
			Mine(2999, testTime(30000), 0x100).
			TestStarted(t).
			// 999 new blocks
			//TestStateSinceHeight(2000, t).
			//Mine(3000, testTime(30000), 0x100).
			//TestFailed(t).
			////1 new block (so 1000 out of the past 1000 are new)
			//TestStateSinceHeight(3000, t).
			//Mine(3999, testTime(30001), 0).
			//TestFailed(t).
			//TestStateSinceHeight(3000, t).
			//Mine(4000, testTime(30002), 0).
			//TestFailed(t).
			//TestStateSinceHeight(3000, t).
			//Mine(14333, testTime(30003), 0).
			//TestFailed(t).
			//TestStateSinceHeight(3000, t).
			//Mine(24000, testTime(40000), 0).
			//TestFailed(t).
			//TestStateSinceHeight(3000, t).

			// DEFINED -> STARTED -> LOCKEDIN at the last minute -> ACTIVE
			Reset().
			TestDefined(t).
			Mine(1, testTime(1), 0).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(1000, testTime(10000)-1, 0x101).
			TestDefined(t).
			// One second more and it would be defined
			TestStateSinceHeight(0, t).
			Mine(2000, testTime(10000), 0x101).
			TestStarted(t).
			// So that's what happens the next period
			TestStateSinceHeight(2000, t).
			Mine(2050, testTime(10010), 0x200).
			TestStarted(t).
			// 50 old blocks
			TestStateSinceHeight(2000, t).
			Mine(2950, testTime(10020), 0x100).
			TestStarted(t).
			// 900 new blocks
			TestStateSinceHeight(2000, t).
			Mine(2999, testTime(19999), 0x200).
			TestStarted(t).
			// 49 old blocks
			TestStateSinceHeight(2000, t).
			Mine(3000, testTime(29999), 0x200).
			//TestLockedIn(t).
			//// 1 old block (so 900 out of the past 1000)
			//TestStateSinceHeight(3000, t).
			//Mine(3999, testTime(30001), 0).
			//TestLockedIn(t).
			//TestStateSinceHeight(3000, t).
			//Mine(4000, testTime(30002), 0).
			//TestActive(t).
			//TestStateSinceHeight(4000, t).
			//Mine(14333, testTime(30003), 0).
			//TestActive(t).
			//TestStateSinceHeight(4000, t).
			//Mine(24000, testTime(40000), 0).
			//TestActive(t).
			//TestStateSinceHeight(4000, t).

			// DEFINED multiple periods -> STARTED multiple periods -> FAILED
			Reset().
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(999, testTime(999), 0).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(1000, testTime(1000), 0).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(2000, testTime(2000), 0).
			TestDefined(t).
			TestStateSinceHeight(0, t).
			Mine(3000, testTime(10000), 0).
			TestStarted(t).
			TestStateSinceHeight(3000, t).
			Mine(4000, testTime(10000), 0).
			TestStarted(t).
			TestStateSinceHeight(3000, t).
			Mine(5000, testTime(10000), 0).
			TestStarted(t).
			TestStateSinceHeight(3000, t).
			Mine(6000, testTime(20000), 0)
		//TestFailed(t).
		//TestStateSinceHeight(6000, t).
		//Mine(7000, testTime(20000), 0x100).
		//TestFailed(t).
		//TestStateSinceHeight(6000, t)
	}

	// Sanity checks of version bit deployments
	mainnetParams := chainparams.MainNetParams
	for i := 0; i < int(consensus.MaxVersionBitsDeployments); i++ {
		bitmask := VersionBitsMask(&mainnetParams, consensus.DeploymentPos(i))
		// Make sure that no deployment tries to set an invalid bit.
		if int64(bitmask)&int64(^VersionBitsTopMask) != int64(bitmask) {
			t.Error("there is am invalid bit to be set")
		}

		// Verify that the deployment windows of different deployment using the
		// same bit are disjoint. This test may need modification at such time
		// as a new deployment is proposed that reuses the bit of an activated
		// soft fork, before the end time of that soft fork.  (Alternatively,
		// the end time of that activated soft fork could be later changed to be
		// earlier to avoid overlap.)
		for j := i + 1; j < int(consensus.MaxVersionBitsDeployments); j++ {
			if VersionBitsMask(&mainnetParams, consensus.DeploymentPos(j)) == bitmask {
				if !(mainnetParams.Deployments[j].StartTime > mainnetParams.Deployments[i].Timeout || mainnetParams.Deployments[i].StartTime > mainnetParams.Deployments[j].Timeout) {
					t.Error("logic error")
				}
			}
		}
	}
}

func TestVersionBitsComputeBlockVersion(t *testing.T) {
	vbc := NewVersionBitsCache()

	// Check that ComputeBlockVersion will set the appropriate bit correctly on mainnet.
	mainnetParams := chainparams.MainNetParams

	// Use the TESTDUMMY deployment for testing purposes.
	bit := mainnetParams.Deployments[consensus.DeploymentTestDummy].Bit
	startTime := mainnetParams.Deployments[consensus.DeploymentTestDummy].StartTime
	timeout := mainnetParams.Deployments[consensus.DeploymentTestDummy].Timeout

	if startTime >= timeout {
		t.Error("startTime should be less than timeout value")
	}

	// In the first chain, test that the bit is set by CBV until it has failed.
	// In the second chain, test the bit is set by CBV while STARTED and
	// LOCKED-IN, and then no longer set while ACTIVE.
	firstChain := newVersionBitsTester()
	secondChain := newVersionBitsTester()

	// Start generating blocks before nStartTime
	Time := startTime - 1

	// Before MedianTimePast of the chain has crossed nStartTime, the bit
	// should not be set.
	lastBlock := firstChain.Mine(2016, Time, VersionBitsLastOldBlockVersion).Tip()
	if (ComputeBlockVersion(lastBlock, &mainnetParams, vbc) & (1 << uint(bit))) != 0 {
		t.Error("expect the next block version & (1<<uint(bit) is 0")
		return
	}

	// Now , the next block version bit should be VERSIONBITS_TOP_BITS
	// Mine 2011 more blocks at the old time, and check that CBV isn't setting
	// the bit yet.
	for i := 1; i < 2012; i++ {
		lastBlock = firstChain.Mine(2016+i, Time, VersionBitsLastOldBlockVersion).Tip()
		// This works because VERSIONBITS_LAST_OLD_BLOCK_VERSION happens to be
		// 4, and the bit we're testing happens to be bit 28.
		if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) != 0 {
			t.Error("expect the next block version & (1<<uint(bit) is 0")
			return
		}
	}

	// Now mine 5 more blocks at the start time -- MTP should not have passed
	// yet, so CBV should still not yet set the bit.
	Time = startTime
	for i := 2012; i < 2016; i++ {
		lastBlock = firstChain.Mine(2016+i, Time, VersionBitsLastOldBlockVersion).Tip()
		if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) != 0 {
			t.Error("expect the next block version & (1<<uint(bit) is 0")
			return
		}
	}

	// Advance to the next period and transition to STARTED,
	lastBlock = firstChain.Mine(6048, Time, VersionBitsLastOldBlockVersion).Tip()
	// so ComputeBlockVersion should now set the bit,
	if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) == 0 {
		t.Error("the bit should have been set")
	}

	// and should also be using the VERSIONBITS_TOP_BITS.
	if int64(ComputeBlockVersion(lastBlock, &mainnetParams, vbc))&VersionBitsTopMask != VersionBitsTopBits {
		t.Error("the bit should be set VERSIONBITS_TOP_BITS")
		return
	}

	// Check that ComputeBlockVersion will set the bit until nTimeout
	Time += 600
	// test blocks for up to 2 time periods
	blocksToMine := 4032
	Height := 6048
	// These blocks are all before nTimeout is reached.
	for Time < timeout && blocksToMine > 0 {
		lastBlock = firstChain.Mine(Height+1, Time, VersionBitsLastOldBlockVersion).Tip()
		if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) == 0 {
			t.Error("the bit should be set 1<<28")
			return
		}
		if int64(ComputeBlockVersion(lastBlock, &mainnetParams, vbc))&VersionBitsTopMask != VersionBitsTopBits {
			t.Error("the bit should be set VERSIONBITS_TOP_BITS")
			return
		}
		blocksToMine--
		Time += 600
		Height++
	}

	Time = timeout
	// FAILED is only triggered at the end of a period, so CBV should be setting
	// the bit until the period transition.
	for i := 0; i < 2015; i++ {
		lastBlock = firstChain.Mine(Height+1, Time, VersionBitsLastOldBlockVersion).Tip()
		if (ComputeBlockVersion(lastBlock, &mainnetParams, vbc) & (1 << uint(bit))) == 0 {
			t.Error("error")
		}
		Height++
	}

	// The next block should trigger no longer setting the bit.
	lastBlock = firstChain.Mine(Height+1, Time, VersionBitsLastOldBlockVersion).Tip()
	if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) != 0 {
		t.Error("the bit should not set")
	}

	// On a new chain:
	// verify that the bit will be set after lock-in, and then stop being set
	// after activation.
	Time = startTime

	// Mine one period worth of blocks, and check that the bit will be on for
	// the next period.
	lastBlock = secondChain.Mine(2016, startTime, VersionBitsLastOldBlockVersion).Tip()
	if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) == 0 {
		t.Error("the bit should be set, because the state is started")
	}

	// Mine another period worth of blocks, signaling the new bit.
	lastBlock = secondChain.Mine(4032, startTime, VersionBitsTopBits|(1<<uint(bit))).Tip()
	// After one period of setting the bit on each block, it should have locked
	// in.
	// We keep setting the bit for one more period though, until activation.
	if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) == 0 {
		t.Error("Now the bit state is lock_in, so the bit should be set in next block ")
		return
	}

	// Now check that we keep mining the block until the end of this period, and
	// then stop at the beginning of the next period.
	lastBlock = secondChain.Mine(6047, startTime, VersionBitsLastOldBlockVersion).Tip()
	if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) == 0 {
		t.Error("Now the bit statr is active, so the bit should be set in next Block ")
		return
	}
	lastBlock = secondChain.Mine(6048, startTime, VersionBitsLastOldBlockVersion).Tip()
	if ComputeBlockVersion(lastBlock, &mainnetParams, vbc)&(1<<uint(bit)) != 0 {
		t.Error("error")
	}

	// Finally, verify that after a soft fork has activated, CBV no longer uses
	// VERSIONBITS_LAST_OLD_BLOCK_VERSION.
	// BOOST_CHECK_EQUAL(ComputeBlockVersion(lastBlock, mainnetParams) &
	// VERSIONBITS_TOP_MASK, VERSIONBITS_TOP_BITS);
	if int64(ComputeBlockVersion(lastBlock, &mainnetParams, vbc))&VersionBitsTopMask != VersionBitsTopBits {
		t.Errorf("when state eqaul active, the bit should not be set ")
	}

}
