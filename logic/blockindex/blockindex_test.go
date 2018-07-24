package blockindex

import (
	"math"
	"math/big"
	"testing"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/util"
)

func TestBlockIndexBuildSkip(t *testing.T) {
	HashMain := make([]*big.Int, 10000)
	BlocksMain := make([]blockindex.BlockIndex, 10000)
	tmpRand := util.NewFastRandomContext(false)

	for i := 0; i < cap(BlocksMain); i++ {
		// Set the hash equal to the height
		HashMain[i] = big.NewInt(int64(i))
		BlocksMain[i].Height = int32(i)
		if i > 0 {
			BlocksMain[i].Prev = &BlocksMain[i-1]
		} else {
			BlocksMain[i].Prev = nil
		}

		//BlocksMain[i].BuildSkip()
		if i < 10 {
			BlocksMain[i].TimeMax = uint32(i)
			BlocksMain[i].Header.Time = uint32(i)
		} else {
			// randomly choose something in the range [MTP, MTP*2]
			medianTimePast := uint32(BlocksMain[i].GetMedianTimePast())
			r := tmpRand.Rand32() % medianTimePast
			BlocksMain[i].Header.Time = r + medianTimePast
			BlocksMain[i].TimeMax = uint32(math.Max(float64(BlocksMain[i].Header.Time), float64(BlocksMain[i-1].TimeMax)))
		}
	}

	// Check that we set nTimeMax up correctly.
	curTimeMax := uint32(0)
	for i := 0; i < len(BlocksMain); i++ {
		curTimeMax = uint32(math.Max(float64(curTimeMax), float64(BlocksMain[i].Header.Time)))
		if curTimeMax != BlocksMain[i].TimeMax {
			t.Errorf("the two element should be equal, left value : %d, right value : %d",
				curTimeMax, BlocksMain[i].TimeMax)
			return
		}
	}

	// Build a CChain for the main branch.
	chains := chain.Chain{}
	chains.SetTip(&BlocksMain[len(BlocksMain)-1])

	for i := 0; i < len(BlocksMain); i++ {
		// Pick a random element in BlocksMain.
		r := tmpRand.Rand32() % uint32(len(BlocksMain))
		testTime := BlocksMain[r].Header.Time
		ret := chains.FindEarliestAtLeast(int64(testTime))
		if ret == nil {
			continue
		}
		if ret.TimeMax < testTime {
			t.Errorf("ret addr : %p, ret.TimeMax : %d, should greater or equal testTime : %d",
				ret, ret.TimeMax, testTime)
			return
		}
		if ret.Prev != nil && ret.Prev.TimeMax > testTime {
			t.Errorf("ret.pprev : %p should be nil or ret.pprev.TimeMax : %d should be "+
				"less testTime : %d", ret.Prev, ret.Prev.TimeMax, testTime)
			return
		}
		if r < uint32(ret.Height) {
			continue
		}
		if BlocksMain[r].GetAncestor(ret.Height) != ret {
			t.Errorf("GetAncestor() return value : %p should be equal ret : %p, "+
				"the find height : %d, r : %d", BlocksMain[r].GetAncestor(ret.Height), ret, ret.Height, r)
			return
		}

	}

}
