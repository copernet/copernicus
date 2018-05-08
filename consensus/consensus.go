package consensus

const (
	// OneMegabyte  1MB
	OneMegabyte = 1000000
	// MaxTxSize  The maximum allowed size for a transaction, in bytes
	MaxTxSize = OneMegabyte
	// LegacyMaxBlockSize  The maximum allowed size for a block, before the UAHF
	LegacyMaxBlockSize = OneMegabyte
	// DefaultMaxBlockSize  Default setting for maximum allowed size for a block, in bytes
	DefaultMaxBlockSize = 8 * OneMegabyte
	// MaxBlockSigopsPerMb  The maximum allowed number of signature check operations per MB
	// in a block (network rule)
	MaxBlockSigopsPerMb = 20000
	// CoinbaseMaturity  Coinbase transaction outputs can only be spent after this number of new
	// blocks (network rule)
	CoinbaseMaturity = 100
	// Interpret sequence numbers as relative lock-time constraints.
)

const (
	// LocktimeVerifySequence ,  Interpret sequence numbers as relative lock-time constraints.
	LocktimeVerifySequence = 1 << iota

	// LocktimeMedianTimePast , Use GetMedianTimePast() instead of nTime for end point timestamp.
	LocktimeMedianTimePast

	// StandardLocktimeVerifyFlags used as the flags parameter to sequence and nLocktime checks in
	// non-consensus code.
	StandardLocktimeVerifyFlags = LocktimeVerifySequence | LocktimeMedianTimePast
)

const MempoolHeight = 0x7FFFFFFF

const (
	//Expiration time for orphan transactions in seconds */
	ORPHAN_TX_EXPIRE_TIME = 20 * 60
	ORPHAN_TX_EXPIRE_INTERVAL = 5 * 60

)

// GetMaxBlockSigOpsCount Compute the maximum number of sigops operation that can contained in a block
// given the block size as parameter. It is computed by multiplying
// MAX_BLOCK_SIGOPS_PER_MB by the size of the block in MB rounded up to the
// closest integer.
func GetMaxBlockSigOpsCount(blockSize uint64) uint64 {
	roundedUp := 1 + ((blockSize - 1) / OneMegabyte)
	return roundedUp * MaxBlockSigopsPerMb
}
