package chain

import (
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/util"
	"testing"
)

func TestChain_GetLocator(t *testing.T) {
	InitGlobalChain()
	tChain := GetInstance()

	tChain.indexMap = make(map[util.Hash]*blockindex.BlockIndex)
	tChain.active = make([]*blockindex.BlockIndex, 0)
	tChain.branch = make([]*blockindex.BlockIndex, 0)
	bIndex := make([]*blockindex.BlockIndex, 50)
	initBits := model.ActiveNetParams.PowLimitBits
	timePerBlock := int64(model.ActiveNetParams.TargetTimePerBlock)
	height := 0

	//Pile up some blocks
	bIndex[0] = blockindex.NewBlockIndex(&model.ActiveNetParams.GenesisBlock.Header)
	tChain.AddToIndexMap(bIndex[0])
	tChain.AddToBranch(bIndex[0])
	tChain.active = append(tChain.active, bIndex[0])

	for height = 1; height < 50; height++ {
		bIndex[height] = getBlockIndex(bIndex[height-1], timePerBlock, initBits)
		tChain.AddToBranch(bIndex[height])
		tChain.AddToIndexMap(bIndex[height])
		tChain.active = append(tChain.active, bIndex[height])
	}

	exp := []util.Hash{
		*bIndex[40].GetBlockHash(),
		*bIndex[39].GetBlockHash(),
		*bIndex[38].GetBlockHash(),
		*bIndex[37].GetBlockHash(),
		*bIndex[36].GetBlockHash(),
		*bIndex[35].GetBlockHash(),
		*bIndex[34].GetBlockHash(),
		*bIndex[33].GetBlockHash(),
		*bIndex[32].GetBlockHash(),
		*bIndex[31].GetBlockHash(),
		*bIndex[30].GetBlockHash(),
		*bIndex[29].GetBlockHash(),
		*bIndex[27].GetBlockHash(),
		*bIndex[23].GetBlockHash(),
		*bIndex[15].GetBlockHash(),
		*bIndex[0].GetBlockHash(),
	}
	locator := tChain.GetLocator(bIndex[40])

	for i, hash := range locator.GetBlockHashList() {
		if hash != exp[i] {
			t.Errorf("GetLocator Error")
		}
	}

	if locator.SetNull(); !locator.IsNull() {
		t.Errorf("Locator setNull failed")
	}

}
