package lscript

import (
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util/amount"
)

type RealChecker struct {
}

func (src *RealChecker) CheckSig(transaction *tx.Tx, signature []byte, pubKey []byte, scriptCode *script.Script,
	nIn int, money amount.Amount, flags uint32) (bool, error) {
	if len(signature) == 0 || len(pubKey) == 0 {
		return false, nil
	}
	hashType := signature[len(signature)-1]
	txSigHash, err := tx.SignatureHash(transaction, scriptCode, uint32(hashType), nIn, money, flags)
	if err != nil {
		return false, err
	}
	signature = signature[:len(signature)-1]
	fOk := tx.CheckSig(txSigHash, signature, pubKey)
	// log.Debug("CheckSig: txid: %s, txSigHash: %s, signature: %s, pubkey: %s, result: %v", transaction.GetHash(),
	// 	txSigHash, hex.EncodeToString(signature), hex.EncodeToString(pubKey), fOk)

	//if !fOk {
	//	panic("CheckSig failed")
	//}
	return fOk, err
}

func (src *RealChecker) CheckLockTime(lockTime int64, txLockTime int64, sequence uint32) bool {
	// There are two kinds of nLockTime: lock-by-blockheight and
	// lock-by-blocktime, distinguished by whether nLockTime <
	// LOCKTIME_THRESHOLD.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nLockTime being tested is the same as the nLockTime in the
	// transaction.
	if !((txLockTime < script.LockTimeThreshold && lockTime < script.LockTimeThreshold) ||
		(txLockTime >= script.LockTimeThreshold && lockTime >= script.LockTimeThreshold)) {
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
	if script.SequenceFinal == sequence {
		return false
	}
	return true
}

func (src *RealChecker) CheckSequence(sequence int64, txToSequence int64, txVersion uint32) bool {
	// Fail if the transaction's version number is not set high enough to
	// trigger BIP 68 rules.
	if txVersion < 2 {
		return false
	}
	// Sequence numbers with their most significant bit set are not consensus
	// constrained. Testing that the transaction's sequence number do not have
	// this bit set prevents using this property to get around a
	// checkSequenceVerify check.
	if txToSequence&script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
		return false
	}
	// Mask off any bits that do not have consensus-enforced meaning before
	// doing the integer comparisons
	nLockTimeMask := script.SequenceLockTimeTypeFlag | script.SequenceLockTimeMask
	txToSequenceMasked := txToSequence & int64(nLockTimeMask)
	nSequenceMasked := sequence & int64(nLockTimeMask)

	// There are two kinds of nSequence: lock-by-blockHeight and
	// lock-by-blockTime, distinguished by whether nSequenceMasked <
	// CTxIn::SEQUENCE_LOCKTIME_TYPE_FLAG.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nSequenceMasked being tested is the same as the nSequenceMasked in the
	// transaction.
	if !((txToSequenceMasked < script.SequenceLockTimeTypeFlag && nSequenceMasked < script.SequenceLockTimeTypeFlag) ||
		(txToSequenceMasked >= script.SequenceLockTimeTypeFlag && nSequenceMasked >= script.SequenceLockTimeTypeFlag)) {
		return false
	}
	if nSequenceMasked > txToSequenceMasked {
		return false
	}
	return true
}

func NewScriptRealChecker() *RealChecker {
	return &RealChecker{}
}
