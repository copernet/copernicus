package script

import (
	"fmt"
)

const (
	ScriptErrOK int = iota
	ScriptErrUnknownError
	ScriptErrEvalFalse
	ScriptErrOpReturn

	/* Max sizes */

	ScriptErrScriptSize
	ScriptErrPushSize
	ScriptErrOpCount
	ScriptErrStackSize
	ScriptErrSigCount
	ScriptErrPubKeyCount

	/* Failed verify operations */

	ScriptErrVerify
	ScriptErrEqualVerify
	ScriptErrCheckMultiSigVerify
	ScriptErrCheckSigVerify
	ScriptErrNumEqualVerify

	/* Logical/Format/Canonical errors */

	ScriptErrBadOpCode
	ScriptErrDisabledOpCode
	ScriptErrInvalidStackOperation
	ScriptErrInvalidAltStackOperation
	ScriptErrUnbalancedConditional

	/* CheckLockTimeVerify and CheckSequenceVerify */

	ScriptErrNegativeLockTime
	ScriptErrUnsatisfiedLockTime

	/* Malleability */

	ScriptErrSigHashType
	ScriptErrSigDer
	ScriptErrMinimalData
	ScriptErrSigPushOnly
	ScriptErrSigHighs
	ScriptErrSigNullDummy
	ScriptErrPubKeyType
	ScriptErrCleanStack
	ScriptErrMinimalIf
	ScriptErrSigNullFail

	/* softFork safeness */

	ScriptErrDiscourageUpgradableNOPs
	ScriptErrDiscourageUpgradableWitnessProgram

	/* misc */
	ScriptErrNonCompressedPubKey

	/* anti replay */
	ScriptErrIllegalForkId
	ScriptErrMustUseForkId

	ScriptErrErrorCount
)

func ScriptErrorString(scriptError int) string {
	switch scriptError {
	case ScriptErrOK:
		return "No error"
	case ScriptErrEvalFalse:
		return "Script evaluated without error but finished with a false/empty top stack element"
	case ScriptErrVerify:
		return "Script failed an OP_VERIFY operation"
	case ScriptErrEqualVerify:
		return "Script failed an OP_EQUALVERIFY operation"
	case ScriptErrCheckMultiSigVerify:
		return "Script failed an OP_CHECKMULTISIGVERIFY operation"
	case ScriptErrCheckSigVerify:
		return "Script failed an OP_CHECKSIGVERIFY operation"
	case ScriptErrNumEqualVerify:
		return "Script failed an OP_NUMEQUALVERIFY operation"
	case ScriptErrScriptSize:
		return "Script is too big"
	case ScriptErrPushSize:
		return "Push value size limit exceeded"
	case ScriptErrOpCount:
		return "Operation limit exceeded"
	case ScriptErrStackSize:
		return "Stack size limit exceeded"
	case ScriptErrSigCount:
		return "Signature count negative or greater than pubKey count"
	case ScriptErrPubKeyCount:
		return "PubKey count negative or limit exceeded"
	case ScriptErrBadOpCode:
		return "OpCode missing or not understood"
	case ScriptErrDisabledOpCode:
		return "Attempted to use a disabled opCode"
	case ScriptErrInvalidStackOperation:
		return "Operation not valid with the current stack size"
	case ScriptErrInvalidAltStackOperation:
		return "Operation not valid with the current altStack size"
	case ScriptErrOpReturn:
		return "OP_RETURN was encountered"
	case ScriptErrUnbalancedConditional:
		return "Invalid OP_IF construction"
	case ScriptErrNegativeLockTime:
		return "Negative lockTime"
	case ScriptErrUnsatisfiedLockTime:
		return "LockTime requirement not satisfied"
	case ScriptErrSigHashType:
		return "Signature hash type missing or not understood"
	case ScriptErrSigDer:
		return "Non-canonical DER signature"
	case ScriptErrMinimalData:
		return "Data push larger than necessary"
	case ScriptErrSigPushOnly:
		return "Only non-push operators allowed in signatures"
	case ScriptErrSigHighs:
		return "Non-canonical signature: S value is unnecessarily high"
	case ScriptErrSigNullDummy:
		return "Dummy CheckMultiSig argument must be zero"
	case ScriptErrPubKeyType:
		return "Public key is neither compressed or uncompressed"
	case ScriptErrMinimalIf:
		return "OP_IF/NOTIF argument must be minimal"
	case ScriptErrSigNullFail:
		return "Signature must be zero for failed CHECK(MULTI)SIG operation"
	case ScriptErrDiscourageUpgradableNOPs:
		return "NOPx reserved for soft-fork upgrades"
	case ScriptErrDiscourageUpgradableWitnessProgram:
		return "Witness version reserved for soft-fork upgrades"
	default:
		break
	}
	return "unknown error"

}

type ErrDesc struct {
	Code int
	Desc string
}

func (e *ErrDesc) Error() string {
	return fmt.Sprintf("script error :%s code:%d", e.Desc, e.Code)
}

func ScriptErr(scriptError int) error {
	str := ScriptErrorString(scriptError)
	return &ErrDesc{
		Code: scriptError,
		Desc: str,
	}
}