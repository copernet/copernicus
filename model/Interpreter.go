package model

import (
	"fmt"
	"reflect"

	"github.com/btcboost/copernicus/algorithm"
	"github.com/btcboost/copernicus/btcutil"
	"github.com/btcboost/copernicus/core"
	"github.com/pkg/errors"
)

type Interpreter struct {
	stack *algorithm.Stack
}

func (interpreter *Interpreter) Verify(tx *Tx, nIn int, scriptSig *Script, scriptPubKey *Script, flags int32) (result bool, err error) {
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

func (interpreter *Interpreter) Exec(tx *Tx, nIn int, stack *algorithm.Stack, script *Script, flags int32) (result bool, err error) {
	bnZero := NewCScriptNum(0)
	bnOne := NewCScriptNum(1)
	//bnFalse := NewCScriptNum(0)
	//bnTrue := NewCScriptNum(1)
	vchFalse := []byte{0}
	//vchZero := []byte{0}
	vchTrue := []byte{1, 1}
	var vchPushValue []byte
	vfExec := algorithm.NewVector()
	altstack := algorithm.NewStack()
	var pbegincodehash int

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
					if nSequence.Value&SEQUENCE_LOCKTIME_DISABLE_FLAG == SEQUENCE_LOCKTIME_DISABLE_FLAG {
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
				//
				// Bitwise logic
				//
			case OP_EQUAL:
			case OP_EQUALVERIFY:
				// case OP_NOTEQUAL: // use OP_NUMNOTEQUAL
				{
					// (x1 x2 - bool)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch1, err := stack.StackTop(-2)
					if err != nil {
						return false, err
					}
					vch2, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					fEqual := reflect.DeepEqual(vch2, vch1)
					// OP_NOTEQUAL is disabled because it would be too
					// easy to say something like n != 1 and have some
					// wiseguy pass in 1 with extra zero bytes after it
					// (numerically, 0x01 == 0x0001 == 0x000001)
					// if (opcode == OP_NOTEQUAL)
					//    fEqual = !fEqual;
					stack.PopStack()
					stack.PopStack()
					if fEqual {
						stack.PushStack(vchTrue)
					} else {
						stack.PushStack(vchFalse)
					}
					if parsedOpcode.opValue == OP_EQUALVERIFY {
						if fEqual {
							stack.PopStack()
						} else {
							return false, core.ScriptErr(core.SCRIPT_ERR_EQUALVERIFY)
						}
					}
					break

				}
				//Numeric
			case OP_1ADD:
			case OP_1SUB:
			case OP_NEGATE:
			case OP_ABS:
			case OP_NOT:
			case OP_0NOTEQUAL:
				{
					// (in -- out)
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					bn, err := GetCScriptNum(vch.([]byte), fRequireMinimal, DEFAULT_MAX_NUM_SIZE)
					if err != nil {
						return false, err
					}
					switch parsedOpcode.opValue {
					case OP_1ADD:
						bn.Value = bn.Value + bnOne.Value
					case OP_1SUB:
						bn.Value = bn.Value - bnOne.Value
					case OP_NEGATE:
						bn.Value = -bn.Value
					case OP_ABS:
						if bn.Value < bnZero.Value {
							bn.Value = -bn.Value
						}
					case OP_NOT:
						if bn.Value == bnZero.Value {
							bn.Value = bnOne.Value
						} else {
							bn.Value = bnZero.Value
						}
					case OP_0NOTEQUAL:
						if bn.Value != bnZero.Value {
							bn.Value = bnOne.Value
						} else {
							bn.Value = bnZero.Value
						}
					default:
						return false, errors.New("invalid opcode")
					}
					stack.PopStack()
					stack.PushStack(bn.Serialize())

				}
			case OP_ADD:
			case OP_SUB:
			case OP_BOOLAND:
			case OP_BOOLOR:
			case OP_NUMEQUAL:
			case OP_NUMEQUALVERIFY:
			case OP_NUMNOTEQUAL:
			case OP_LESSTHAN:
			case OP_GREATERTHAN:
			case OP_LESSTHANOREQUAL:
			case OP_GREATERTHANOREQUAL:
			case OP_MIN:
			case OP_MAX:
				{
					// (x1 x2 -- out)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch1, err := stack.StackTop(-2)
					if err != nil {
						return false, err
					}
					vch2, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					bn1, err := GetCScriptNum(vch1.([]byte), fRequireMinimal, DEFAULT_MAX_NUM_SIZE)
					if err != nil {
						return false, err
					}
					bn2, err := GetCScriptNum(vch2.([]byte), fRequireMinimal, DEFAULT_MAX_NUM_SIZE)
					if err != nil {
						return false, err
					}
					bn := NewCScriptNum(0)
					switch parsedOpcode.opValue {
					case OP_ADD:
						bn.Value = bn1.Value + bn2.Value
					case OP_SUB:
						bn.Value = bn1.Value - bn2.Value
					case OP_BOOLAND:
						if bn1.Value != bnZero.Value && bn2.Value != bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_BOOLOR:
						if bn1.Value != bnZero.Value || bn2.Value != bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_NUMEQUAL:
						if bn1.Value == bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_NUMEQUALVERIFY:
						if bn1.Value == bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_NUMNOTEQUAL:
						if bn1.Value != bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_LESSTHAN:
						if bn1.Value < bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_GREATERTHAN:
						if bn1.Value > bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_LESSTHANOREQUAL:
						if bn1.Value <= bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_GREATERTHANOREQUAL:
						if bn1.Value >= bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
					case OP_MIN:
						if bn1.Value < bn2.Value {
							bn = bn1
						} else {
							bn = bn2
						}
					case OP_MAX:
						if bn1.Value > bn2.Value {
							bn = bn1
						} else {
							bn = bn2
						}
					default:
						return false, errors.New("invalid opcode")
					}
					stack.PopStack()
					stack.PopStack()
					stack.PushStack(bn.Serialize())
					if parsedOpcode.opValue == OP_NUMEQUALVERIFY {
						vch, err := stack.StackTop(-2)
						if err != nil {
							return false, err
						}
						if CastToBool(vch.([]byte)) {
							stack.PopStack()
						} else {
							return false, core.ScriptErr(core.SCRIPT_ERR_NUMEQUALVERIFY)
						}
					}
				}

				// Crypto

			case OP_RIPEMD160:
			case OP_SHA1:
			case OP_SHA256:
			case OP_HASH160:
			case OP_HASH256:
				{
					// (in -- hash)
					var vchHash []byte
					if stack.Size() < 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					switch parsedOpcode.opValue {
					case OP_RIPEMD160:
						vchHash = btcutil.Ripemd160(vch.([]byte))
					case OP_SHA1:
						result := btcutil.Sha1(vch.([]byte))
						copy(vchHash[:], result[:])
					case OP_SHA256:
						vchHash = core.Sha256Bytes(vch.([]byte))
					case OP_HASH160:
						vchHash = btcutil.Hash160(vch.([]byte))
					case OP_HASH256:
						vchHash = core.Sha256Bytes(vch.([]byte))
					}
					stack.PopStack()
					stack.PushStack(vchHash)

				}
			case OP_CODESEPARATOR:
				{
					// Hash starts after the code separator
					pbegincodehash = i

				}
			case OP_CHECKSIG:
			case OP_CHECKSIGVERIFY:
				{
					// (sig pubkey -- bool)
					if stack.Size() < 2 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vchSig, err := stack.StackTop(-2)
					if err != nil {
						return false, err
					}
					vchPubkey, err := stack.StackTop(-1)
					if err != nil {
						return false, err
					}
					checkSig, err := core.CheckSignatureEncoding(vchSig.([]byte), uint32(flags))
					if err != nil {
						return false, err
					}
					checkPubKey, err := core.CheckPubKeyEncoding(vchPubkey.([]byte), uint32(flags))
					if err != nil {
						return false, err
					}
					if !checkPubKey || !checkSig {
						return false, errors.New("check public key or sig failed")
					}
					// Subset of script starting at the most recent
					// codeseparator
					scriptCode := NewScriptWithRaw(script.bytes[pbegincodehash:])
					fmt.Print(scriptCode)
					txHash, err := SignatureHash(tx, scriptCode, int(flags), nIn)
					if err != nil {
						return false, err
					}
					fSuccess, err := CheckSig(txHash, vchSig.([]byte), vchPubkey.([]byte))
					if !fSuccess &&
						(flags&core.SCRIPT_VERIFY_NULLFAIL == core.SCRIPT_VERIFY_NULLFAIL) &&
						len(vchSig.([]byte)) > 0 {
						return false, core.ScriptErr(core.SCRIPT_ERR_SIG_NULLFAIL)

					}
					stack.PopStack()
					stack.PopStack()
					if fSuccess {
						stack.PushStack(vchTrue)
					} else {
						stack.PushStack(vchFalse)
					}
					if parsedOpcode.opValue == OP_CHECKSIGVERIFY {
						if fSuccess {
							stack.PopStack()
						} else {
							return false, core.ScriptErr(core.SCRIPT_ERR_CHECKSIGVERIFY)
						}
					}
				}
			case OP_CHECKMULTISIG:
			case OP_CHECKMULTISIGVERIFY:
				{
					// ([sig ...] num_of_signatures [pubkey ...]
					// num_of_pubkeys -- bool)
					i := 1
					if stack.Size() < i {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}

					vch, err := stack.StackTop(-i)
					if err != nil {
						return false, err
					}
					nKeysNum, err := GetCScriptNum(vch.([]byte), fRequireMinimal, DEFAULT_MAX_NUM_SIZE)
					if err != nil {
						return false, err
					}
					nKeysCount := nKeysNum.Int32()
					if nKeysCount < 0 || nKeysCount > MAX_OPS_PER_SCRIPT {
						return false, core.ScriptErr(core.SCRIPT_ERR_OP_COUNT)
					}
					i++
					iKey := i
					// ikey2 is the position of last non-signature item in
					// the stack. Top stack item = 1. With
					// SCRIPT_VERIFY_NULLFAIL, this is used for cleanup if
					// operation fails.
					iKey2 := nKeysCount + 2
					i += int(nKeysCount)
					if stack.Size() < i {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					sigsVch, err := stack.StackTop(-i)
					if err != nil {
						return false, err
					}
					nSigsNum, err := GetCScriptNum(sigsVch.([]byte), fRequireMinimal, DEFAULT_MAX_NUM_SIZE)
					if err != nil {
						return false, err
					}
					nSigsCount := nSigsNum.Int32()
					if nSigsCount < 0 || nSigsCount > nKeysCount {
						return false, core.ScriptErr(core.SCRIPT_ERR_SIG_COUNT)
					}
					i++
					isig := i
					i += int(nSigsCount)
					if stack.Size() < i {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}

					// Subset of script starting at the most recent
					// codeseparator
					scriptCode := NewScriptWithRaw(script.bytes[pbegincodehash:])

					// Drop the signature in pre-segwit scripts but not
					// segwit scripts
					for k := 0; k < int(nSigsCount); k++ {
						vchSig, err := stack.StackTop(-isig - k)
						if err != nil {
							return false, err
						}
						CleanupScriptCode(scriptCode, vchSig.([]byte), flags)
					}
					fSuccess := true
					for fSuccess && nSigsCount > 0 {
						vchSig, err := stack.StackTop(-isig)
						if err != nil {
							return false, err
						}
						vchPubkey, err := stack.StackTop(-iKey)
						if err != nil {
							return false, err
						}
						// Note how this makes the exact order of
						// pubkey/signature evaluation distinguishable by
						// CHECKMULTISIG NOT if the STRICTENC flag is set.
						// See the script_(in)valid tests for details.
						checkSig, err := core.CheckSignatureEncoding(vchSig.([]byte), uint32(flags))
						if err != nil {
							return false, err
						}
						checkPubKey, err := core.CheckPubKeyEncoding(vchPubkey.([]byte), uint32(flags))
						if err != nil {
							return false, err
						}
						if !checkSig || !checkPubKey {
							return false, errors.New("check sig or public key failed")
						}
						txHash, err := SignatureHash(tx, scriptCode, int(flags), nIn)
						if err != nil {
							return false, err
						}
						fOk, err := CheckSig(txHash, vchSig.([]byte), vchPubkey.([]byte))
						if err != nil {
							return false, err
						}
						if fOk {
							isig++
							nSigsCount--
						}
						iKey++
						nKeysCount--
						// If there are more signatures left than keys left,
						// then too many signatures have failed. Exit early,
						// without checking any further signatures.
						if nSigsCount > nKeysCount {
							fSuccess = false
						}
					}
					// Clean up stack of actual arguments

					for i > 1 {
						vch, err := stack.StackTop(-isig)
						if err != nil {
							return false, err
						}
						if !fSuccess &&
							(flags&core.SCRIPT_VERIFY_NULLFAIL == core.SCRIPT_VERIFY_NULLFAIL) &&
							iKey2 == 0 && len(vch.([]byte)) > 0 {
							return false, core.ScriptErr(core.SCRIPT_ERR_SIG_NULLFAIL)

						}
						if iKey2 > 0 {
							iKey2--
						}
						stack.PopStack()
						i--
					}
					// A bug causes CHECKMULTISIG to consume one extra
					// argument whose contents were not checked in any way.
					//
					// Unfortunately this is a potential source of
					// mutability, so optionally verify it is exactly equal
					// to zero prior to removing it from the stack.
					if stack.Size() > 1 {
						return false, core.ScriptErr(core.SCRIPT_ERR_INVALID_STACK_OPERATION)
					}
					vch, err = stack.StackTop(-isig)
					if err != nil {
						return false, err
					}
					if flags&core.SCRIPT_VERIFY_NULLFAIL == core.SCRIPT_VERIFY_NULLFAIL && len(vch.([]byte)) > 0 {
						return false, err
					}
					stack.PopStack()
					if fSuccess {
						stack.PushStack(vchTrue)
					} else {
						stack.PushStack(vchFalse)
					}
					if parsedOpcode.opValue == OP_CHECKMULTISIGVERIFY {
						if fSuccess {
							stack.PopStack()
						} else {
							return false, core.ScriptErr(core.SCRIPT_ERR_CHECKMULTISIGVERIFY)
						}
					}
					if stack.Size()+altstack.Size() > 1000 {
						return false, core.ScriptErr(core.SCRIPT_ERR_STACK_SIZE)
					}
				}

			}
		}
	}
	return true, nil
}

func CleanupScriptCode(scriptCode *Script, vchSig []byte, flags int32) {
	nHashType := GetHashType(vchSig)
	fmt.Print(nHashType)
	scriptCode.FindAndDelete(scriptCode)
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
