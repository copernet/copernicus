package global

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/util"
)

var chainGlobal *ChainGlobal

type BlockFileInfoMap map[int]*block.BlockFileInfo
type BlockIndexMap map[util.Hash]*blockindex.BlockIndex
type ChainGlobal struct {
	GlobalBlockFileInfoMap                               BlockFileInfoMap   // storage all blockfileinfo
	GlobalBlockIndexMap                                  BlockIndexMap      // storage all block index
	GlobalLastBlockFile                                  int                //last block file no.
	GlobalLastWrite, GlobalLastFlush, GlobalLastSetChain int                // last update time
	GlobalBlocksUnlinkedMap                              map[*blockindex.BlockIndex]*blockindex.BlockIndex
	DefaultMaxMemPoolSize                                uint
	GlobalSetDirtyFileInfo                               map[int]bool      // temp for update file info
	GlobalTimeReadFromDisk                               int64
	GlobalReindex                                        bool
	GlobalTxIndex                                        bool
	GlobalHavePruned                                     bool
	
}

func InitChainGlobal() *ChainGlobal {
	cg := new(ChainGlobal)
	cg.GlobalBlockFileInfoMap = make(BlockFileInfoMap)
	cg.GlobalBlockIndexMap = make(BlockIndexMap)
	cg.GlobalBlocksUnlinkedMap = make(map[*blockindex.BlockIndex]*blockindex.BlockIndex)
	cg.GlobalSetDirtyFileInfo = make(map[int]bool)
	return cg
}

func GetChainGlobalInstance() *ChainGlobal {
	if chainGlobal == nil {
		chainGlobal = InitChainGlobal()
	}
	return chainGlobal
}
