package global

import (
	"sync"
	
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/util"
)

const (
	/** The maximum size of a blk?????.dat file (since 0.8) */
	MaxBlockFileSize = uint32(0x8000000)
	/** The pre-allocation chunk size for blk?????.dat files (since 0.8)  预先分配的文件大小*/
	BlockFileChunkSize = 0x1000000
	/** The pre-allocation chunk size for rev?????.dat files (since 0.8) */
	UndoFileChunkSize = 0x100000
	DefaultMaxMemPoolSize =300
)

var CsMain *sync.RWMutex = new(sync.RWMutex)

var CsLastBlockFile *sync.RWMutex = new(sync.RWMutex)

var persistGlobal *PersistGlobal
type BlockFileInfoList []*block.BlockFileInfo
type DirtyBlockIndex  map[util.Hash]*blockindex.BlockIndex
type PersistGlobal struct {
	GlobalBlockFileInfo         BlockFileInfoList
	GlobalLastBlockFile                                  int                //last block file no.
	GlobalLastWrite, GlobalLastFlush, GlobalLastSetChain int                // last update time
	DefaultMaxMemPoolSize                                uint
	GlobalDirtyFileInfo                               map[int]bool      // temp for update file info
	GlobalDirtyBlockIndex        DirtyBlockIndex
	GlobalTimeReadFromDisk                               int64

	
}


func InitPersistGlobal() *PersistGlobal {
	cg := new(PersistGlobal)
	cg.GlobalBlockFileInfo = make([]*block.BlockFileInfo, 0, 1000)
	cg.GlobalDirtyFileInfo = make(map[int]bool)
	cg.GlobalDirtyBlockIndex = make(DirtyBlockIndex)
	return cg
}

func GetInstance() *PersistGlobal {
	if persistGlobal == nil {
		persistGlobal = InitPersistGlobal()
	}
	return persistGlobal
}
