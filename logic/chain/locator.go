package chain

import (
	"github.com/btcboost/copernicus/model/block"
	mchain "github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/persist/global"
	"github.com/btcboost/copernicus/util"
)

const (
	MaxHeadersResults = 2000
	MaxBlocksResults = 500

)
func LocateBlocks(locator *mchain.BlockLocator, endHash *util.Hash) []util.Hash {
	global.CsMain.Lock()
	defer global.CsMain.Unlock()
	var bi *blockindex.BlockIndex
	gChain := mchain.GetInstance()
	ret := make([]util.Hash, 0)
	
	bi = FindForkInGlobalIndex(gChain, locator)
	if bi != nil{
		bi =  gChain.Next(bi)
	}
	
	nLimits := MaxBlocksResults
	for ;bi != nil && nLimits >0 ;bi = gChain.Next(bi){
		if bi.GetBlockHash().IsEqual(endHash){
			break
		}
		ret = append(ret, *bi.GetBlockHash())
		nLimits -= 1
	}
	return ret
}

func LocateHeaders(locator *mchain.BlockLocator, endHash *util.Hash) []block.BlockHeader {
	global.CsMain.Lock()
	defer global.CsMain.Unlock()
	var bi *blockindex.BlockIndex
	gChain := mchain.GetInstance()
	ret := make([]block.BlockHeader, 0)
	if locator.IsNull(){
		bi = gChain.FindBlockIndex(*endHash)
		if bi == nil{
			return ret
		}
	}else{
		bi = FindForkInGlobalIndex(gChain, locator)
		if bi != nil{
			bi =  gChain.Next(bi)
		}
	}
	nLimits := MaxHeadersResults
	for ;bi != nil && nLimits >0 ;bi = gChain.Next(bi){
		if bi.GetBlockHash().IsEqual(endHash){
			break
		}
		bh := bi.GetBlockHeader()
		ret = append(ret, *bh)
		nLimits -= 1
	}
	return ret
}

func FindForkInGlobalIndex(chain *mchain.Chain, locator *mchain.BlockLocator) *blockindex.BlockIndex {
	gChain := mchain.GetInstance()
	// Find the first block the caller has in the main chain
	for _, hash := range locator.GetBlockHashList() {
		bi := gChain.FindBlockIndex(hash)
		if bi != nil {
			if chain.Contains(bi) {
				return bi
			}
			if bi.GetAncestor(chain.Height()) == chain.Tip() {
				return chain.Tip()
			}
		}
	}
	return chain.Genesis()
}


