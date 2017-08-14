package scripts

import (
	"github.com/btcboost/copernicus/core"
)

type Interpreter struct {
	stack *Stack
}

func (interpreter *Interpreter) Verify(scriptSig *CScript, scriptPubKey *CScript, flags int32) (result bool, err error) {
	if flags&core.SCRIPT_VERIFY_SIGPUSHONLY != 0 && !scriptSig.IsPushOnly() {
		err = core.ScriptErr(core.SCRIPT_ERR_SIG_PUSHONLY)
		return
	}

	var stack, stackCopy Stack
	result, err = interpreter.Exec(&stack, scriptSig, flags)
	if err != nil {
		return
	}
	if !result {
		return
	}
	if flags&core.SCRIPT_VERIFY_P2SH != 0 {
		stackCopy = stack
	}
	result, err = interpreter.Exec(&stack, scriptPubKey, flags)
	if err != nil {
		return
	}
	if !result {
		return
	}
	if stack.Empty() {
		return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
	}
	if !CastToBool(stack.Last()) {
		return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
	}
	// Additional validation for spend-to-script-hash transactions:
	if (flags&core.SCRIPT_VERIFY_P2SH == core.SCRIPT_VERIFY_P2SH) &&
		scriptPubKey.IsPayToScriptHash() {
		// scriptSig must be literals-only or validation fails
		if !scriptSig.IsPushOnly() {
			return false, core.ScriptErr(core.SCRIPT_ERR_SIG_PUSHONLY)
		}
		Swap(&stack, &stackCopy)
		// stack cannot be empty here, because if it was the P2SH  HASH <> EQUAL
		// scriptPubKey would be evaluated with an empty stack and the
		// EvalScript above would return false.
		if stack.Empty() {
			return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
		}
		pubKeySerialized := stack.Last()
		pubKey2 := NewScriptWithRaw(pubKeySerialized)

		stack.PopStack()
		result, err = interpreter.Exec(&stack, pubKey2, flags)
		if err != nil {
			return
		}
		if !result {
			return
		}
		if stack.Empty() {
			return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
		}
		if !CastToBool(stack.Last()) {
			return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
		}

		// The CLEANSTACK check is only performed after potential P2SH evaluation,
		// as the non-P2SH evaluation of a P2SH script will obviously not result in
		// a clean stack (the P2SH inputs remain). The same holds for witness
		// evaluation.

		if flags&core.SCRIPT_VERIFY_CLEANSTACK != 0 {
			// Disallow CLEANSTACK without P2SH, as otherwise a switch
			// CLEANSTACK->P2SH+CLEANSTACK would be possible, which is not a
			// softfork (and P2SH should be one).
			if flags&core.SCRIPT_VERIFY_P2SH == 0 {
				return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
			}
			if stack.Size() != 1 {
				return false, core.ScriptErr(core.SCRIPT_ERR_CLEANSTACK)
			}
		}
		return true, nil

	}

	return
}

func (interpreter *Interpreter) Exec(stack *Stack, script *CScript, flags int32) (result bool, err error) {
	//bnZero := NewCScriptNum(0)
	//bnOne := NewCScriptNum(1)
	//bnFalse := NewCScriptNum(0)
	//bnTrue := NewCScriptNum(1)
	//vchFalse := []byte{0}
	//vchZero := []byte{0}
	//vchTrue := []byte{1, 1}
	var vchPushValue []byte
	var vfExec []bool
	//var altstack *Stack

	if script.Size() > MAX_SCRIPT_SIZE {
		return false, core.ScriptErr(core.SCRIPT_ERR_SCRIPT_SIZE)
	}
	parsedOpcodes, err := script.ParseScript()
	if err != nil {
		return false, err
	}
	nOpCount := 0
	fRequireMinimal := (flags & core.SCRIPT_VERIFY_MINIMALDATA) != 0
	for i := 0; i < len(parsedOpcodes); i++ {
		parsedOpcode := parsedOpcodes[i]
		fExec := CountEqualElement(vfExec, false) == 0
		if len(vchPushValue) > MAX_SCRIPT_ELEMENT_SIZE {
			return false, core.ScriptErr(core.SCRIPT_ERR_PUSH_SIZE)
		}
		nOpCount++
		// Note how OP_RESERVED does not count towards the opcode limit.
		if parsedOpcode.opValue > OP_16 && nOpCount > MAX_OPS_PER_SCRIPT {
			return false, core.ScriptErr(core.SCRIPT_ERR_OP_COUNT)
		}

		if parsedOpcode.opValue == OP_CAT || parsedOpcode.opValue == OP_SUBSTR || parsedOpcode.opValue == OP_LEFT ||
			parsedOpcode.opValue == OP_RIGHT || parsedOpcode.opValue == OP_INVERT || parsedOpcode.opValue == OP_AND ||
			parsedOpcode.opValue == OP_OR || parsedOpcode.opValue == OP_XOR || parsedOpcode.opValue == OP_2MUL ||
			parsedOpcode.opValue == OP_2DIV || parsedOpcode.opValue == OP_MUL || parsedOpcode.opValue == OP_DIV ||
			parsedOpcode.opValue == OP_MOD || parsedOpcode.opValue == OP_LSHIFT ||
			parsedOpcode.opValue == OP_RSHIFT {
			// Disabled opcodes.
			return false, core.ScriptErr(core.SCRIPT_ERR_DISABLED_OPCODE)
		}

		if fExec && 0 <= parsedOpcode.opValue && parsedOpcode.opValue <= OP_PUSHDATA4 {
			if fRequireMinimal &&
				!CheckMinimalPush(vchPushValue, int32(parsedOpcode.opValue)) {
				return false, core.ScriptErr(core.SCRIPT_ERR_MINIMALDATA)
			}
			stack.PushBack(vchPushValue)
		} else if fExec || (OP_IF <= parsedOpcode.opValue && parsedOpcode.opValue <= OP_ENDIF) {
			switch parsedOpcode.opValue {

			}
		}
	}

	return
}
func CastToBool(vch []byte) bool {
	for i := 0; i < len(vch); i++ {
		if vch[i] != 0 {
			// Can be negative zero
			if i == len(vch)-1 && vch[i] == 0x80 {
				return false
			}
			return true
		}
	}
	return false
}
func CountEqualElement(list []bool, value bool) int {
	count := 0
	for i := 0; i < len(list); i++ {
		if list[i] == value {
			count++
		}
	}
	return count
}
func NewInterpreter() *Interpreter {
	return nil
}
