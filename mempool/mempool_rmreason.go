package mempool

/**
 * Reason why a transaction was removed from the mempool, this is passed to the
 * notification signal.
 */

const (
	UNKNOWN int = iota
	EXPIRY
	SIZELIMIT
	REORG
	BLOCK
	CONFLICT
	REPLACED
)
