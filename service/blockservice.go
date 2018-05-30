package service

import (
	"github.com/btcboost/copernicus/log"
	lblock "github.com/btcboost/copernicus/logic/block"
	lchain "github.com/btcboost/copernicus/logic/chain"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/persist/global"

)


func ProcessBlockHeader(headerList []*block.BlockHeader, lastIndex *blockindex.BlockIndex) error {
	log.Debug("ProcessBlockHeader======%#v", headerList)
	for _, header := range headerList{
		index, err :=  lchain.AcceptBlockHeader(header)
		if err != nil{
			return err
		}
		lastIndex = index
	}
	return nil
}

func ProcessBlock(b *block.Block) (bool, error) {
	log.Debug("ProcessBlock==%#v", b)
	gChain := chain.GetInstance()
	isNewBlock := false
	var err error

	bIndex := gChain.FindBlockIndex(b.GetHash())
	if bIndex != nil {
		if bIndex.Accepted() {
			return isNewBlock,nil
		}
	}

	ret := ProcessNewBlock(b, true, &isNewBlock)
	// bIndex,err = lchain.AcceptBlock(b, &params)
	if !ret {
		return isNewBlock, err
	}
	

	return isNewBlock, err
}


func ProcessNewBlock(pblock *block.Block, fForceProcessing bool, fNewBlock *bool) bool {

	if fNewBlock != nil {
		*fNewBlock = false
	}
	state := block.ValidationState{}
	// Ensure that CheckBlock() passes before calling AcceptBlock, as
	// belt-and-suspenders.
	ret := lblock.CheckBlock(pblock, &state, true, true, )

	global.CsMain.Lock()
	defer global.CsMain.Unlock()
	if ret {
		lchain.AcceptBlock(pblock, &state, fForceProcessing, fNewBlock)
	}
	
	lchain.CheckBlockIndex()
	if !ret {
		// todo !!! add asynchronous notification
		log.Error(" AcceptBlock FAILED ")
		return false
	}

	// Only used to report errors, not invalidity - ignore it
	if !lchain.ActivateBestChain(&state, pblock) {
		log.Error(" ActivateBestChain failed ")
		return false
	}

	return true
}
