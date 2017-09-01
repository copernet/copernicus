package mempool

import "github.com/btcboost/copernicus/model"

type TxMempoolEntry struct {
	TxRef         *model.Tx
	Fee           int64
	TxSize        int
	UsageSize     int
	LocalTime     int64
	EntryPriority float64
	EntryHeight   int
	//!< Sum of all txin values that are already in blockchain
	InChainInputValue int64
	SpendsCoinbase    bool
	SigOpCount        int64
	FeeDelta          int64
}
