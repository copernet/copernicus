package model

import (
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
)

func VerfySinature(vchSig []byte, pubkey *core.PublicKey, sigHash *utils.Hash) (bool, error) {
	sign, err := core.ParseDERSignature(vchSig)
	if err != nil {
		return false, err
	}
	result := sign.Verify(sigHash.GetCloneBytes(), pubkey)
	return result, nil
}

func CheckSig(signHash *utils.Hash, vchSigIn []byte, vchPubKey []byte) (bool, error) {
	if len(vchPubKey) == 0 {
		return false, errors.New("public key is nil")
	}
	if len(vchSigIn) == 0 {
		return false, errors.New("signature is nil")
	}
	publicKey, err := core.ParsePubKey(vchPubKey)
	if err != nil {
		return false, err
	}
	ret, err := VerfySinature(vchSigIn, publicKey, signHash)
	if err != nil {
		return false, err
	}
	if !ret {
		return false, errors.New("VerfySinature is failed")
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
	if (txLockTime < LOCKTIME_THRESHOLD && lockTime < LOCKTIME_THRESHOLD) ||
		(txLockTime >= LOCKTIME_THRESHOLD && lockTime >= LOCKTIME_THRESHOLD) {
		return false
	}

	// Now that we know we're comparing apples-to-apples, the comparison is a
	// simple numeric one.
	if lockTime > txLockTime {
		return false
	}
	// Finally the nLockTime feature can be disabled and thus
	// CHECKLOCKTIMEVERIFY bypassed if every txin has been finalized by setting
	// nSequence to maxint. The transaction would be allowed into the
	// blockchain, making the opcode ineffective.
	//
	// Testing if this vin is not final is sufficient to prevent this condition.
	// Alternatively we could test all inputs, but testing just this input
	// minimizes the data required to prove correct CHECKLOCKTIMEVERIFY
	// execution.
	if SEQUENCE_FINAL == sequence {
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
	// CHECKSEQUENCEVERIFY check.
	if txToSequence&SEQUENCE_LOCKTIME_DISABLE_FLAG == 1 {
		return false
	}
	// Mask off any bits that do not have consensus-enforced meaning before
	// doing the integer comparisons
	nLockTimeMask := SEQUENCE_LOCKTIME_TYPE_FLAG | SEQUENCE_LOCKTIME_MASK
	txToSequenceMasked := txToSequence & int64(nLockTimeMask)
	nSequenceMasked := sequence & int64(nLockTimeMask)

	// There are two kinds of nSequence: lock-by-blockheight and
	// lock-by-blocktime, distinguished by whether nSequenceMasked <
	// CTxIn::SEQUENCE_LOCKTIME_TYPE_FLAG.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nSequenceMasked being tested is the same as the nSequenceMasked in the
	// transaction.
	if !(txToSequenceMasked < SEQUENCE_LOCKTIME_TYPE_FLAG && nSequenceMasked < SEQUENCE_LOCKTIME_TYPE_FLAG) ||
		(txToSequenceMasked >= SEQUENCE_LOCKTIME_TYPE_FLAG && nSequenceMasked >= SEQUENCE_LOCKTIME_TYPE_FLAG) {
		return false
	}
	if nSequenceMasked > txToSequenceMasked {
		return false
	}
	return true
}

//func VerifyScript()  {
//
//}
