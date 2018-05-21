package chain

import (
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/persist/disk"
	"github.com/btcboost/copernicus/model/blockindex"
)

func FindForkInGlobalIndex(chain *chain.Chain, locator *chain.BlockLocator) *blockindex.BlockIndex {
	// Find the first block the caller has in the main chain
	for _, hash := range locator.GetBlockHashList() {
		mi, ok := disk.GlobalBlockIndexMap[hash]
		if ok {
			if chain.Contains(mi) {
				return mi
			}
			if mi.GetAncestor(chain.Height()) == chain.Tip() {
				return chain.Tip()
			}
		}
	}
	return chain.Genesis()
}


