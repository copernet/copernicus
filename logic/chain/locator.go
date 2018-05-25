package chain

import (
	mchain "github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/util"
)

func LocateBlocks(bl *mchain.BlockLocator, endHash *util.Hash,maxLength int) error {
	return nil
}

func LocateHeaders(bl *mchain.BlockLocator, endHash *util.Hash,maxLength int) error {
	
	return nil
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


