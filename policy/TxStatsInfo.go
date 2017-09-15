package policy

type TxStatsInfo struct {
	BlockHeight int
	BucketIndex int
}

func NewTxStatsInfo() *TxStatsInfo {
	txStatsInfo := TxStatsInfo{}
	return &txStatsInfo
}
