package policy

import (
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utxo"
)

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

/*IsStandardTx check for standard transaction types
 * @return True if all outputs (scriptPubKeys) use only standard transaction
 * forms
 */
func IsStandardTx(tx *core.Tx, reason *string) bool {
	if tx.Version > core.MaxStandardVersion || tx.Version < 1 {
		*reason = "Version"
		return false
	}

	// Extremely large transactions with lots of inputs can cost the network
	// almost as much to process as they cost the sender in fees, because
	// computing signature hashes is O(ninputs*txsize). Limiting transactions
	// to MAX_STANDARD_TX_SIZE mitigates CPU exhaustion attacks.
	if uint(tx.SerializeSize()) > MaxStandardTxSize {
		*reason = "Tx-size"
		return false
	}

	for _, txIn := range tx.Ins {
		// Biggest 'standard' txin is a 15-of-15 P2SH multiSig with compressed
		// keys (remember the 520 byte limit on redeemScript size). That works
		// out to a (15*(33+1))+3=513 byte redeemScript, 513+1+15*(73+1)+3=1627
		// bytes of scriptSig, which we round off to 1650 bytes for some minor
		// future-proofing. That's also enough to spend a 20-of-20 CHECKMULTISIG
		// scriptPubKey, though such a scriptPubKey is not considered standard.
		if txIn.Script.Size() > 1650 {
			*reason = "scriptsig-size"
			return false
		}
		if !txIn.Script.IsPushOnly() {
			*reason = "scriptsig-not-pushonly"
			return false
		}
	}

	nDataOut := uint(0)
	whichType := 0
	for _, txOut := range tx.Outs {
		if !IsStandard(txOut.Script, &whichType) {
			*reason = "scriptpubkey"
			return false
		}

		if whichType == core.TxNullData {
			nDataOut++
		} else if whichType == core.TxMultiSig && !conf.GlobalValueInstance.GetIsBareMultiSigStd() {
			*reason = "bare-multisig"
			return false
		} else if txOut.IsDust(conf.GlobalValueInstance.GetDustRelayFee()) {
			*reason = "dust"
			return false
		}
	}

	if nDataOut > 1 {
		*reason = "multi-op-return"
		return false
	}
	return true
}

/*IsStandard Check transaction inputs to mitigate two potential denial-of-service attacks:
 *
 * 1. scriptSigs with extra data stuffed into them, not consumed by scriptPubKey
 * (or P2SH script)
 * 2. P2SH scripts with a crazy number of expensive CHECKSIG/CHECKMULTISIG
 * operations
 *
 * Why bother? To avoid denial-of-service attacks; an attacker can submit a
 * standard HASH... OP_EQUAL transaction, which will get accepted into blocks.
 * The redemption script can be anything; an attacker could use a very
 * expensive-to-check-upon-redemption script like:
 *   DUP CHECKSIG DROP ... repeated 100 times... OP_1
 */
func IsStandard(scriptPubKey *core.Script, whichType *int) bool {
	vSolutions := container.NewVector()
	if !core.Solver(scriptPubKey, whichType, vSolutions) {
		return false
	}

	if *whichType == core.TxMultiSig {
		m := vSolutions.Array[0].(container.Vector).Array[0].(byte)
		n := vSolutions.Array[0].(container.Vector).Array[0].(byte)
		// Support up to x-of-3 multisig txns as standard
		if n < 1 || n > 3 {
			return false
		}
		if m < 1 || m > n {
			return false
		}
	} else if *whichType == core.TxNullData &&
		(!conf.GlobalValueInstance.GetAcceptDataCarrier() ||
			uint(scriptPubKey.Size()) > conf.GlobalValueInstance.GetMaxDataCarrierBytes()) {
		return false
	}
	return *whichType != core.TxNonStandard
}

// AreInputsStandard Check for standard transaction types
// cache Map of previous transactions that have outputs we're spending
func AreInputsStandard(tx *core.Tx, cache *utxo.CoinsViewCache) bool {
	if tx.IsCoinBase() {
		// CoinBases don't use vin normally
		return true
	}

	for index, vin := range tx.Ins {
		prev := cache.GetOutputFor(vin)

		solutions := container.NewVector()
		var whichType int
		// get the scriptPubKey corresponding to this input:
		prevScript := prev.Script
		if !core.Solver(prevScript, &whichType, solutions) {
			return false
		}

		if whichType == core.TxScriptHash {
			interpreter := core.NewInterpreter()

			//convert the scriptSig into a stack, so we can inspect the
			//redeemScript
			coin := utxo.NewEmptyCoin()
			exist := cache.GetCoin(vin.PreviousOutPoint, coin)
			if exist {
				ret, err := interpreter.Verify(tx, index, vin.Script, coin.TxOut.Script, crypto.ScriptVerifyNone)
				if err != nil || !ret {
					return false
				}
			}

			if interpreter.IsEmpty() {
				return false
			}

			b := make([]byte, 0)
			list := interpreter.List()
			for _, item := range list {
				byteFromStack, ok := item.([]byte)
				if !ok {
					panic("not []byte")
				}
				b = append(b, byteFromStack...)
			}
			subscript := core.NewScriptRaw(b)
			count, _ := subscript.GetSigOpCountWithAccurate(true)
			if uint(count) > MaxP2SHSigOps {
				return false
			}
		}
	}
	return true
}
