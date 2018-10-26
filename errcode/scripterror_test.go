package errcode

import "testing"

func TestScriptErr_String(t *testing.T) {
	tests := []struct {
		in   ScriptErr
		want string
	}{
		{ScriptErrOK, "No error"},
		{ScriptErrUnknownError, "unknown error"},
		{ScriptErrEvalFalse, "Script evaluated without error but finished with a false/empty top stack element"},
		{ScriptErrOpReturn, "OP_RETURN was encountered"},
		/* Max sizes */
		{ScriptErrScriptSize, "Script is too big"},
		{ScriptErrPushSize, "Push value size limit exceeded"},
		{ScriptErrOpCount, "Operation limit exceeded"},
		{ScriptErrStackSize, "Stack size limit exceeded"},
		{ScriptErrSigCount, "Signature count negative or greater than pubKey count"},
		{ScriptErrPubKeyCount, "PubKey count negative or limit exceeded"},
		/*  Operands checks */
		{ScriptErrInvalidOperandSize, "unknown error"},
		{ScriptErrInvalidNumberRange, "unknown error"},
		{ScriptErrImpossibleEncoding, "unknown error"},
		{ScriptErrInvalidSplitRange, "unknown error"},
		/*  Failed verify operations */
		{ScriptErrVerify, "Script failed an OP_VERIFY operation"},
		{ScriptErrEqualVerify, "Script failed an OP_EQUALVERIFY operation"},
		{ScriptErrCheckMultiSigVerify, "Script failed an OP_CHECKMULTISIGVERIFY operation"},
		{ScriptErrCheckSigVerify, "Script failed an OP_CHECKSIGVERIFY operation"},
		{ScriptErrNumEqualVerify, "Script failed an OP_NUMEQUALVERIFY operation"},
		/* Logical/Format/Canonical errors */
		{ScriptErrBadOpCode, "OpCode missing or not understood"},
		{ScriptErrDisabledOpCode, "Attempted to use a disabled opCode"},
		{ScriptErrInvalidStackOperation, "Operation not valid with the current stack size"},
		{ScriptErrInvalidAltStackOperation, "Operation not valid with the current altStack size"},
		{ScriptErrUnbalancedConditional, "Invalid OP_IF construction"},
		/* Divisor errors */
		{ScriptErrDivByZero, "unknown error"},
		{ScriptErrModByZero, "unknown error"},
		/* CheckLockTimeVerify and CheckSequenceVerify */
		{ScriptErrNegativeLockTime, "Negative lockTime"},
		{ScriptErrUnsatisfiedLockTime, "LockTime requirement not satisfied"},
		/* Malleability */
		{ScriptErrSigHashType, "Signature hash type missing or not understood"},
		{ScriptErrSigDer, "Non-canonical DER signature"},
		{ScriptErrMinimalData, "Data push larger than necessary"},
		{ScriptErrSigPushOnly, "Only non-push operators allowed in signatures"},
		{ScriptErrSigHighs, "Non-canonical signature: S value is unnecessarily high"},
		{ScriptErrSigNullDummy, "Dummy CheckMultiSig argument must be zero"},
		{ScriptErrPubKeyType, "Public key is neither compressed or uncompressed"},
		{ScriptErrCleanStack, "Script did not clean its stack"},
		{ScriptErrMinimalIf, "OP_IF/NOTIF argument must be minimal"},
		{ScriptErrSigNullFail, "Signature must be zero for failed CHECK(MULTI)SIG operation"},
		/* softFork safeness */
		{ScriptErrDiscourageUpgradableNops, "NOPx reserved for soft-fork upgrades"},
		{ScriptErrDiscourageUpgradableWitnessProgram, "Witness version reserved for soft-fork upgrades"},
		// ScriptErrNonCompressedPubKey description script errors
		{ScriptErrNonCompressedPubKey, "unknown error"},
		// ScriptErrIllegalForkID anti replay
		{ScriptErrIllegalForkID, "Illegal use of SIGHASH_FORKID"},
		{ScriptErrMustUseForkID, "unknown error"},
		{ScriptErrErrorCount, "unknown error"},
		// ScriptErrSize other errcode
		{ScriptErrSize, "unknown error"},
		{ScriptErrNonStandard, "unknown error"},
		{ScriptErrNullData, "unknown error"},
		{ScriptErrBareMultiSig, "unknown error"},
		{ScriptErrMultiOpReturn, "unknown error"},
		{ScriptErrInvalidSignatureEncoding, "unknown error"},
		{ScriptErrInvalidOpCode, "unknown error"},
		{ScriptErrInValidPubKeyOrSig, "unknown error"},
		{ScriptErrScriptSigNotPushOnly, "unknown error"},
	}

	if len(tests)-1 != int(ScriptErrScriptSigNotPushOnly)-int(ScriptErrOK) {
		t.Errorf("It appears an error code was added without adding an " +
			"associated stringer test")
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result := test.in.String()
		if result != test.want {
			t.Errorf("String #%d\n got: %s want: %s", i, result,
				test.want)
			continue
		}
	}
}
