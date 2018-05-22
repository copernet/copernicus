package disk

const (
	/** The maximum size of a blk?????.dat file (since 0.8) */
	MaxBlockFileSize = 0x8000000
	/** The pre-allocation chunk size for blk?????.dat files (since 0.8)  预先分配的文件大小*/
	BlockFileChunkSize = 0x1000000
	/** The pre-allocation chunk size for rev?????.dat files (since 0.8) */
	UndoFileChunkSize = 0x100000
	DefaultMaxMemPoolSize =300
)
