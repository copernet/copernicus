package btcjson

type ExcessiveBlockSizeResult struct {
	ExcessiveBlockSize uint64 `json:"excessiveBlockSize"`
}

type SetExcessiveBlockCmd struct {
	BlockSize uint64 `json:"blockSize"`
}
