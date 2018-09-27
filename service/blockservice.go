package service

import (
	"fmt"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblock"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"time"
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

	err := ProcessNewBlock(b, true, &isNewBlock)

	if err != nil {
		log.Trace("processBlock failed ...")
		return isNewBlock, err
	}

	coinsTipHash, _ = coinsTip.GetBestBlock()
	log.Trace("After process block: %s, Global Chain height: %d, tipHash: %s, coinsTip hash: %s",
		h.String(), gChain.Height(), gChain.Tip().GetBlockHash().String(), coinsTipHash.String())

	fmt.Printf("Processed block: %s, Chain height: %d, tipHash: %s, coinsTip hash: %s currenttime:%v\n",
		h.String(), gChain.Height(), gChain.Tip().GetBlockHash().String(), coinsTipHash.String(), time.Now())

	return isNewBlock, err
}

func ProcessNewBlock(pblock *block.Block, fForceProcessing bool, fNewBlock *bool) error {

	if fNewBlock != nil {
		*fNewBlock = false
	}

	// Ensure that CheckBlock() passes before calling AcceptBlock, as
	// belt-and-suspenders.
	if err := lblock.CheckBlock(pblock, true, true); err != nil {
		log.Error("check block failed, please check: %v", err)
		return err
	}
	persist.CsMain.Lock()
	defer persist.CsMain.Unlock()

	if _, _, err := lblock.AcceptBlock(pblock, fForceProcessing, fNewBlock); err != nil {
		h := pblock.GetHash()
		log.Error(" AcceptBlock FAILED: %s", h.String())
		return err
	}

	chain.GetInstance().SendNotification(chain.NTBlockAccepted, pblock)

	if err := lchain.CheckBlockIndex(); err != nil {
		log.Error("check block index failed, please check: %v", err)
		return err
	}

	// Only used to report errors, not invalidity - ignore it
	if err := lchain.ActivateBestChain(pblock); err != nil {
		log.Error(" ActivateBestChain failed :%v", err)
		return err
	}

	return nil
}
