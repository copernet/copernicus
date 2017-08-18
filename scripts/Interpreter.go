package scripts

import (
	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/model"
)

type Interpreter struct {
	stack *algorithm.Stack
}

func (interpreter *Interpreter) Verify(tx *model.Tx, nIn int, scriptSig *CScript, scriptPubKey *CScript, flags int32) (result bool, err error) {
	if flags&core.SCRIPT_VERIFY_SIGPUSHONLY != 0 && !scriptSig.IsPushOnly() {
		err = core.ScriptErr(core.SCRIPT_ERR_SIG_PUSHONLY)
		return
	}

	var stack, stackCopy algorithm.Stack
	result, err = interpreter.Exec(tx, nIn, &stack, scriptSig, flags)
	if err != nil {
		return
	}
	if !result {
		return
	}
	if flags&core.SCRIPT_VERIFY_P2SH != 0 {
		stackCopy = stack
	}
	result, err = interpreter.Exec(tx, nIn, &stack, scriptPubKey, flags)
	if err != nil {
		return
	}
	if !result {
		return
	}
	if stack.Empty() {
		return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
	}
	if !CastToBool(stack.Last().([]byte)) {
		return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
	}
	// Additional validation for spend-to-script-hash transactions:
	if (flags&core.SCRIPT_VERIFY_P2SH == core.SCRIPT_VERIFY_P2SH) &&
		scriptPubKey.IsPayToScriptHash() {
		// scriptSig must be literals-only or validation fails
		if !scriptSig.IsPushOnly() {
			return false, core.ScriptErr(core.SCRIPT_ERR_SIG_PUSHONLY)
		}
		algorithm.SwapStack(&stack, &stackCopy)
		// stack cannot be empty here, because if it was the P2SH  HASH <> EQUAL
		// scriptPubKey would be evaluated with an empty stack and the
		// EvalScript above would return false.
		if stack.Empty() {
			return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
		}
		pubKeySerialized := stack.Last().([]byte)
		pubKey2 := NewScriptWithRaw(pubKeySerialized)

		stack.PopStack()
		result, err = interpreter.Exec(tx, nIn, &stack, pubKey2, flags)
		if err != nil {
			return
		}
		if !result {
			return
		}
		if stack.Empty() {
			return false, core.ScriptErr(core.SCRIPT_ERR_EVAL_FALSE)
		}
		if !CastToBool(stack.Last().([]byte)) {
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

func (interpreter *Interpreter) Exec(tx *model.Tx, nIn int, stack *algorithm.Stack, script *CScript, flags int32) (result bool, err error) {
	//bnZero := NewCScriptNum(0)
	//bnOne := NewCScriptNum(1)
	//bnFalse := NewCScriptNum(0)
	//bnTrue := NewCScriptNum(1)
	//vchFalse := []byte{0}
	//vchZero := []byte{0}
	//vchTrue := []byte{1, 1}
	var vchPushValue []byte
	vfExec := algorithm.NewVector()
	altstack := algorithm.NewStack()

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
		fExec := vfExec.CountEqualElement(false) == 0
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
			stack.PushStack(vchPushValue)
		} else if fExec || (OP_IF <= parsedOpcode.opValue && parsedOpcode.opValue <= OP_ENDIF) {
			switch parsedOpcode.opValue {
			//
			// Push value
			//
			case OP_1NEGATE:
			case OP_1:
			case OP_2:
			case OP_3:
			case OP_4:
			case OP_5:
			case OP_6:
			case OP_7:
			case OP_8:
			case OP_9:
			case OP_10:
			case OP_11:
			case OP_12:
			case OP_13:
			case OP_14:
			case OP_15:
			case OP_16:
				{
					// ( -- value)
					bn := NewCScriptNum(int64(parsedOpcode.opValue) - int64(OP_1-1))
					stack.PushStack(bn.Serialize())
					// The result of these opcodes should always be the
					// minimal way to push the data they push, so no need
					// for a CheckMinimalPush here.
					break
				}
				//
				// Control
				//
			case OP_NOP:
				break
			case OP_CHECKLOCKTIMEVERIFY:
				{
					if flags&core.SCRIPT_VERIFY_CHECKLOCKTIMEVERIFY == 0 {
						// not enabled; treat as a NOP2
						if flags&core.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS ==
							core.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS {
							return false, core.ScriptErr(core.SCRIPT_ERR_DISCOURAGE_UPGRADABLE_NOPS)

						}
						break

					}
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					// Note that elsewhere numeric opcodes are limited to
					// operands in the range -2**31+1 to 2**31-1, however it
					// is legal for opcodes to produce results exceeding
					// that range. This limitation is implemented by
					// CScriptNum's default 4-byte limit.
					//
					// If we kept to that limit we'd have a year 2038
					// problem, even though the nLockTime field in
					// transactions themselves is uint32 which only becomes
					// meaningless after the year 2106.
					//
					// Thus as a special case we tell CScriptNum to accept
					// up to 5-byte bignums, which are good until 2**39-1,
					// well beyond the 2**32-1 limit of the nLockTime field
					// itself.
					topBytes, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					nLocktime, err := GetCScriptNum(topBytes.([]byte), fRequireMinimal, 5)
					if err != nil {
						return false, err
					}
					// In the rare event that the argument may be < 0 due to
					// some arithmetic being done first, you can always use
					// 0 MAX CHECKLOCKTIMEVERIFY.
					if nLocktime.Value < 0 {
						return false, core.ScriptErr(core.SCRIPT_ERR_NEGATIVE_LOCKTIME)
					}
					// Actually compare the specified lock time with the
					// transaction.
					if !CheckLockTime(nLocktime.Value, int64(tx.LockTime), tx.Ins[nIn].Sequence) {
						return false, core.ScriptErr(core.SCRIPT_ERR_UNSATISFIED_LOCKTIME)
					}
					break
				}
			case OP_CHECKSEQUENCEVERIFY:
				{
					if flags&core.SCRIPT_VERIFY_CHECKSEQUENCEVERIFY != core.SCRIPT_VERIFY_CHECKSEQUENCEVERIFY {
						// not enabled; treat as a NOP3
						if flags&core.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS == core.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS {
							return false, core.ScriptErr(core.SCRIPT_ERR_DISCOURAGE_UPGRADABLE_NOPS)

						}
						break
					}
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}

					// nSequence, like nLockTime, is a 32-bit unsigned
					// integer field. See the comment in CHECKLOCKTIMEVERIFY
					// regarding 5-byte numeric operands.
					topBytes, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					nSequence, err := GetCScriptNum(topBytes.([]byte), fRequireMinimal, 5)
					if err != nil {
						return false, err
					}
					// In the rare event that the argument may be < 0 due to
					// some arithmetic being done first, you can always use
					// 0 MAX CHECKSEQUENCEVERIFY.
					if nSequence.Value < 0 {
						return false, core.ScriptErr(core.SCRIPT_ERR_NEGATIVE_LOCKTIME)
					}

					// To provide for future soft-fork extensibility, if the
					// operand has the disabled lock-time flag set,
					// CHECKSEQUENCEVERIFY behaves as a NOP.
					if nSequence.Value&model.SEQUENCE_LOCKTIME_DISABLE_FLAG == model.SEQUENCE_LOCKTIME_DISABLE_FLAG {
						break
					}
					if !CheckSequence(nSequence.Value, int64(tx.Ins[nIn].Sequence), tx.Version) {
						return false, core.ScriptErr(core.SCRIPT_ERR_UNSATISFIED_LOCKTIME)
					}
					break
				}
			case OP_NOP1:
			case OP_NOP4:
			case OP_NOP5:
			case OP_NOP6:
			case OP_NOP7:
			case OP_NOP8:
			case OP_NOP9:
			case OP_NOP10:
				{
					if flags&core.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS == core.SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS {
						return false, core.ScriptErr(core.SCRIPT_ERR_DISCOURAGE_UPGRADABLE_NOPS)
					}
					break
				}
			case OP_IF:
			case OP_NOTIF:
				{
					// <expression> if [statements] [else [statements]]
					// endif
					fValue := false
					if fExec {
						if stack.Size() < 1 {
							return false, core.ScriptErr(core.SCRIPT_ERR_UNBALANCED_CONDITIONAL)
						}
						vch, err := stack.StackTop(-1)
						if err != nil {
							return false, err
						}
						vchBytes := vch.([]byte)
						if flags&core.SCRIPT_VERIFY_MINIMALIF == core.SCRIPT_VERIFY_MINIMALIF {
							if len(vchBytes) > 1 {
								return false, core.ScriptErr(core.SCRIPT_ERR_MINIMALIF)
							}
							if len(vchBytes) == 1 && vchBytes[0] != 1 {
								return false, core.ScriptErr(core.SCRIPT_ERR_MINIMALIF)
							}

						}
						fValue = CastToBool(vchBytes)
						if parsedOpcode.opValue == OP_NOTIF {
							fValue = !fValue
						}
						stack.PopStack()

					}
					vfExec.PushBack(fValue)
					break
				}
			case OP_ELSE:
				{
					if vfExec.Empty() {
						return false, core.ScriptErr(core.SCRIPT_ERR_UNBALANCED_CONDITIONAL)
					}
					vfBack := !vfExec.Back().(bool)
					vfExec.SetBack(vfBack)
					break
				}
			case OP_ENDIF:
				{
					if vfExec.Empty() {
						return false, core.ScriptErr(core.SCRIPT_ERR_UNBALANCED_CONDITIONAL)
					}
					vfExec.PopBack()
					break
				}
			case OP_VERIF:
				{
					// (true -- ) or
					// (false -- false) and return
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					vchBytes := vch.([]byte)
					fValue := CastToBool(vchBytes)
					if fValue {
						stack.PopStack()
					} else {
						return false, core.ScriptErr(core.SCRIPT_ERR_VERIFY)
					}
					break
				}
			case OP_RETURN:
				{
					return false, core.ScriptErr(core.SCRIPT_ERR_OP_RETURN)
				}
				//
				// Stack ops
				//
			case OP_TOALTSTACK:
				{
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					altTop, err := altstack.StackTop(-1)
					if err != nil {
						return false, err
					}
					stack.PushStack(altTop)
					altstack.PopStack()
					break
				}
			case OP_2DROP:
				{
					// (x1 x2 -- )
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					stack.PopStack()
					stack.PopStack()
					break
				}
			case OP_2DUP:
				{
					// (x1 x2 -- x1 x2 x1 x2)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_ALTSTACK_OPERATION)
					}
					vch1, err := stack.StackTop(-2)
					if err != nil {
						return false, err
					}
					vch2, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					stack.PushStack(vch1)
					stack.PushStack(vch2)
					break
				}
			case OP_3DUP:
				{
					// (x1 x2 x3 -- x1 x2 x3 x1 x2 x3)
					if stack.Size() < 3 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch1, err := stack.StackTop(-3)
					if err != nil {
						return false, err
					}
					vch2, err := stack.StackTop(-2)
					if err != nil {
						return false, err
					}
					vch3, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					stack.PushStack(vch1)
					stack.PushStack(vch2)
					stack.PushStack(vch3)
					break
				}
			case OP_2OVER:
				{
					// (x1 x2 x3 x4 -- x1 x2 x3 x4 x1 x2)
					if stack.Size() < 4 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch1, err := stack.StackTop(-4)
					if err != nil {
						return false, err
					}
					vch2, err := stack.StackTop(-3)
					if err != nil {
						return false, err
					}
					stack.PushStack(vch1)
					stack.PushStack(vch2)
					break
				}
			case OP_2ROT:
				{
					// (x1 x2 x3 x4 x5 x6 -- x3 x4 x5 x6 x1 x2)
					if stack.Size() < 6 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch1, err := stack.StackTop(-6)
					if err != nil {
						return false, err
					}
					vch2, err := stack.StackTop(-5)
					if err != nil {
						return false, err
					}
					stack.Erase(stack.Size()-6, stack.Size()-4)
					stack.PushStack(vch1)
					stack.PushStack(vch2)
					break
				}
			case OP_2SWAP:
				{
					// (x1 x2 x3 x4 -- x3 x4 x1 x2)
					if stack.Size() < 4 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					stack.Swap(stack.Size()-4, stack.Size()-2)
					stack.Swap(stack.Size()-3, stack.Size()-1)
					break
				}
			case OP_IFDUP:
				{
					// (x - 0 | x x)
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					vchBytes := vch.([]byte)
					if CastToBool(vchBytes) {
						stack.PushStack(vch)
					}
				}
			case OP_DEPTH:
				{
					// -- stacksize
					bn := NewCScriptNum(int64(stack.Size()))
					stack.PushStack(bn.Serialize())
					break
				}
			case OP_DROP:
				{
					// (x -- )
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					stack.PopStack()
					break
				}
			case OP_DUP:
				{
					// (x -- x x)
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					stack.PushStack(vch)
					break
				}
			case OP_NIP:
				{
					// (x1 x2 -- x2)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					stack.RemoveAt(stack.Size() - 2)
					break
				}
			case OP_OVER:
				{
					// (x1 x2 -- x1 x2 x1)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-2)
					if err != nil {
						return false, err
					}
					stack.PushStack(vch)
					break
				}
			case OP_PICK:
			case OP_ROLL:
				{
					// (xn ... x2 x1 x0 n - xn ... x2 x1 x0 xn)
					// (xn ... x2 x1 x0 n - ... x2 x1 x0 xn)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					scriptNum, err := GetCScriptNum(vch.([]byte), fRequireMinimal, DEFAULT_MAX_NUM_SIZE)
					if err != nil {
						return false, err

					}
					n := scriptNum.Int32()
					stack.PopStack()
					if n < 0 || n >= int32(stack.Size()) {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vchn, err := stack.StackTop(int(-n - 1))
					if err != nil {
						return false, err
					}
					if parsedOpcode.opValue == OP_ROLL {
						stack.RemoveAt(stack.Size() - int(n) - 1)
					}
					stack.PushStack(vchn)
					break
				}
			case OP_ROT:
				{
					// (x1 x2 x3 -- x2 x3 x1)
					//  x2 x1 x3  after first swap
					//  x2 x3 x1  after second swap
					if stack.Size() < 3 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					stack.Swap(stack.Size()-3, stack.Size()-2)
					stack.Swap(stack.Size()-2, stack.Size()-1)
					break
				}
			case OP_SWAP:
				{
					// (x1 x2 -- x2 x1)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					stack.Swap(stack.Size()-2, stack.Size()-1)
					break

				}
			case OP_TUCK:
				{
					// (x1 x2 -- x2 x1 x2)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					err = stack.Insert(stack.Size()-2, vch)
					if err != nil {
						return false, err
					}
					break
				}
			case OP_SIZE:
				{
					// (in -- in size)
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					size := len(vch.([]byte))
					bn := NewCScriptNum(int64(size))
					stack.PushStack(bn.Serialize())
					break
				}

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

func NewInterpreter() *Interpreter {
	return nil
}
