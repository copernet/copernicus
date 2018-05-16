package tx
/*
import (
	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
)

func VerifySignature(vchSig []byte, pubkey *crypto.PublicKey, sigHash utils.Hash) (bool, error) {
	sign, err := crypto.ParseDERSignature(vchSig)
	if err != nil {
		return false, err
	}
	result := sign.Verify(sigHash.GetCloneBytes(), pubkey)
	return result, nil
}

func CheckSig(signHash utils.Hash, vchSigIn []byte, vchPubKey []byte) (bool, error) {
	if len(vchPubKey) == 0 {
		return false, errors.New("public key is nil")
	}
	if len(vchSigIn) == 0 {
		return false, errors.New("signature is nil")
	}
	publicKey, err := crypto.ParsePubKey(vchPubKey)
	if err != nil {
		return false, err
	}

	ret, err := VerifySignature(vchSigIn, publicKey, signHash)
	if err != nil {
		return false, err
	}
	if !ret {
		return false, errors.New("VerifySignature is failed")
	}
	return true, nil

}

func GetHashType(vchSig []byte) byte {
	if len(vchSig) == 0 {
		return 0
	}
	return vchSig[len(vchSig)-1]
}

func CheckLockTime(lockTime int64, txLockTime int64, sequence uint32) bool {
	// There are two kinds of nLockTime: lock-by-blockheight and
	// lock-by-blocktime, distinguished by whether nLockTime <
	// LOCKTIME_THRESHOLD.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nLockTime being tested is the same as the nLockTime in the
	// transaction.
	if (txLockTime < LockTimeThreshold && lockTime < LockTimeThreshold) ||
		(txLockTime >= LockTimeThreshold && lockTime >= LockTimeThreshold) {
		return false
	}

	// Now that we know we're comparing apples-to-apples, the comparison is a
	// simple numeric one.
	if lockTime > txLockTime {
		return false
	}
	// Finally the nLockTime feature can be disabled and thus
	// checkLockTimeVerify bypassed if every txIN has been finalized by setting
	// nSequence to maxInt. The transaction would be allowed into the
	// blockChain, making the opCode ineffective.
	//
	// Testing if this vin is not final is sufficient to prevent this condition.
	// Alternatively we could test all inputs, but testing just this input
	// minimizes the data required to prove correct checkLockTimeVerify
	// execution.
	if SequenceFinal == sequence {
		return false
	}
	return true
}

func CheckSequence(sequence int64, txToSequence int64, version int32) bool {
	// Fail if the transaction's version number is not set high enough to
	// trigger BIP 68 rules.
	if version < 2 {
		return false
	}
	// Sequence numbers with their most significant bit set are not consensus
	// constrained. Testing that the transaction's sequence number do not have
	// this bit set prevents using this property to get around a
	// checkSequenceVerify check.
	if txToSequence&SequenceLockTimeDisableFlag == SequenceLockTimeDisableFlag {
		return false
	}
	// Mask off any bits that do not have consensus-enforced meaning before
	// doing the integer comparisons
	nLockTimeMask := SequenceLockTimeTypeFlag | SequenceLockTimeMask
	txToSequenceMasked := txToSequence & int64(nLockTimeMask)
	nSequenceMasked := sequence & int64(nLockTimeMask)

	// There are two kinds of nSequence: lock-by-blockHeight and
	// lock-by-blockTime, distinguished by whether nSequenceMasked <
	// CTxIn::SEQUENCE_LOCKTIME_TYPE_FLAG.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nSequenceMasked being tested is the same as the nSequenceMasked in the
	// transaction.
	if !((txToSequenceMasked < SequenceLockTimeTypeFlag && nSequenceMasked < SequenceLockTimeTypeFlag) ||
		(txToSequenceMasked >= SequenceLockTimeTypeFlag && nSequenceMasked >= SequenceLockTimeTypeFlag)) {
		return false
	}
	if nSequenceMasked > txToSequenceMasked {
		return false
	}
	return true
}

func VerifyScript(tx *Tx, index int, scriptSig *Script, scriptPubKey *Script, flags uint32, err *crypto.ScriptError) bool {

	SetError(err, crypto.ScriptErrUnknownError)

	// If forkID is enabled, we also ensure strict encoding.
	if flags&crypto.ScriptEnableSigHashForkID != 0 {
		flags |= crypto.ScriptVerifyStrictenc
	}

	if flags&crypto.ScriptVerifySigPushOnly != 0 && !scriptSig.IsPushOnly() {
		return SetError(err, crypto.ScriptErrSigPushOnly)
	}

	ip := NewInterpreter()
	var copyIP *Interpreter
	ret, e := ip.Verify(tx, index, scriptSig, scriptPubKey, flags) // todo confirm
	if e != nil || !ret {
		return false
	}

	if flags&crypto.ScriptVerifyP2SH != 0 {
		copyIP.stack = ip.stack.Copy()
	}
	ret, e = ip.Verify(tx, index, scriptSig, scriptPubKey, flags) // todo confirm
	if e != nil || !ret {
		return false
	}
	if ip.IsEmpty() {
		return SetError(err, crypto.ScriptErrEvalFalse)
	}

	stackTop, ok := ip.stack.Last().([]byte)
	if ok {
		if !CastToBool(stackTop) {
			return SetError(err, crypto.ScriptErrEvalFalse)
		}
	} else {
		panic("error") // todo confirm: false or panic
	}

	// Additional validation for spend-to-script-hash transactions:
	if flags&crypto.ScriptVerifyP2SH != 0 && scriptPubKey.IsPayToScriptHash() {
		// scriptSig must be literals-only or validation fails
		if !scriptSig.IsPushOnly() {
			return SetError(err, crypto.ScriptErrSigPushOnly)
		}

		// Restore stack
		if copyIP != nil {
			container.SwapStack(ip.stack, copyIP.stack)
		}
		// stack cannot be empty here, because if it was the P2SH  HASH <> EQUAL
		// scriptPubKey would be evaluated with an empty stack and the
		// EvalScript above would return false.
		if ip.IsEmpty() {
			panic("stack should not be empty")
		}
		pubKeySerialized := ip.stack.Last()
		var pubKey2 *Script
		cov, ok := pubKeySerialized.([]byte)
		if ok {
			pubKey2 = NewScriptRaw(cov)
		} else {
			panic("error") // todo confirm: false or panic
		}

		ip.stack.PopStack()

		ret, e := ip.Verify(tx, index, scriptSig, pubKey2, flags) // todo confirm
		if e != nil || !ret {
			return false
		}

		if ip.IsEmpty() {
			return SetError(err, crypto.ScriptErrEvalFalse)
		}

		tmp := ip.stack.Last()
		cov2, ok2 := tmp.([]byte)
		if ok2 {
			if !CastToBool(cov2) {
				return SetError(err, crypto.ScriptErrEvalFalse)
			}
		} else {
			panic("error") // todo confirm: false or panic
		}
	}
	// The cleanStack check is only performed after potential P2SH evaluation,
	// as the non-P2SH evaluation of a P2SH script will obviously not result in
	// a clean stack (the P2SH inputs remain). The same holds for witness
	// evaluation.
	if flags&crypto.ScriptVerifyCleanStack != 0 {
		// Disallow cleanStack without P2SH, as otherwise a switch
		// cleanStack->P2SH+cleanStack would be possible, which is not a
		// softFork (and P2SH should be one).
		if flags&crypto.ScriptVerifyP2SH == 0 {
			panic("error")
		}
		if ip.stack.Size() != 1 {
			return SetError(err, crypto.ScriptErrCleanStack)
		}
	}

	return SetSuccess(err)
}

func SetError(ret *crypto.ScriptError, seterror crypto.ScriptError) bool {
	if ret != nil {
		*ret = seterror
	}
	return false
}

func SetSuccess(ret *crypto.ScriptError) bool {
	if ret != nil {
		*ret = crypto.ScriptErrOK
	}
	return true
}
*/