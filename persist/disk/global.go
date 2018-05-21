package disk

import (
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
)
type BlockFileInfoMap map[int] *block.BlockFileInfo
type BlockIndexMap  map[int] *blockindex.BlockIndex
var GlobalBlockFileInfoMap = make(BlockFileInfoMap)

var GlobalBlockIndexMap = make(BlockIndexMap)

var GlobalLastBlockFile int = 0

var GlobalLastWrite,GlobalLastFlush,GlobalLastSetChain = 0,0,0

var DefaultMaxMemPoolSize uint = 300
var GlobalSetDirtyFileInfo = make(map[int]bool)
var GlobalTimeReadFromDisk int64 = 0

const (
	/** The maximum size of a blk?????.dat file (since 0.8) */
	MaxBlockFileSize = 0x8000000
	/** The pre-allocation chunk size for blk?????.dat files (since 0.8)  预先分配的文件大小*/
	BlockFileChunkSize = 0x1000000
	/** The pre-allocation chunk size for rev?????.dat files (since 0.8) */
	UndoFileChunkSize  = 0x100000
)