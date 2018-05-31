package btcjson

type GetExcessiveBlockCmd struct {
	BlockSize uint64 `json:"blockSize"`
}

type ExcessiveBlockSizeResult struct {
	ExcessiveBlockSize uint64 `json:"excessiveBlockSize"`
}

type SetExcessiveBlockCmd struct {
	BlockSize uint64 `json:"blockSize"`
}
