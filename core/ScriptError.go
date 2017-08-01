package core

import "github.com/pkg/errors"

type ScriptError int

const (
	SCRIPT_ERR_OK ScriptError = iota
	SCRIPT_ERR_UNKNOWN_ERROR
	SCRIPT_ERR_EVAL_FALSE
	SCRIPT_ERR_OP_RETURN

	/* Max sizes */

	SCRIPT_ERR_SCRIPT_SIZE
	SCRIPT_ERR_PUSH_SIZE
	SCRIPT_ERR_OP_COUNT
	SCRIPT_ERR_STACK_SIZE
	SCRIPT_ERR_SIG_COUNT
	SCRIPT_ERR_PUBKEY_COUNT

	/* Failed verify operations */

	SCRIPT_ERR_VERIFY
	SCRIPT_ERR_EQUALVERIFY
	SCRIPT_ERR_CHECKMULTISIGVERIFY
	SCRIPT_ERR_CHECKSIGVERIFY
	SCRIPT_ERR_NUMEQUALVERIFY

	/* Logical/Format/Canonical errors */

	SCRIPT_ERR_BAD_OPCODE
	SCRIPT_ERR_DISABLED_OPCODE
	SCRIPT_ERR_INVALID_STACK_OPERATION
	SCRIPT_ERR_INVALID_ALTSTACK_OPERATION
	SCRIPT_ERR_UNBALANCED_CONDITIONAL

	/* CHECKLOCKTIMEVERIFY and CHECKSEQUENCEVERIFY */

	SCRIPT_ERR_NEGATIVE_LOCKTIME
	SCRIPT_ERR_UNSATISFIED_LOCKTIME

	/* Malleability */

	SCRIPT_ERR_SIG_HASHTYPE
	SCRIPT_ERR_SIG_DER
	SCRIPT_ERR_MINIMALDATA
	SCRIPT_ERR_SIG_PUSHONLY
	SCRIPT_ERR_SIG_HIGH_S
	SCRIPT_ERR_SIG_NULLDUMMY
	SCRIPT_ERR_PUBKEYTYPE
	SCRIPT_ERR_CLEANSTACK
	SCRIPT_ERR_MINIMALIF
	SCRIPT_ERR_SIG_NULLFAIL

	/* softfork safeness */

	SCRIPT_ERR_DISCOURAGE_UPGRADABLE_NOPS
	SCRIPT_ERR_DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM

	/* segregated witness */

	SCRIPT_ERR_WITNESS_PROGRAM_WRONG_LENGTH
	SCRIPT_ERR_WITNESS_PROGRAM_WITNESS_EMPTY
	SCRIPT_ERR_WITNESS_PROGRAM_MISMATCH
	SCRIPT_ERR_WITNESS_MALLEATED
	SCRIPT_ERR_WITNESS_MALLEATED_P2SH
	SCRIPT_ERR_WITNESS_UNEXPECTED
	SCRIPT_ERR_WITNESS_PUBKEYTYPE

	SCRIPT_ERR_ERROR_COUNT

	/* misc */

	SCRIPT_ERR_NONCOMPRESSED_PUBKEY
)

