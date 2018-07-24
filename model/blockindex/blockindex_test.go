package blockindex

import (
	"math"
	"math/big"
	"testing"

	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/util"
)

const SkipListLength = 30000

func TestBlockIndexGetAncestor(t *testing.T) {
	vIndex := make([]blockindex.BlockIndex, SkipListLength)

	for i := 0; i < SkipListLength; i++ {
		vIndex[i].Height = int32(i)
		if i == 0 {
			vIndex[i].Prev = nil
		} else {
			vIndex[i].Prev = &vIndex[i-1]
		}
		vIndex[i].BuildSkip()
	}

	for i := 0; i < SkipListLength; i++ {
		if i > 0 {
			//fmt.Println(vIndex[i].Skip == nil) //nil because not init vIndex[i].Skip
			if vIndex[i].Skip != &vIndex[vIndex[i].Skip.Height] {
				t.Errorf("the two element addr should be equal, expect %p, but get value : %p",
					vIndex[i].Skip, &vIndex[vIndex[i].Skip.Height])
				return
			}
			if vIndex[i].Skip.Height > int32(i) {
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
	tmpRand := util.NewFastRandomContext(false)
	for i := 0; i < 1000; i++ {
		from := tmpRand.Rand32() % (SkipListLength - 1)
		to := tmpRand.Rand32() % (from + 1)

		if vIndex[SkipListLength-1].GetAncestor(int32(from)) != &vIndex[from] {
			t.Errorf("the two element should be equal, left value : %p, right value : %p",
				vIndex[SkipListLength-1].GetAncestor(int32(from)), &vIndex[from])
			return
		}
		if vIndex[from].GetAncestor(int32(to)) != &vIndex[to] {
			t.Errorf("the two element should be equal, left value : %p, right value : %p",
				vIndex[from].GetAncestor(int32(to)), &vIndex[from])
			return
		}
		if vIndex[from].GetAncestor(0) != &vIndex[0] {
			t.Errorf("the two element should be equal, left value : %p, right value : %p",
				vIndex[from].GetAncestor(0), &vIndex[0])
			return
		}
	}

}

func TestGetBlockTimeMax(t *testing.T) {
	var bIndex blockindex.BlockIndex
	var testValue = uint32(1324)
	bIndex.TimeMax = testValue
	if bIndex.GetBlockTimeMax() != testValue {
		t.Errorf("GetBlockTimeMax is wrong")
	}
}

func TestWaitingData(t *testing.T) {
	var bIndex blockindex.BlockIndex
	bIndex.Status = blockindex.StatusWaitingData
	if !bIndex.WaitingData() {
		t.Errorf("WaitingData is wrong")
	}
}

func TestAllValid(t *testing.T) {
	var bIndex blockindex.BlockIndex
	bIndex.Status = blockindex.StatusAllValid
	if !bIndex.AllValid() {
		t.Errorf("AllValid is wrong")
	}
}

func TestIndexStored(t *testing.T) {
	var bIndex blockindex.BlockIndex
	bIndex.Status = blockindex.StatusIndexStored
	if !bIndex.IndexStored() {
		t.Errorf("IndexStored is wrong")
	}
}

func TestAllStored(t *testing.T) {
	var bIndex blockindex.BlockIndex
	bIndex.Status = blockindex.StatusDataStored
	if !bIndex.AllStored() {
		t.Errorf("AllStored is wrong")
	}
}

func TestAccepted(t *testing.T) {
	var bIndex blockindex.BlockIndex
	bIndex.Status = blockindex.StatusAccepted
	if !bIndex.Accepted() {
		t.Errorf("Accepted is wrong")
	}
}

func TestFailed(t *testing.T) {
	var bIndex blockindex.BlockIndex
	bIndex.Status = blockindex.StatusFailed
	if !bIndex.Failed() {
		t.Errorf("Failed is wrong")
	}
}

func TestGetUndoPos(t *testing.T) {
	var bIndex blockindex.BlockIndex
	var testInt = int32(34536)
	var testUint = uint32(53645)
	bIndex.File = testInt
	bIndex.UndoPos = testUint
	var ret = bIndex.GetUndoPos()
	if ret.File != testInt || ret.Pos != testUint {
		t.Errorf("TestGetUndoPos is wrong")
	}
}


func TestGetBlockPos(t *testing.T) {
	var bIndex blockindex.BlockIndex
	var testInt = int32(34536)
	var testUint = uint32(53645)
	bIndex.File = testInt
	bIndex.DataPos = testUint
	var ret = bIndex.GetBlockPos()
	if ret.File != testInt || ret.Pos != testUint {
		t.Errorf("TestGetDataPos is wrong")
	}
}


func TestGetBlockHash(t *testing.T) {
	var bIndex blockindex.BlockIndex
	var testHash util.Hash
	bIndex.SetBlockHash(testHash)
	if *bIndex.GetBlockHash() != testHash {
		t.Errorf("GetBlockHash is wrong")
	}
}


func TestAddStatus(t *testing.T) {
	var bIndex blockindex.BlockIndex
	var testStatu = uint32(34432)
	var testStatus = uint32(5666)
	bIndex.Status = testStatus
	bIndex.AddStatus(testStatu)
	if (bIndex.Status & testStatu != testStatu) {
		t.Errorf("AddStatus is wrong")
	}
}


func TestSubStatus(t *testing.T) {
	var bIndex blockindex.BlockIndex
	var testStatu = uint32(34432)
	var testStatus = uint32(5666)
	bIndex.Status = testStatus
	bIndex.SubStatus(testStatu)
	if (bIndex.Status & testStatu != 0) {
		t.Errorf("SubStatus is wrong")
	}
}

func TestGetBlockHeader(t *testing.T) {
	var bIndex blockindex.BlockIndex
	if bIndex.GetBlockHeader() != &bIndex.Header {
		t.Errorf("GetBlockHeader is wrong")
	}

}

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
