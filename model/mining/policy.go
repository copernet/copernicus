package mining

import "github.com/btcboost/copernicus/model/consensus"

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

	/*DefaultMaxMemPoolSize default for -maxMemPool, maximum megabytes of memPool memory usage */
	DefaultMaxMemPoolSize uint = 300

	/*MaxStandardP2WSHStackItems the maximum number of witness stack items in a standard P2WSH script */
	MaxStandardP2WSHStackItems uint = 100

	/*MaxStandardP2WSHStackItemSize the maximum size of each witness stack item in a standard P2WSH script */
	MaxStandardP2WSHStackItemSize uint = 80

	/*MaxStandardP2WSHScriptSize the maximum size of a standard witnessScript */
	MaxStandardP2WSHScriptSize uint = 3600

	/*StandardScriptVerifyFlags standard script verification flags that standard transactions will comply
	 * with. However scripts violating these flags may still be present in valid
	 * blocks and we must accept those blocks.
	 */
	StandardScriptVerifyFlags uint = crypto.ScriptVerifyP2SH | crypto.ScriptVerifyDersig |
		crypto.ScriptVerifyStrictenc | crypto.ScriptVerifyMinimalData |
		crypto.ScriptVerifyNullDummy | crypto.ScriptVerifyDiscourageUpgradableNOPs |
		crypto.ScriptVerifyCleanStack | crypto.ScriptVerifyNullFail |
		crypto.ScriptVerifyCheckLockTimeVerify | crypto.ScriptVerifyCheckSequenceVerify |
		crypto.ScriptVerifyLows | crypto.ScriptVerifyDiscourageUpgradAbleWitnessProgram

	/*StandardNotMandatoryVerifyFlags for convenience, standard but not mandatory verify flags. */
	StandardNotMandatoryVerifyFlags = int(StandardScriptVerifyFlags) & (^MandatoryScriptVerifyFlags)

	/*StandardLockTimeVerifyFlags used as the flags parameter to sequence and LockTime checks in
	 * non-core code. */
	StandardLockTimeVerifyFlags uint = consensus.LocktimeVerifySequence | consensus.LocktimeMedianTimePast

	// MandatoryScriptVerifyFlags mandatory script verification flags that all new blocks must comply with for
	// them to be valid. (but old blocks may not comply with) Currently just P2SH,
	// but in the future other flags may be added, such as a soft-fork to enforce
	// strict DER encoding.
	//
	// Failing one of these tests may trigger a DoS ban - see CheckInputs() for
	// details.
	MandatoryScriptVerifyFlags = crypto.ScriptVerifyP2SH | crypto.ScriptVerifyStrictenc | crypto.ScriptEnableSigHashForkID
)
