package core

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

const (
	/*DefaultMaxGeneratedBlockSize default for -blockMaxsize, which controls the maximum size of block the
	 * mining code will create **/
	DefaultMaxGeneratedBlockSize uint64 = 2 * OneMegaByte
	/** Default for -blockprioritypercentage, define the amount of block space
	 * reserved to high priority transactions **/

	DefaultBlockPriorityPercentage uint64= 5

	/*DefaultBlockMinTxFee default for -blockMinTxFee, which sets the minimum feeRate for a transaction
	 * in blocks created by mining code **/
	DefaultBlockMinTxFee uint = 1000

	MaxStandardVersion = 2

	/*MaxStandardTxSize the maximum size for transactions we're willing to relay/mine */
	MaxStandardTxSize uint = 100000

	/*MaxP2SHSigOps maximum number of signature check operations in an IsStandard() P2SH script*/
	MaxP2SHSigOps uint = 15

	/*MaxStandardTxSigOps the maximum number of sigops we're willing to relay/mine in a single tx */
	MaxStandardTxSigOps = uint(MaxTxSigOpsCount / 5)

	/*DefaultMaxMemPoolSize default for -maxMemPool, maximum megabytes of memPool memory usage */
	DefaultMaxMemPoolSize uint = 300

	/** Default for -incrementalrelayfee, which sets the minimum feerate increase
 	* for mempool limiting or BIP 125 replacement **/
	DefaultIncrementalRelayFee int64 = 1000

	/** Default for -bytespersigop */
	DefaultBytesPerSigop uint= 20

	/** The maximum number of witness stack items in a standard P2WSH script */
	MaxStandardP2WSHStackItems uint = 100

	/*MaxStandardP2WSHStackItemSize the maximum size of each witness stack item in a standard P2WSH script */
	MaxStandardP2WSHStackItemSize uint = 80

	/*MaxStandardP2WSHScriptSize the maximum size of a standard witnessScript */
	MaxStandardP2WSHScriptSize uint = 3600


	// MandatoryScriptVerifyFlags mandatory script verification flags that all new blocks must comply with for
	// them to be valid. (but old blocks may not comply with) Currently just P2SH,
	// but in the future other flags may be added, such as a soft-fork to enforce
	// strict DER encoding.
	//
	// Failing one of these tests may trigger a DoS ban - see CheckInputs() for
	// details.
	MANDATORY_SCRIPT_VERIFY_FLAGS uint =
		SCRIPT_VERIFY_P2SH | SCRIPT_VERIFY_STRICTENC |
			SCRIPT_ENABLE_SIGHASH_FORKID | SCRIPT_VERIFY_LOW_S | SCRIPT_VERIFY_NULLFAIL

	/*StandardScriptVerifyFlags standard script verification flags that standard transactions will comply
	 * with. However scripts violating these flags may still be present in valid
	 * blocks and we must accept those blocks.
	 */
	StandardScriptVerifyFlags uint = MANDATORY_SCRIPT_VERIFY_FLAGS | SCRIPT_VERIFY_DERSIG |
		SCRIPT_VERIFY_MINIMALDATA | SCRIPT_VERIFY_NULLDUMMY |
		SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS | SCRIPT_VERIFY_CLEANSTACK |
		SCRIPT_VERIFY_NULLFAIL | SCRIPT_VERIFY_CHECKLOCKTIMEVERIFY |
		SCRIPT_VERIFY_CHECKSEQUENCEVERIFY | SCRIPT_VERIFY_LOW_S |
		SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM

	/*StandardNotMandatoryVerifyFlags for convenience, standard but not mandatory verify flags. */
	StandardNotMandatoryVerifyFlags uint= StandardScriptVerifyFlags & (^MANDATORY_SCRIPT_VERIFY_FLAGS)

	/*StandardLockTimeVerifyFlags used as the flags parameter to sequence and LockTime checks in
	 * non-core code. */
	StandardLockTimeVerifyFlags uint = LocktimeVerifySequence | LocktimeMedianTimePast
)

//GetMaxBlockSigOpsCount Compute the maximum number of sigops operation that can contained in a block
//given the block size as parameter. It is computed by multiplying
//MAX_BLOCK_SIGOPS_PER_MB by the size of the block in MB rounded up to the
//closest integer.
func GetMaxBlockSigOpsCount(blockSize uint64) uint64 {
	roundedUp := 1 + ((blockSize - 1) / OneMegaByte)
	return roundedUp * MaxBlockSigopsPerMb
}
