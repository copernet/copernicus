package core

import (
	"math"
	"math/big"
	"testing"

	"github.com/btcboost/copernicus/utils"
)

const SkipListLength = 300000

func TestBlockIndexGetAncestor(t *testing.T) {
	vIndex := make([]BlockIndex, SkipListLength)

	for i := 0; i < SkipListLength; i++ {
		vIndex[i].Height = i
		if i == 0 {
			vIndex[i].Prev = nil
		} else {
			vIndex[i].Prev = &vIndex[i-1]
		}
		vIndex[i].BuildSkip()
	}

	for i := 0; i < SkipListLength; i++ {
		if i > 0 {
			if vIndex[i].Skip != &vIndex[vIndex[i].Skip.Height] {
				t.Errorf("the two element addr should be equal, expect %p, but get value : %p",
					vIndex[i].Skip, &vIndex[vIndex[i].Skip.Height])
				return
			}
			if vIndex[i].Skip.Height > i {
				t.Errorf("the skip height : %d should be less the index : %d",
					vIndex[i].Skip.Height, i)
				return
			}
		} else {
			if vIndex[i].Skip != nil {
				t.Errorf("the index : %d pskip should be equal nil, but the actual : %p",
					i, vIndex[i].Skip)
				return
			}
		}
	}
	tmpRand := utils.NewFastRandomContext(false)
	for i := 0; i < 1000; i++ {
		from := tmpRand.Rand32() % (SkipListLength - 1)
		to := tmpRand.Rand32() % (from + 1)

		if vIndex[SkipListLength-1].GetAncestor(int(from)) != &vIndex[from] {
			t.Errorf("the two element should be equal, left value : %p, right value : %p",
				vIndex[SkipListLength-1].GetAncestor(int(from)), &vIndex[from])
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
	HashMain := make([]*big.Int, 10000)
	BlocksMain := make([]BlockIndex, 10000)
	tmpRand := utils.NewFastRandomContext(false)

	for i := 0; i < cap(BlocksMain); i++ {
		// Set the hash equal to the height
		HashMain[i] = big.NewInt(int64(i))
		BlocksMain[i].Height = i
		if i > 0 {
			BlocksMain[i].Prev = &BlocksMain[i-1]
		} else {
			BlocksMain[i].Prev = nil
		}

		BlocksMain[i].BuildSkip()
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
	chain := Chain{}
	chain.SetTip(&BlocksMain[len(BlocksMain)-1])

	// Verify that FindEarliestAtLeast is correct
	for _, v := range chain.Chain {
		_ = v.Height
	}
	for i := 0; i < len(BlocksMain); i++ {
		// Pick a random element in BlocksMain.
		r := tmpRand.Rand32() % uint32(len(BlocksMain))
		testTime := BlocksMain[r].Header.Time
		ret := chain.FindEarliestAtLeast(int64(testTime))
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
