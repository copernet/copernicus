package service

import (
	"fmt"
	"github.com/copernet/copernicus/log"
	lblock "github.com/copernet/copernicus/logic/block"
	lchain "github.com/copernet/copernicus/logic/chain"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist/global"
)

func ProcessBlockHeader(headerList []*block.BlockHeader, lastIndex *blockindex.BlockIndex) error {
	log.Debug("ProcessBlockHeader begin, header number : %d", len(headerList))
	for _, header := range headerList {
		index, err := lblock.AcceptBlockHeader(header)
		if err != nil {
			return err
		}
		lastIndex = index
	}
	beginHash := headerList[0].GetHash()
	endHash := headerList[len(headerList)-1].GetHash()
	log.Trace("processBlockHeader success, blockNumber : %d, lastBlockHeight : %d, beginBlockHash : %s, "+
		"endBlockHash : %s. ", len(headerList), lastIndex.Height, beginHash.String(), endHash.String())
	return nil
}

func ProcessBlock(b *block.Block) (bool, error) {
	h := b.GetHash()
	gChain := chain.GetInstance()
	coinsTip := utxo.GetUtxoCacheInstance()
	coinsTipHash, _ := coinsTip.GetBestBlock()

	log.Trace("Begin processing block: %s, Global Chain height: %d, tipHash: %s, coinsTip hash: %s",
		h.String(), gChain.Height(), gChain.Tip().GetBlockHash().String(), coinsTipHash.String())

	isNewBlock := false
	var err error

	//bIndex := gChain.FindBlockIndex(h)
	//if bIndex != nil {
	//	if bIndex.Accepted() {
	//		log.Trace("this block have be sucessed process, height: %d, hash: %s",
	//			bIndex.Height, bIndex.GetBlockHash().String())
	//		return isNewBlock, nil
	//	}
	//}
	//log.Trace("gchan height : %d, begin to processNewBlock ...", gChain.Height())
	err = ProcessNewBlock(b, true, &isNewBlock)
	// bIndex,err = lchain.AcceptBlock(b, &params)
	if err != nil {
		log.Trace("processBlock failed ...")
		return isNewBlock, err
	}

	coinsTipHash, _ = coinsTip.GetBestBlock()
	log.Trace("After process block: %s, Global Chain height: %d, tipHash: %s, coinsTip hash: %s",
		h.String(), gChain.Height(), gChain.Tip().GetBlockHash().String(), coinsTipHash.String())

	fmt.Printf("Processed block: %s, Chain height: %d, tipHash: %s, coinsTip hash: %s\n",
		h.String(), gChain.Height(), gChain.Tip().GetBlockHash().String(), coinsTipHash.String())

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
		_, _, err = lblock.AcceptBlock(pblock, fForceProcessing, fNewBlock)
	}
	if err != nil {
		// todo !!! add asynchronous notification
		log.Error(" AcceptBlock FAILED ")
		return err
	}

	lchain.CheckBlockIndex()

	// Only used to report errors, not invalidity - ignore it
	if err = lchain.ActivateBestChain(pblock); err != nil {
		log.Error(" ActivateBestChain failed :%v", err)
		return err
	}

	return nil
}
