package script

import (
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util/amount"
)

type ScriptEmptyChecker struct {
}

func (sec *ScriptEmptyChecker) CheckSig(transaction *tx.Tx, signature []byte, pubKey []byte, scriptCode *script.Script,
	nIn int, money amount.Amount, flags uint32) (bool, error) {
	return false, errcode.New(errcode.ScriptErrInvalidOpCode)
}

func (src *ScriptEmptyChecker) CheckLockTime(lockTime int64, txLockTime int64, sequence uint32) bool {
	return false
}

func (src *ScriptEmptyChecker) CheckSequence(sequence int64, txToSequence int64, txVersion uint32) bool {
	return false
}

func NewScriptEmptyChecker() *ScriptEmptyChecker {
	return &ScriptEmptyChecker{}
}
