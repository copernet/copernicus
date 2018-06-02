package service

import (
	"fmt"
	
	"github.com/btcboost/copernicus/log"
	lblock "github.com/btcboost/copernicus/logic/block"
	lchain "github.com/btcboost/copernicus/logic/chain"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/persist/global"

)


func ProcessBlockHeader(headerList []*block.BlockHeader, lastIndex *blockindex.BlockIndex) error {
	log.Debug("ProcessBlockHeader======%#v", headerList)
	for _, header := range headerList{
		index, err :=  lblock.AcceptBlockHeader(header)
		if err != nil{
			return err
		}
		lastIndex = index
	}
	beginHash := headerList[0].GetHash()
	endHash := headerList[len(headerList) - 1].GetHash()
	log.Trace("processBlockHeader success, blockNumber : %d, lastBlockHeight : %d, beginBlockHash : %s, " +
		"endBlockHash : %s. ", len(headerList), lastIndex.Height, beginHash.String(), endHash.String())
	return nil
}

func ProcessBlock(b *block.Block) (bool, error) {
	gChain := chain.GetInstance()
	coinsTip := utxo.GetUtxoCacheInstance()
	fmt.Println("gchan==%d====%#v",gChain.Height(),gChain.Tip(), coinsTip)
	
	isNewBlock := false
	var err error

	bIndex := gChain.FindBlockIndex(b.GetHash())
	if bIndex != nil {
		if bIndex.Accepted() {
			return isNewBlock,nil
		}
	}

	err = ProcessNewBlock(b, true, &isNewBlock)
	// bIndex,err = lchain.AcceptBlock(b, &params)
	if err!=nil {
		return isNewBlock, err
	}
	return isNewBlock, err
}


func ProcessNewBlock(pblock *block.Block, fForceProcessing bool, fNewBlock *bool) error {

	if fNewBlock != nil {
		*fNewBlock = false
	}
	// Ensure that CheckBlock() passes before calling AcceptBlock, as
	// belt-and-suspenders.
	err := lblock.CheckBlock(pblock)
	global.CsMain.Lock()
	defer global.CsMain.Unlock()
	if err == nil {
		_,_,err = lblock.AcceptBlock(pblock,fForceProcessing, fNewBlock)
	}
	
	lchain.CheckBlockIndex()
	if err!=nil {
		// todo !!! add asynchronous notification
		log.Error(" AcceptBlock FAILED ")
		return err
	}

	// Only used to report errors, not invalidity - ignore it
	if err = lchain.ActivateBestChain(pblock);err!=nil {
		log.Error(" ActivateBestChain failed ")
		return err
	}

	return nil
}
