package lscript

import (
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

type EmptyChecker struct {
}

func (sec *EmptyChecker) CheckSig(transaction *tx.Tx, signature []byte, pubKey []byte, scriptCode *script.Script,
	nIn int, money amount.Amount, flags uint32) (bool, error) {
	return false, errcode.New(errcode.ScriptErrInvalidOpCode)
}

func (sec *EmptyChecker) CheckLockTime(lockTime int64, txLockTime int64, sequence uint32) bool {
	return false
}

func (sec *EmptyChecker) CheckSequence(sequence int64, txToSequence int64, txVersion uint32) bool {

	return false
}

func (sec *EmptyChecker) VerifySignature(vchSig []byte, pubKey *crypto.PublicKey, sigHash *util.Hash) (bool, error) {
	return false, nil
}

func NewScriptEmptyChecker() *EmptyChecker {
	return &EmptyChecker{}
}
