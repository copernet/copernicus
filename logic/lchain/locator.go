package lchain

import (
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/util"
)

const (
	MaxHeadersResults = 2000
	MaxBlocksResults  = 500
)

func LocateBlocks(locator *chain.BlockLocator, endHash *util.Hash) []util.Hash {
	persist.CsMain.Lock()
	defer persist.CsMain.Unlock()
	var bi *blockindex.BlockIndex
	gChain := chain.GetInstance()
	ret := make([]util.Hash, 0)

	bi = FindForkInGlobalIndex(gChain, locator)
	if bi != nil {
		bi = gChain.Next(bi)
	}

	nLimits := MaxBlocksResults
	for ; bi != nil && nLimits > 0; bi = gChain.Next(bi) {
		if bi.GetBlockHash().IsEqual(endHash) {
			break
		}
		ret = append(ret, *bi.GetBlockHash())
		nLimits--
	}
	return ret
}

func LocateHeaders(locator *chain.BlockLocator, endHash *util.Hash) []block.BlockHeader {
	persist.CsMain.Lock()
	defer persist.CsMain.Unlock()
	var bi *blockindex.BlockIndex
	gChain := chain.GetInstance()
	ret := make([]block.BlockHeader, 0)
	if locator.IsNull() {
		bi = gChain.FindBlockIndex(*endHash)
		if bi == nil {
			return ret
		}
	} else {
		bi = FindForkInGlobalIndex(gChain, locator)
		if bi != nil {
			bi = gChain.Next(bi)
		}
	}
	nLimits := MaxHeadersResults
	for ; bi != nil && nLimits > 0; bi = gChain.Next(bi) {
		if bi.GetBlockHash().IsEqual(endHash) {
			break
		}
		bh := bi.GetBlockHeader()
		ret = append(ret, *bh)
		nLimits--
	}
	return ret
}

func FindForkInGlobalIndex(chains *chain.Chain, locator *chain.BlockLocator) *blockindex.BlockIndex {
	gChain := chain.GetInstance()
	// Find the first block the caller has in the main chain
	for _, hash := range locator.GetBlockHashList() {
		bi := gChain.FindBlockIndex(hash)
		if bi != nil {
			if chains.Contains(bi) {
				return bi
			}
			if bi.GetAncestor(chains.Height()) == chains.Tip() {
				return chains.Tip()
			}
		}
	}
	return chains.Genesis()
}
