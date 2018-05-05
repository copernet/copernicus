package consensus

const (
	//OneMegabyte  1MB
	OneMegaByte = 1000000

	//MaxTxSize  The maximum allowed size for a transaction, in bytes
	MaxTxSize = OneMegaByte

	//LegacyMaxBlockSize  The maximum allowed size for a block, before the UAHF
	LegacyMaxBlockSize = OneMegaByte

	//DefaultMaxBlockSize  Default setting for maximum allowed size for a block, in bytes
	DefaultMaxBlockSize = 32 * OneMegaByte

	/*MaxBlockSigopsPerMb  The maximum allowed number of signature check operations per MB in a block
	* (network rule) */
	MaxBlockSigopsPerMb = 20000

	/*MaxTxSigOpsCount allowed number of signature check operations per transaction. */
	MaxTxSigOpsCount = 20000

	/** Coinbase transaction outputs can only be spent after this number of new
	 * blocks (network rule) */
	CoinbaseMaturity = 100
)

const (
	// LocktimeVerifySequence ,  Interpret sequence numbers as relative lock-time constraints.
	LocktimeVerifySequence = 1 << iota

	// LocktimeMedianTimePast , Use GetMedianTimePast() instead of nTime for end point timestamp.
	LocktimeMedianTimePast
)

// GetMaxBlockSigOpsCount Compute the maximum number of sigops operation that can contained in a block
// given the block size as parameter. It is computed by multiplying
// MAX_BLOCK_SIGOPS_PER_MB by the size of the block in MB rounded up to the
// closest integer.

func GetMaxBlockSigOpsCount(blockSize uint64) uint64 {
	roundedUp := 1 + ((blockSize - 1) / OneMegaByte)
	return roundedUp * MaxBlockSigopsPerMb
}
