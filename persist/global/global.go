package global

import (
	"sync"

	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/util"
)

const (
	// MaxBlockFileSize is the maximum size of a blk?????.dat file (since 0.8) */
	MaxBlockFileSize = uint32(0x8000000)
	// BlockFileChunkSize is the pre-allocation chunk size for blk?????.dat files (since 0.8)  预先分配的文件大小*/
	BlockFileChunkSize = 0x1000000
	// UndoFileChunkSize is the pre-allocation chunk size for rev?????.dat files (since 0.8) */
	UndoFileChunkSize     = 0x100000
	DefaultMaxMemPoolSize = 300
)

var CsMain = new(sync.RWMutex)

var CsLastBlockFile = new(sync.RWMutex)

var persistGlobal *PersistGlobal

type BlockFileInfoList []*block.BlockFileInfo
type DirtyBlockIndex map[util.Hash]*blockindex.BlockIndex
type PersistGlobal struct {
	GlobalBlockFileInfo                                  BlockFileInfoList
	GlobalLastBlockFile                                  int32 //last block file no.
	GlobalLastWrite, GlobalLastFlush, GlobalLastSetChain int   // last update time
	DefaultMaxMemPoolSize                                uint
	GlobalDirtyFileInfo                                  map[int32]bool // temp for update file info
	GlobalDirtyBlockIndex                                DirtyBlockIndex
	GlobalTimeReadFromDisk                               int64
	GlobalTimeConnectTotal                               int64
	GlobalTimeChainState                                 int64
	GlobalTimeFlush                                      int64
	GlobalTimeCheck                                      int64
	GlobalTimeForks                                      int64
	GlobalTimePostConnect                                int64
	GlobalTimeTotal                                      int64
	GlobalBlockSequenceID                                int32
}

func (pg *PersistGlobal) AddDirtyBlockIndex(hash util.Hash, pindex *blockindex.BlockIndex) {
	pg.GlobalDirtyBlockIndex[hash] = pindex
}
func (pg *PersistGlobal) AddBlockSequenceID() {
	pg.GlobalBlockSequenceID++
}
func InitPersistGlobal() *PersistGlobal {
	cg := new(PersistGlobal)
	cg.GlobalBlockFileInfo = make([]*block.BlockFileInfo, 0, 1000)
	cg.GlobalDirtyFileInfo = make(map[int32]bool)
	cg.GlobalDirtyBlockIndex = make(DirtyBlockIndex)
	return cg
}

func GetInstance() *PersistGlobal {
	if persistGlobal == nil {
		persistGlobal = InitPersistGlobal()
	}
	return persistGlobal
}
