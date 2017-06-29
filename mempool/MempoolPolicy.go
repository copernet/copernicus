package mempool

type PoolPolicy struct {
	DisableRelayPriority bool
	RelayNonStandard     bool
	FreeTxRelayLimit     float64
	MaxOrphanTxs         int
	MaxOrphanTxSize      int
	MaxSigOpsPerTx       int
	//MinRealyTxFee        btcutil.Amount
}
