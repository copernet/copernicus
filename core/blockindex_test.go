package core

import (
	"math"
	"math/big"
	"testing"

	"github.com/btcboost/copernicus/utils"
)

const SKIPLIST_LENGTH = 300000

func TestBlockIndexGetAncestor(t *testing.T) {
	vIndex := make([]BlockIndex, SKIPLIST_LENGTH)

	for i := 0; i < SKIPLIST_LENGTH; i++ {
		vIndex[i].Height = i
		if i == 0 {
			vIndex[i].PPrev = nil
		} else {
			vIndex[i].PPrev = &vIndex[i-1]
		}
		vIndex[i].BuildSkip()
	}

	for i := 0; i < SKIPLIST_LENGTH; i++ {
		if i > 0 {
			if vIndex[i].PSkip != &vIndex[vIndex[i].PSkip.Height] {
				t.Errorf("the two element addr should be equal, expect %p, but get value : %p",
					vIndex[i].PSkip, &vIndex[vIndex[i].PSkip.Height])
				return
			}
			if vIndex[i].PSkip.Height > i {
				t.Errorf("the skip height : %d should be less the index : %d",
					vIndex[i].PSkip.Height, i)
				return
			}
		} else {
			if vIndex[i].PSkip != nil {
				t.Errorf("the index : %d pskip should be equal nil, but the actual : %p",
					i, vIndex[i].PSkip)
				return
			}
		}
	}
	tmpRand := utils.NewFastRandomContext(false)
	for i := 0; i < 1000; i++ {
		from := tmpRand.Rand32() % (SKIPLIST_LENGTH - 1)
		to := tmpRand.Rand32() % (from + 1)

		if vIndex[SKIPLIST_LENGTH-1].GetAncestor(int(from)) != &vIndex[from] {
			t.Errorf("the two element should be equal, left value : %p, right value : %p",
				vIndex[SKIPLIST_LENGTH-1].GetAncestor(int(from)), &vIndex[from])
			return
		}
		if vIndex[from].GetAncestor(int(to)) != &vIndex[to] {
			t.Errorf("the two element should be equal, left value : %p, right value : %p",
				vIndex[from].GetAncestor(int(to)), &vIndex[from])
			return
		}
		if vIndex[from].GetAncestor(0) != &vIndex[0] {
			t.Errorf("the two element should be equal, left value : %p, right value : %p",
				vIndex[from].GetAncestor(0), &vIndex[0])
			return
		}
	}

}

func TestBlockIndexBuildSkip(t *testing.T) {
	vHashMain := make([]*big.Int, 10000)
	vBlocksMain := make([]BlockIndex, 10000)
	tmpRand := utils.NewFastRandomContext(false)

	for i := 0; i < cap(vBlocksMain); i++ {
		// Set the hash equal to the height
		vHashMain[i] = big.NewInt(int64(i))
		vBlocksMain[i].Height = i
		if i > 0 {
			vBlocksMain[i].PPrev = &vBlocksMain[i-1]
		} else {
			vBlocksMain[i].PPrev = nil
		}

		vBlocksMain[i].BuildSkip()
		if i < 10 {
			vBlocksMain[i].TimeMax = uint32(i)
			vBlocksMain[i].Time = uint32(i)
		} else {
			// randomly choose something in the range [MTP, MTP*2]
			medianTimePast := uint32(vBlocksMain[i].GetMedianTimePast())
			r := tmpRand.Rand32() % medianTimePast
			vBlocksMain[i].Time = r + medianTimePast
			vBlocksMain[i].TimeMax = uint32(math.Max(float64(vBlocksMain[i].Time), float64(vBlocksMain[i-1].TimeMax)))
		}
	}

	//Check that we set nTimeMax up correctly.
	curTimeMax := uint32(0)
	for i := 0; i < len(vBlocksMain); i++ {
		curTimeMax = uint32(math.Max(float64(curTimeMax), float64(vBlocksMain[i].Time)))
		if curTimeMax != vBlocksMain[i].TimeMax {
			t.Errorf("the two element should be equal, left value : %d, right value : %d",
				curTimeMax, vBlocksMain[i].TimeMax)
			return
		}
	}

	// Build a CChain for the main branch.
	chain := Chain{}
	chain.SetTip(&vBlocksMain[len(vBlocksMain)-1])

	// Verify that FindEarliestAtLeast is correct
	for _, v := range chain.VChain {
		_ = v.Height
	}
	for i := 0; i < len(vBlocksMain); i++ {
		// Pick a random element in vBlocksMain.
		r := tmpRand.Rand32() % uint32(len(vBlocksMain))
		testTime := vBlocksMain[r].Time
		ret := chain.FindEarliestAtLeast(int64(testTime))
		if ret == nil {
			continue
		}
		if ret.TimeMax < testTime {
			t.Errorf("ret addr : %p, ret.TimeMax : %d, should greater or equal testTime : %d",
				ret, ret.TimeMax, testTime)
			return
		}
		if ret.PPrev != nil && ret.PPrev.TimeMax > testTime {
			t.Errorf("ret.pprev : %p should be nil or ret.pprev.TimeMax : %d should be "+
				"less testTime : %d", ret.PPrev, ret.PPrev.TimeMax, testTime)
			return
		}
		if r < uint32(ret.Height) {
			continue
		}
		if vBlocksMain[r].GetAncestor(ret.Height) != ret {
			t.Errorf("GetAncestor() return value : %p should be equal ret : %p, "+
				"the find height : %d, r : %d", vBlocksMain[r].GetAncestor(ret.Height), ret, ret.Height, r)
			return
		}

	}

}
