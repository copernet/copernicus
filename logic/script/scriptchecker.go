package script

import (
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util/amount"
)

type ScriptChecker interface {
	CheckLockTime(lockTime int64, txLockTime int64, sequence uint32) bool
	CheckSequence(sequence int64, txToSequence int64, txVersion uint32) bool
	CheckSig(transaction *tx.Tx, signature []byte, pubKey []byte, scriptCode *script.Script,
		nIn int, money amount.Amount, flags uint32) (bool, error)
}
