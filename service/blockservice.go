package service

import (
	
	
	"github.com/astaxie/beego/logs"
	lblock "github.com/btcboost/copernicus/logic/block"
	lchain "github.com/btcboost/copernicus/logic/chain"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/persist/global"

)


func ProcessBlockHeader(bl * block.BlockHeader) error {
	return nil
}

func ProcessBlock(b *block.Block) (bool, error) {
	gChain := chain.GetInstance()
	isNewBlock := false
	var err error

	bIndex := gChain.FindBlockIndex(b.GetHash())
	if bIndex != nil {
		if bIndex.Accepted() {
			return isNewBlock,nil
		}
	}

	params := chainparams.MainNetParams
	ret := ProcessNewBlock(&params, b, true, &isNewBlock)
	// bIndex,err = lchain.AcceptBlock(b, &params)
	if !ret {
		return isNewBlock, err
	}
	

	return isNewBlock, err
}


func ProcessNewBlock(param *chainparams.BitcoinParams, pblock *block.Block, fForceProcessing bool, fNewBlock *bool) bool {

	if fNewBlock != nil {
		*fNewBlock = false
	}
	state := block.ValidationState{}
	// Ensure that CheckBlock() passes before calling AcceptBlock, as
	// belt-and-suspenders.
	ret := lblock.CheckBlock(param, pblock, &state, true, true, )

	global.CsMain.Lock()
	defer global.CsMain.Unlock()
	if ret {
		lchain.AcceptBlock(param, pblock, &state, fForceProcessing, fNewBlock)
	}
	
	lchain.CheckBlockIndex(param)
	if !ret {
		// todo !!! add asynchronous notification
		logs.Error(" AcceptBlock FAILED ")
		return false
	}

	// Only used to report errors, not invalidity - ignore it
	if !lchain.ActivateBestChain(param, &state, pblock) {
		logs.Error(" ActivateBestChain failed ")
		return false
	}

	return true
}
