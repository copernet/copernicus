package policy

import (
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utxo"
)

const (
	/*MAX_TX_SIGOPS_COUNT allowed number of signature check operations per transaction. */
	MAX_TX_SIGOPS_COUNT uint64 = 20000
	/*ONE_MEGABYTE 1MB */
	ONE_MEGABYTE uint64 = 1000000

	/*DEFAULT_MAX_GENERATED_BLOCK_SIZE Default for -blockmaxsize, which controls the maximum size of block the
	 * mining code will create **/
	DEFAULT_MAX_GENERATED_BLOCK_SIZE uint64 = 2 * ONE_MEGABYTE

	DEFAULT_MAX_BLOCK_SIZE = 8 * ONE_MEGABYTE

	/*DEFAULT_BLOCK_PRIORITY_SIZE Default for -blockprioritysize, maximum space for zero/low-fee transactions*/
	DEFAULT_BLOCK_PRIORITY_SIZE uint64 = 0

	/*DEFAULT_BLOCK_MIN_TX_FEE Default for -blockmintxfee, which sets the minimum feerate for a transaction
	 * in blocks created by mining code **/
	DEFAULT_BLOCK_MIN_TX_FEE uint = 1000

	/*MAX_STANDARD_TX_SIZE The maximum size for transactions we're willing to relay/mine */
	MAX_STANDARD_TX_SIZE uint = 100000

	/*MAX_P2SH_SIGOPS Maximum number of signature check operations in an IsStandard() P2SH script*/
	MAX_P2SH_SIGOPS uint = 15

	/*MAX_STANDARD_TX_SIGOPS The maximum number of sigops we're willing to relay/mine in a single tx */
	MAX_STANDARD_TX_SIGOPS uint = uint(MAX_TX_SIGOPS_COUNT / 5)

	/*DEFAULT_MAX_MEMPOOL_SIZE Default for -maxmempool, maximum megabytes of mempool memory usage */
	DEFAULT_MAX_MEMPOOL_SIZE uint = 300

	/*MAX_STANDARD_P2WSH_STACK_ITEMS The maximum number of witness stack items in a standard P2WSH script */
	MAX_STANDARD_P2WSH_STACK_ITEMS uint = 100

	/*MAX_STANDARD_P2WSH_STACK_ITEM_SIZE The maximum size of each witness stack item in a standard P2WSH script */
	MAX_STANDARD_P2WSH_STACK_ITEM_SIZE uint = 80

	/*MAX_STANDARD_P2WSH_SCRIPT_SIZE The maximum size of a standard witnessScript */
	MAX_STANDARD_P2WSH_SCRIPT_SIZE uint = 3600

	/*STANDARD_SCRIPT_VERIFY_FLAGS Standard script verification flags that standard transactions will comply
	 * with. However scripts violating these flags may still be present in valid
	 * blocks and we must accept those blocks.
	 */
	STANDARD_SCRIPT_VERIFY_FLAGS uint = core.SCRIPT_VERIFY_P2SH | core.SCRIPT_VERIFY_DERSIG |
		core.SCRIPT_VERIFY_STRICTENC | core.SCRIPT_VERIFY_MINIMALDATA |
		core.SCRIPT_VERIFY_NULLDUMMY | core.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS |
		core.SCRIPT_VERIFY_CLEANSTACK | core.SCRIPT_VERIFY_NULLFAIL |
		core.SCRIPT_VERIFY_CHECKLOCKTIMEVERIFY | core.SCRIPT_VERIFY_CHECKSEQUENCEVERIFY |
		core.SCRIPT_VERIFY_LOW_S | core.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM

	/*STANDARD_NOT_MANDATORY_VERIFY_FLAGS For convenience, standard but not mandatory verify flags. */
	STANDARD_NOT_MANDATORY_VERIFY_FLAGS int = int(STANDARD_SCRIPT_VERIFY_FLAGS) & (^MANDATORY_SCRIPT_VERIFY_FLAGS)

	/*STANDARD_LOCKTIME_VERIFY_FLAGS Used as the flags parameter to sequence and nLocktime checks in
	 * non-consensus code. */
	STANDARD_LOCKTIME_VERIFY_FLAGS uint = consensus.LocktimeVerifySequence | consensus.LocktimeMedianTimePast

	// MANDATORY_SCRIPT_VERIFY_FLAGS Mandatory script verification flags that all new blocks must comply with for
	// them to be valid. (but old blocks may not comply with) Currently just P2SH,
	// but in the future other flags may be added, such as a soft-fork to enforce
	// strict DER encoding.
	//
	// Failing one of these tests may trigger a DoS ban - see CheckInputs() for
	// details.
	MANDATORY_SCRIPT_VERIFY_FLAGS = core.SCRIPT_VERIFY_P2SH | core.SCRIPT_VERIFY_STRICTENC | core.SCRIPT_ENABLE_SIGHASH_FORKID
)

