package mining

const (
	/*MaxTxSigOpsCount allowed number of signature check operations per transaction. */
	MaxTxSigOpsCount uint64 = 20000
	/*OneMegaByte 1MB */
	OneMegaByte uint64 = 1000000

	/*DefaultMaxGeneratedBlockSize default for -blockMaxsize, which controls the maximum size of block the
	 * mining code will create **/
	DefaultMaxGeneratedBlockSize uint64 = 2 * OneMegaByte

	DefaultMaxBlockSize = 8 * OneMegaByte

	/*DefaultBlockPrioritySize default for -blockPrioritySize, maximum space for zero/low-fee transactions*/
	DefaultBlockPrioritySize uint64 = 0

	/*DefaultBlockMinTxFee default for -blockMinTxFee, which sets the minimum feeRate for a transaction
	 * in blocks created by mining code **/
	DefaultBlockMinTxFee uint = 1000

	/*MaxStandardTxSize the maximum size for transactions we're willing to relay/mine */
	MaxStandardTxSize uint = 100000

	/*MaxP2SHSigOps maximum number of signature check operations in an IsStandard() P2SH script*/
	MaxP2SHSigOps uint = 15

	/*MaxStandardTxSigOps the maximum number of sigops we're willing to relay/mine in a single tx */
	MaxStandardTxSigOps = uint(MaxTxSigOpsCount / 5)
)
