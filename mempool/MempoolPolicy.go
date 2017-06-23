package mempool

import "btcutil"

type MempoolPolicy struct {
	DisableRelayPriority bool
	RelayNonStandard     bool
	FreeTxRelayLimit     float64
	MaxOrphanTxs         int
	MaxOrphanTxSize      int
	MaxSigOpsPerTx       int
	MinRealyTxFee        btcutil.Amount
}
