package policy

import (
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utxo"
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
