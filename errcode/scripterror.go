package errcode

type ScriptErr int

const (
	ScriptErrOK ScriptErr = ScriptErrorBase + iota
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

	/*  Operands checks */

	ScriptErrInvalidOperandSize
	ScriptErrInvalidNumberRange
	ScriptErrImpossibleEncoding
	ScriptErrInvalidSplitRange

	/*  Failed verify operations */

	ScriptErrVerify
	ScriptErrEqualVerify
	ScriptErrCheckMultiSigVerify
	ScriptErrCheckSigVerify
	ScriptErrCheckDataSigVerify
	ScriptErrNumEqualVerify

	/* Logical/Format/Canonical errors */

	ScriptErrBadOpCode
	ScriptErrDisabledOpCode
	ScriptErrInvalidStackOperation
	ScriptErrInvalidAltStackOperation
	ScriptErrUnbalancedConditional

	/* Divisor errors */

	ScriptErrDivByZero
	ScriptErrModByZero

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

	ScriptErrDiscourageUpgradableNops
	ScriptErrDiscourageUpgradableWitnessProgram

	// ScriptErrNonCompressedPubKey description script errors

	ScriptErrNonCompressedPubKey

	// ScriptErrIllegalForkID anti replay

	ScriptErrIllegalForkID
	ScriptErrMustUseForkID

	ScriptErrErrorCount

	// ScriptErrSize other errcode
	ScriptErrSize
	ScriptErrNonStandard
	ScriptErrNullData
	ScriptErrBareMultiSig
	ScriptErrMultiOpReturn
	ScriptErrInvalidSignatureEncoding

	ScriptErrInvalidOpCode
	ScriptErrInValidPubKeyOrSig
	ScriptErrScriptSigNotPushOnly
)

/*
var scriptErrorToString = map[ScriptErr]string{
	ScriptErrOK: "No error",
	ScriptErrEvalFalse: "Script evaluated without error but finished with a false/empty top stack element",
	ScriptErrVerify: "Script failed an OP_VERIFY operation",
	ScriptErrEqualVerify: "Script failed an OP_EQUALVERIFY operation",
	ScriptErrCheckMultiSigVerify: "Script failed an OP_CHECKMULTISIGVERIFY operation",
	ScriptErrCheckSigVerify: "Script failed an OP_CHECKSIGVERIFY operation",
	ScriptErrNumEqualVerify: "Script failed an OP_NUMEQUALVERIFY operation",
	ScriptErrScriptSize:
	ScriptErrPushSize:
	ScriptErrOpCount:
	ScriptErrStackSize:
	ScriptErrSigCount
	ScriptErrPubKeyCount
	ScriptErrBadOpCode
	ScriptErrDisabledOpCode
	ScriptErrInvalidStackOperation
	ScriptErrOpReturn
	ScriptErrUnbalancedConditional
	ScriptErrNegativeLockTime
	ScriptErrUnsatisfiedLockTime
	ScriptErrSigHashType
	ScriptErrSigDer
	ScriptErrMinimalData
	ScriptErrSigPushOnly
	ScriptErrSigHighs
	ScriptErrSigNullDummy
	ScriptErrPubKeyType
	ScriptErrMinimalIf
	ScriptErrSigNullFail
	ScriptErrDiscourageUpgradableNOPs
	ScriptErrDiscourageUpgradableWitnessProgram
}*/

func scriptErrorString(scriptError ScriptErr) string {
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
	case ScriptErrCheckDataSigVerify:
		return "Script failed on OP_CHECKDATASIGVERIFY operation"
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
	case ScriptErrCleanStack:
		return "Script did not clean its stack"
	case ScriptErrMinimalIf:
		return "OP_IF/NOTIF argument must be minimal"
	case ScriptErrSigNullFail:
		return "Signature must be zero for failed CHECK(MULTI)SIG operation"
	case ScriptErrIllegalForkID:
		return "Illegal use of SIGHASH_FORKID"
	case ScriptErrDiscourageUpgradableNops:
		return "NOPx reserved for soft-fork upgrades"
	case ScriptErrDiscourageUpgradableWitnessProgram:
		return "Witness version reserved for soft-fork upgrades"
	default:
		break
	}
	return "unknown error"

}

func (se ScriptErr) String() string {
	return scriptErrorString(se)
}
