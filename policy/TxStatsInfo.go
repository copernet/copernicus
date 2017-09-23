package policy

type TxStatsInfo struct {
	BlockHeight uint
	BucketIndex uint
}

func NewTxStatsInfo() *TxStatsInfo {
	txStatsInfo := TxStatsInfo{}
	return &txStatsInfo
}