/*IsStandardTx Check for standard transaction types
 * @return True if all outputs (scriptPubKeys) use only standard transaction
 * forms
 */
func IsStandardTx(tx *model.Tx, reason *string) bool {
	if tx.Version > model.MAX_STANDARD_VERSION || tx.Version < 1 {
		*reason = "Version"
		return false
	}

	// Extremely large transactions with lots of inputs can cost the network
	// almost as much to process as they cost the sender in fees, because
	// computing signature hashes is O(ninputs*txsize). Limiting transactions
	// to MAX_STANDARD_TX_SIZE mitigates CPU exhaustion attacks.
	if uint(tx.SerializeSize()) > MAX_STANDARD_TX_SIZE {
		*reason = "Tx-size"
		return false
	}

	for _, txIn := range tx.Ins {
		// Biggest 'standard' txin is a 15-of-15 P2SH multisig with compressed
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

		if whichType == model.TX_NULL_DATA {
			nDataOut++
		} else if whichType == model.TX_MULTISIG && !conf.GlobalValueInstance.GetIsBareMultisigStd() {
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
func IsStandard(scriptPubKey *model.Script, whichType *int) bool {
	vSolutions := algorithm.NewVector()
	if !model.Solver(scriptPubKey, whichType, vSolutions) {
		return false
	}

	if *whichType == model.TX_MULTISIG {
		m := vSolutions.Array[0].(algorithm.Vector).Array[0].(byte)
		n := vSolutions.Array[0].(algorithm.Vector).Array[0].(byte)
		// Support up to x-of-3 multisig txns as standard
		if n < 1 || n > 3 {
			return false
		}
		if m < 1 || m > n {
			return false
		}
	} else if *whichType == model.TX_NULL_DATA &&
		(!conf.GlobalValueInstance.GetAcceptDatacarrier() ||
			uint(scriptPubKey.Size()) > conf.GlobalValueInstance.GetMaxDatacarrierBytes()) {
		return false
	}
	return *whichType != model.TX_NONSTANDARD
}

// AreInputsStandard Check for standard transaction types
// cache Map of previous transactions that have outputs we're spending
func AreInputsStandard(tx *model.Tx, cache *utxo.CoinsViewCache) bool {
	if tx.IsCoinBase() {
		// Coinbases don't use vin normally
		return true
	}

	for _, vin := range tx.Ins {
		prev := cache.GetOutputFor(vin)

		solutions := algorithm.NewVector()
		var whichType int
		// get the scriptPubKey corresponding to this input:
		prevScript := prev.Script
		if !model.Solver(prevScript, &whichType, solutions) {
			return false
		}

		if whichType == model.TX_SCRIPTHASH {
			stack := algorithm.NewVector()
			//i := model.NewInterpreter()

			//convert the scriptSig into a stack, so we can inspect the
			//redeemScript

			//if i.Verify(tx, len(tx.Ins), vin.Script, core.SCRIPT_VERIFY_NONE) {		// todo complete
			//	return false
			//}

			if stack.Size() == 0 {
				return false
			}

			b := make([]byte, 0)
			for _, item := range stack.Array {
				b = append(b, item.(byte))
			}
			subscript := model.NewScriptRaw(b)
			count, _ := subscript.GetSigOpCount()
			if uint(count) > MAX_P2SH_SIGOPS {
				return false
			}

		}
	}
	return true
}