func ScriptErrorString(scriptError ScriptError) string {
	switch scriptError {
	case SCRIPT_ERR_OK:
		return "No error"
	case SCRIPT_ERR_EVAL_FALSE:
		return "Script evaluated without error but finished with a false/empty top stack element"
	case SCRIPT_ERR_VERIFY:
		return "Script failed an OP_VERIFY operation"
	case SCRIPT_ERR_EQUALVERIFY:
		return "Script failed an OP_EQUALVERIFY operation"
	case SCRIPT_ERR_CHECKMULTISIGVERIFY:
		return "Script failed an OP_CHECKMULTISIGVERIFY operation"
	case SCRIPT_ERR_CHECKSIGVERIFY:
		return "Script failed an OP_CHECKSIGVERIFY operation"
	case SCRIPT_ERR_NUMEQUALVERIFY:
		return "Script failed an OP_NUMEQUALVERIFY operation"
	case SCRIPT_ERR_SCRIPT_SIZE:
		return "Script is too big"
	case SCRIPT_ERR_PUSH_SIZE:
		return "Push value size limit exceeded"
	case SCRIPT_ERR_OP_COUNT:
		return "Operation limit exceeded"
	case SCRIPT_ERR_STACK_SIZE:
		return "Stack size limit exceeded"
	case SCRIPT_ERR_SIG_COUNT:
		return "Signature count negative or greater than pubkey count"
	case SCRIPT_ERR_PUBKEY_COUNT:
		return "Pubkey count negative or limit exceeded"
	case SCRIPT_ERR_BAD_OPCODE:
		return "Opcode missing or not understood"
	case SCRIPT_ERR_DISABLED_OPCODE:
		return "Attempted to use a disabled opcode"
	case SCRIPT_ERR_INVALID_STACK_OPERATION:
		return "Operation not valid with the current stack size"
	case SCRIPT_ERR_INVALID_ALTSTACK_OPERATION:
		return "Operation not valid with the current altstack size"
	case SCRIPT_ERR_OP_RETURN:
		return "OP_RETURN was encountered"
	case SCRIPT_ERR_UNBALANCED_CONDITIONAL:
		return "Invalid OP_IF construction"
	case SCRIPT_ERR_NEGATIVE_LOCKTIME:
		return "Negative locktime"
	case SCRIPT_ERR_UNSATISFIED_LOCKTIME:
		return "Locktime requirement not satisfied"
	case SCRIPT_ERR_SIG_HASHTYPE:
		return "Signature hash type missing or not understood"
	case SCRIPT_ERR_SIG_DER:
		return "Non-canonical DER signature"
	case SCRIPT_ERR_MINIMALDATA:
		return "Data push larger than necessary"
	case SCRIPT_ERR_SIG_PUSHONLY:
		return "Only non-push operators allowed in signatures"
	case SCRIPT_ERR_SIG_HIGH_S:
		return "Non-canonical signature: S value is unnecessarily high"
	case SCRIPT_ERR_SIG_NULLDUMMY:
		return "Dummy CHECKMULTISIG argument must be zero"
	case SCRIPT_ERR_MINIMALIF:
		return "OP_IF/NOTIF argument must be minimal"
	case SCRIPT_ERR_SIG_NULLFAIL:
		return "Signature must be zero for failed CHECK(MULTI)SIG operation"
	case SCRIPT_ERR_DISCOURAGE_UPGRADABLE_NOPS:
		return "NOPx reserved for soft-fork upgrades"
	case SCRIPT_ERR_DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM:
		return "Witness version reserved for soft-fork upgrades"
	case SCRIPT_ERR_PUBKEYTYPE:
		return "Public key is neither compressed or uncompressed"
	case SCRIPT_ERR_WITNESS_PROGRAM_WRONG_LENGTH:
		return "Witness program has incorrect length"
	case SCRIPT_ERR_WITNESS_PROGRAM_WITNESS_EMPTY:
		return "Witness program was passed an empty witness"
	case SCRIPT_ERR_WITNESS_PROGRAM_MISMATCH:
		return "Witness program hash mismatch"
	case SCRIPT_ERR_WITNESS_MALLEATED:
		return "Witness requires empty scriptSig"
	case SCRIPT_ERR_WITNESS_MALLEATED_P2SH:
		return "Witness requires only-redeemscript scriptSig"
	case SCRIPT_ERR_WITNESS_UNEXPECTED:
		return "Witness provided for non-witness script"
	case SCRIPT_ERR_WITNESS_PUBKEYTYPE:
		return "Using non-compressed keys in segwit"
	case SCRIPT_ERR_UNKNOWN_ERROR:
	case SCRIPT_ERR_ERROR_COUNT:
	default:
		break
	}
	return "unknown error"

}

func ScriptErr(scriptError ScriptError) error {
	str := ScriptErrorString(scriptError)
	return errors.Errorf("script error :%s code:%d", str, scriptError)
}
