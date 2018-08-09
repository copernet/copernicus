package blockindex

import (
	"bytes"
	"math/rand"
	"time"
	"reflect"
	"testing"

	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chainparams"
)

const SkipListLength = 30000

func TestBlockIndexGetAncestor(t *testing.T) {
	vIndex := make([]BlockIndex, SkipListLength)

	for i := 0; i < SkipListLength; i++ {
		vIndex[i].Height = int32(i)
		if i == 0 {
			vIndex[i].Prev = nil
		} else {
			vIndex[i].Prev = &vIndex[i-1]
		}
		//vIndex[i].BuildSkip()
	}

	/*for i := 0; i < SkipListLength; i++ {
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
	}*/
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
	var bIndex BlockIndex
	testValue := uint32(1324)
	bIndex.TimeMax = testValue
	if bIndex.GetBlockTimeMax() != testValue {
		t.Errorf("GetBlockTimeMax is wrong")
	}
}

func TestWaitingData(t *testing.T) {
	var bIndex BlockIndex
	bIndex.Status = StatusWaitingData
	if !bIndex.WaitingData() {
		t.Errorf("WaitingData is wrong")
	}
}

func TestAllValid(t *testing.T) {
	var bIndex BlockIndex
	bIndex.Status = StatusAllValid
	if !bIndex.AllValid() {
		t.Errorf("AllValid is wrong")
	}
}

func TestIndexStored(t *testing.T) {
	var bIndex BlockIndex
	bIndex.Status = StatusIndexStored
	if !bIndex.IndexStored() {
		t.Errorf("IndexStored is wrong")
	}
}

func TestAllStored(t *testing.T) {
	var bIndex BlockIndex
	bIndex.Status = StatusDataStored
	if !bIndex.AllStored() {
		t.Errorf("AllStored is wrong")
	}
}

func TestAccepted(t *testing.T) {
	var bIndex BlockIndex
	bIndex.Status = StatusAccepted
	if !bIndex.Accepted() {
		t.Errorf("Accepted is wrong")
	}
}

func TestFailed(t *testing.T) {
	var bIndex BlockIndex
	bIndex.Status = StatusFailed
	if !bIndex.Failed() {
		t.Errorf("Failed is wrong")
	}
}

func TestGetUndoPos(t *testing.T) {
	var bIndex BlockIndex
	testInt := int32(34536)
	testUint := uint32(53645)
	bIndex.File = testInt
	bIndex.UndoPos = testUint
	var ret = bIndex.GetUndoPos()
	if ret.File != testInt || ret.Pos != testUint {
		t.Errorf("TestGetUndoPos is wrong")
	}
}

func TestGetBlockPos(t *testing.T) {
	var bIndex BlockIndex
	testInt := int32(34536)
	testUint := uint32(53645)
	bIndex.File = testInt
	bIndex.DataPos = testUint
	var ret = bIndex.GetBlockPos()
	if ret.File != testInt || ret.Pos != testUint {
		t.Errorf("TestGetDataPos is wrong")
	}
}

func TestGetBlockHash(t *testing.T) {
	var bIndex BlockIndex
	var testHash util.Hash
	bIndex.SetBlockHash(testHash)
	if *bIndex.GetBlockHash() != testHash {
		t.Errorf("GetBlockHash is wrong")
	}
}

func TestAddStatus(t *testing.T) {
	var bIndex BlockIndex
	testStatu := uint32(34432)
	testStatus := uint32(5666)
	bIndex.Status = testStatus
	bIndex.AddStatus(testStatu)
	if bIndex.Status & testStatu != testStatu {
		t.Errorf("AddStatus is wrong")
	}
}

func TestSubStatus(t *testing.T) {
	var bIndex BlockIndex
	testStatu := uint32(34432)
	testStatus := uint32(5666)
	bIndex.Status = testStatus
	bIndex.SubStatus(testStatu)
	if bIndex.Status & testStatu != 0 {
		t.Errorf("SubStatus is wrong")
	}
}

func TestGetBlockHeader(t *testing.T) {
	var bIndex BlockIndex
	if bIndex.GetBlockHeader() != &bIndex.Header {
		t.Errorf("GetBlockHeader is wrong")
	}

}

func TestIsGenesis(t *testing.T) {
	var params chainparams.BitcoinParams
	var bIndex BlockIndex
	params.GenesisBlock = new(block.Block)
	bIndex.SetBlockHash(params.GenesisBlock.GetHash())
	if !bIndex.IsGenesis(&params) {
		t.Errorf("IsGenesis is wrong")
	}

}

func TestIsReplayProtectionEnabled(t *testing.T) {
	var params chainparams.BitcoinParams
	var bIndex BlockIndex
	params.MagneticAnomalyActivationTime = bIndex.GetMedianTimePast() + 1
	if bIndex.IsReplayProtectionEnabled(&params) {
		t.Errorf("IsCashHFEnabled is wrong, got true, want false")
	}
	params.MagneticAnomalyActivationTime = bIndex.GetMedianTimePast()
	if !bIndex.IsReplayProtectionEnabled(&params) {
		t.Errorf("IsCashHFEnabled is wrong, got false, want true")
	}
}

func TestNewBlockIndex(t *testing.T) {
	var bHeader block.BlockHeader
	var bIndex = NewBlockIndex(&bHeader)
	if bIndex.Header != bHeader || bIndex.GetBlockTimeMax() != 0 ||
		bIndex.GetBlockPos().Pos != 0 || bIndex.GetDataPos() != 0 ||
		bIndex.GetUndoPos().Pos != 0 {
		t.Errorf("NewBlockIndex is wrong")
	}
}

func TestGetMedianTimePast(t *testing.T) {
	BlocksMain := make([]BlockIndex, 11)
	Times := [medianTimeSpan]uint32 {3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5}
	for i:= 0; i < medianTimeSpan; i++ {
		BlocksMain[i].Header.Time = Times[i]
		if i > 0 {
			BlocksMain[i].Prev = &BlocksMain[i-1]
		} else {
			BlocksMain[i].Prev = nil
		}
	}
	ret := BlocksMain[medianTimeSpan - 1].GetMedianTimePast()
	want := int64(4)
	if ret != want {
		t.Errorf("GetMedianTimePast is wrong, got %d, want %d", ret, want)
	}
	ret = BlocksMain[medianTimeSpan - 4].GetMedianTimePast()
	want = int64(4)
	if ret != want {
		t.Errorf("GetMedianTimePast is wrong, got %d, want %d", ret, want)
	}
}

func TestSerialize(t *testing.T) {
	var bIndex1, bIndex2 BlockIndex
	buf := bytes.NewBuffer(nil);
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 100; i++ {
		bIndex1.Height = r.Int31()
		bIndex1.Status = r.Uint32()
		bIndex1.TxCount = r.Int31()
		bIndex1.File = r.Int31()
		bIndex1.DataPos = r.Uint32()
		bIndex1.UndoPos = r.Uint32()
		err := bIndex1.Serialize(buf)
		if err != nil {
			t.Error(err)
		}
		err = bIndex2.Unserialize(buf)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(bIndex1, bIndex2) {
			t.Errorf("Unserialize after Serialize returns differently bIndex1=%#v, bIndex2=%#v",
				bIndex1, bIndex2)
			return
		}
	}
}