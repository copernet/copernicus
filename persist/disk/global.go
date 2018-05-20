package disk

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
)
type vfile uint32
type BlockFileInfoMap map[vfile] *block.BlockFileInfo
type BlockIndexMap  map[vfile] *blockindex.BlockIndex
var GlobalBlockFileInfoMap = new(BlockFileInfoMap)

var GlobalBlockIndexMap = new(BlockIndexMap)

var GlobalLastBlockFile vfile = 0

var GlobalLastWrite,GlobalLastFlush,GlobalLastSetChain = 0,0,0


