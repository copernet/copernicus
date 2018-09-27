package mining

import "sync/atomic"

var (
	maxBlockSize  = DefaultMaxBlockSize
	//blockPriorityPercentage uint64		// not be used at current version
)

func GetBlockSize() uint64 {
	return atomic.LoadUint64(&maxBlockSize)
}

func SetBlockSize(size uint64) {
	atomic.StoreUint64(&maxBlockSize, size)
}
