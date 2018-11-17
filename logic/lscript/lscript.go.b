package lscript

import (
	"bytes"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
)

func VerifyScript(transaction *tx.Tx, scriptSig *script.Script, scriptPubKey *script.Script,
	nIn int, value amount.Amount, flags uint32, scriptChecker Checker) error {
	if flags&script.ScriptEnableSigHashForkID == script.ScriptEnableSigHashForkID {
		flags |= script.ScriptVerifyStrictEnc
	}
	if flags&script.ScriptVerifySigPushOnly == script.ScriptVerifySigPushOnly && !scriptSig.IsPushOnly() {
		log.Debug("ScriptErrSigPushOnly")
		return errcode.New(errcode.ScriptErrSigPushOnly)
	}
	stack := util.NewStack()
	err := EvalScript(stack, scriptSig, transaction, nIn, value, flags, scriptChecker)
	if err != nil {
		return err
	}
	stackCopy := stack.Copy()
	err = EvalScript(stack, scriptPubKey, transaction, nIn, value, flags, scriptChecker)
	if err != nil {
		return err
	}
	if stack.Empty() {
		log.Debug("ScriptErrEvalFalse")
		return errcode.New(errcode.ScriptErrEvalFalse)
	}
	vch := stack.Top(-1)
	if !script.BytesToBool(vch.([]byte)) {
		log.Debug("ScriptErrEvalFalse")
		return errcode.New(errcode.ScriptErrEvalFalse)
	}

	if flags&script.ScriptVerifyP2SH == script.ScriptVerifyP2SH && scriptPubKey.IsPayToScriptHash() {
		if !scriptSig.IsPushOnly() {
			log.Debug("ScriptErrScriptSigNotPushOnly")
			return errcode.New(errcode.ScriptErrSigPushOnly)
		}
		util.Swap(stack, stackCopy)
		topBytes := stack.Top(-1)
		stack.Pop()
		scriptPubKey2 := script.NewScriptRaw(topBytes.([]byte))
		err = EvalScript(stack, scriptPubKey2, transaction, nIn, value, flags, scriptChecker)
		if err != nil {
			return err
		}
		if stack.Empty() {
			log.Debug("ScriptErrEvalFalse")
			return errcode.New(errcode.ScriptErrEvalFalse)
		}
		vch1 := stack.Top(-1)
		if !script.BytesToBool(vch1.([]byte)) {
			log.Debug("ScriptErrEvalFalse")
			return errcode.New(errcode.ScriptErrEvalFalse)
		}
	}

	// The CLEANSTACK check is only performed after potential P2SH evaluation,
	// as the non-P2SH evaluation of a P2SH script will obviously not result in
	// a clean stack (the P2SH inputs remain). The same holds for witness
	// evaluation.
	if (flags & script.ScriptVerifyCleanStack) != 0 {
		// Disallow CLEANSTACK without P2SH, as otherwise a switch
		// CLEANSTACK->P2SH+CLEANSTACK would be possible, which is not a
		// softfork (and P2SH should be one).
		if flags&script.ScriptVerifyP2SH == 0 {
			panic("flags err")
		}
		if stack.Size() != 1 {
			log.Debug("ScriptErrCleanStack")
			return errcode.New(errcode.ScriptErrCleanStack)
		}
	}
	return nil
}

func EvalScript(stack *util.Stack, s *script.Script, transaction *tx.Tx, nIn int,
	money amount.Amount, flags uint32, scriptChecker Checker) error {

	if s.GetBadOpCode() {
		log.Debug("ScriptErrBadOpCode, txid: %s, input: %d", transaction.GetHash().String(), nIn)
		return errcode.New(errcode.ScriptErrBadOpCode)
	}
	if s.Size() > script.MaxScriptSize {
		log.Debug("ScriptErrScriptSize, txid: %s, input: %d", transaction.GetHash().String(), nIn)
		return errcode.New(errcode.ScriptErrScriptSize)
	}

	nOpCount := 0

	bnZero := script.ScriptNum{Value: 0}
	bnOne := script.ScriptNum{Value: 1}
	bnFalse := script.ScriptNum{Value: 0}
	bnTrue := script.ScriptNum{Value: 1}

	beginCodeHash := 0
	var fRequireMinimal bool
	if flags&script.ScriptVerifyMinmalData == script.ScriptVerifyMinmalData {
		fRequireMinimal = true
	} else {
		fRequireMinimal = false
	}

	var fExec bool
	stackExec := util.NewStack()
	stackAlt := util.NewStack()

	for i, e := range s.ParsedOpCodes {
		if stackExec.CountBool(false) == 0 {
			fExec = true
		} else {
			fExec = false
		}
		if len(e.Data) > script.MaxScriptElementSize {
			log.Debug("ScriptErrElementSize")
			return errcode.New(errcode.ScriptErrPushSize)
		}

		// Note how OP_RESERVED does not count towards the opCode limit.
		if e.OpValue > opcodes.OP_16 {
			nOpCount++
			if nOpCount > script.MaxOpsPerScript {
				log.Debug("ScriptErrOpCount")
				return errcode.New(errcode.ScriptErrOpCount)
			}
		}

		if script.IsOpCodeDisabled(e.OpValue, flags) {
			// Disabled opcodes.
			log.Debug("ScriptDisabledOpCode:%d, flags:%d", e.OpValue, flags)
			return errcode.New(errcode.ScriptErrDisabledOpCode)
		}

		if fExec && 0 <= e.OpValue && e.OpValue <= opcodes.OP_PUSHDATA4 {
			if fRequireMinimal && !e.CheckMinimalDataPush() {
				log.Debug("ScriptErrMinimalData")
				return errcode.New(errcode.ScriptErrMinimalData)
			}
			stack.Push(e.Data)
		} else if fExec || (opcodes.OP_IF <= e.OpValue && e.OpValue <= opcodes.OP_ENDIF) {
			switch e.OpValue {
			// Push value
			case opcodes.OP_1NEGATE:
				fallthrough
			case opcodes.OP_1:
				fallthrough
			case opcodes.OP_2:
				fallthrough
			case opcodes.OP_3:
				fallthrough
			case opcodes.OP_4:
				fallthrough
			case opcodes.OP_5:
				fallthrough
			case opcodes.OP_6:
				fallthrough
			case opcodes.OP_7:
				fallthrough
			case opcodes.OP_8:
				fallthrough
			case opcodes.OP_9:
				fallthrough
			case opcodes.OP_10:
				fallthrough
			case opcodes.OP_11:
				fallthrough
			case opcodes.OP_12:
				fallthrough
			case opcodes.OP_13:
				fallthrough
			case opcodes.OP_14:
				fallthrough
			case opcodes.OP_15:
				fallthrough
			case opcodes.OP_16:

				// ( -- value)
				bn := script.NewScriptNum(int64(e.OpValue) - int64(opcodes.OP_1-1))
				stack.Push(bn.Serialize())
				// The result of these opcodes should always be the
				// minimal way to push the data they push, so no need
				// for a CheckMinimalPush here.

				//
				// Control
				//
			case opcodes.OP_NOP:
			case opcodes.OP_CHECKLOCKTIMEVERIFY:
				if flags&script.ScriptVerifyCheckLockTimeVerify == 0 {
					// not enabled; treat as a NOP2
					if flags&script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
						log.Debug("ScriptErrDiscourageUpgradableNops")
						return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)
					}
					break
				}
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
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
				topBytes := stack.Top(-1)
				if topBytes == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				nLocktime, err := script.GetScriptNum(topBytes.([]byte), fRequireMinimal, 5)
				//nLocktime, err := script.GetScriptNum(topBytes.([]byte), true, 5)
				if err != nil {
					return err
				}
				// In the rare event that the argument may be < 0 due to
				// some arithmetic being done first, you can always use
				// 0 MAX CHECKLOCKTIMEVERIFY.
				if nLocktime.Value < 0 {
					log.Debug("ScriptErrNegativeLockTime")
					return errcode.New(errcode.ScriptErrNegativeLockTime)
				}
				// Actually compare the specified lock time with the
				// transaction.
				if !scriptChecker.CheckLockTime(nLocktime.Value, int64(transaction.GetLockTime()), transaction.GetIns()[nIn].Sequence) {
					log.Debug("ScriptErrUnsatisfiedLockTime")
					return errcode.New(errcode.ScriptErrUnsatisfiedLockTime)
				}

			case opcodes.OP_CHECKSEQUENCEVERIFY:
				if flags&script.ScriptVerifyCheckSequenceVerify == 0 {
					// not enabled; treat as a NOP3
					if flags&script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
						log.Debug("ScriptErrDiscourageUpgradableNops")
						return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)

					}
					break
				}
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				// nSequence, like nLockTime, is a 32-bit unsigned
				// integer field. See the comment in checkLockTimeVerify
				// regarding 5-byte numeric operands.
				topBytes := stack.Top(-1)
				if topBytes == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				nSequence, err := script.GetScriptNum(topBytes.([]byte), fRequireMinimal, 5)
				//nSequence, err := script.GetScriptNum(topBytes.([]byte), true, 5)
				if err != nil {
					return err
				}
				// In the rare event that the argument may be < 0 due to
				// some arithmetic being done first, you can always use
				// 0 MAX checkSequenceVerify.
				if nSequence.Value < 0 {
					log.Debug("ScriptErrNegativeLockTime")
					return errcode.New(errcode.ScriptErrNegativeLockTime)
				}

				// To provide for future soft-fork extensibility, if the
				// operand has the disabled lock-time flag set,
				// checkSequenceVerify behaves as a NOP.
				if nSequence.Value&script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
					break
				}
				if !scriptChecker.CheckSequence(nSequence.Value, int64(transaction.GetIns()[nIn].Sequence), uint32(transaction.GetVersion())) {
					log.Debug("ScriptErrUnsatisfiedLockTime")
					return errcode.New(errcode.ScriptErrUnsatisfiedLockTime)
				}

			case opcodes.OP_NOP1:
				fallthrough
			case opcodes.OP_NOP4:
				fallthrough
			case opcodes.OP_NOP5:
				fallthrough
			case opcodes.OP_NOP6:
				fallthrough
			case opcodes.OP_NOP7:
				fallthrough
			case opcodes.OP_NOP8:
				fallthrough
			case opcodes.OP_NOP9:
				fallthrough
			case opcodes.OP_NOP10:
				if flags&script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
					log.Debug("ScriptErrDiscourageUpgradableNops")
					return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)
				}
			case opcodes.OP_IF:
				fallthrough
			case opcodes.OP_NOTIF:
				// <expression> if [statements] [else [statements]]
				// endif
				fValue := false
				if fExec {
					if stack.Size() < 1 {
						log.Debug("ScriptErrUnbalancedConditional")
						return errcode.New(errcode.ScriptErrUnbalancedConditional)
					}
					vch := stack.Top(-1)
					if vch == nil {
						log.Debug("ScriptErrUnbalancedConditional")
						return errcode.New(errcode.ScriptErrUnbalancedConditional)
					}
					vchBytes := vch.([]byte)
					if flags&script.ScriptVerifyMinimalIf == script.ScriptVerifyMinimalIf {
						if len(vchBytes) > 1 {
							log.Debug("ScriptErrMinimalIf")
							return errcode.New(errcode.ScriptErrMinimalIf)
						}
						if len(vchBytes) == 1 && vchBytes[0] != 1 {
							log.Debug("ScriptErrMinimalIf")
							return errcode.New(errcode.ScriptErrMinimalIf)
						}
					}
					fValue = script.BytesToBool(vchBytes)
					if e.OpValue == opcodes.OP_NOTIF {
						fValue = !fValue
					}
					stack.Pop()
				}

				stackExec.Push(fValue)

			case opcodes.OP_ELSE:
				if stackExec.Empty() {
					log.Debug("ScriptErrUnbalancedConditional")
					return errcode.New(errcode.ScriptErrUnbalancedConditional)
				}
				vfBack := !stackExec.Top(-1).(bool)
				if !stackExec.SetTop(-1, vfBack) {
					log.Debug("ScriptErrUnbalancedConditional")
					return errcode.New(errcode.ScriptErrUnbalancedConditional)
				}
			case opcodes.OP_ENDIF:
				if stackExec.Empty() {
					log.Debug("ScriptErrUnbalancedConditional")
					return errcode.New(errcode.ScriptErrUnbalancedConditional)
				}
				stackExec.Pop()

			case opcodes.OP_VERIFY:

				// (true -- ) or
				// (false -- false) and return
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vchBytes := vch.([]byte)
				fValue := script.BytesToBool(vchBytes)
				if fValue {
					stack.Pop()
				} else {
					log.Debug("ScriptErrVerify")
					return errcode.New(errcode.ScriptErrVerify)
				}

			case opcodes.OP_RETURN:
				log.Debug("ScriptErrOpReturn")
				return errcode.New(errcode.ScriptErrOpReturn)
				//
				// Stack ops
				//
			case opcodes.OP_TOALTSTACK:

				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stackAlt.Push(vch)
				stack.Pop()

			case opcodes.OP_FROMALTSTACK:
				if stackAlt.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidAltStackOperation)
				}
				stack.Push(stackAlt.Top(-1))
				stackAlt.Pop()
			case opcodes.OP_2DROP:

				// (x1 x2 -- )
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Pop()
				stack.Pop()

			case opcodes.OP_2DUP:

				// (x1 x2 -- x1 x2 x1 x2)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-2)
				if vch1 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-1)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Push(vch1)
				stack.Push(vch2)

			case opcodes.OP_3DUP:

				// (x1 x2 x3 -- x1 x2 x3 x1 x2 x3)
				if stack.Size() < 3 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-3)
				if vch1 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-2)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch3 := stack.Top(-1)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Push(vch1)
				stack.Push(vch2)
				stack.Push(vch3)

			case opcodes.OP_2OVER:

				// (x1 x2 x3 x4 -- x1 x2 x3 x4 x1 x2)
				if stack.Size() < 4 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-4)
				if vch1 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-3)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Push(vch1)
				stack.Push(vch2)

			case opcodes.OP_2ROT:

				// (x1 x2 x3 x4 x5 x6 -- x3 x4 x5 x6 x1 x2)
				if stack.Size() < 6 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-6)
				if vch1 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-5)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Erase(stack.Size()-6, stack.Size()-4)
				stack.Push(vch1)
				stack.Push(vch2)

			case opcodes.OP_2SWAP:

				// (x1 x2 x3 x4 -- x3 x4 x1 x2)
				if stack.Size() < 4 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Swap(stack.Size()-4, stack.Size()-2)
				stack.Swap(stack.Size()-3, stack.Size()-1)

			case opcodes.OP_IFDUP:

				// (x - 0 | x x)
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vchBytes := vch.([]byte)
				if script.BytesToBool(vchBytes) {
					stack.Push(vch)
				}

			case opcodes.OP_DEPTH:

				// -- stacksize
				bn := script.NewScriptNum(int64(stack.Size()))
				stack.Push(bn.Serialize())

			case opcodes.OP_DROP:

				// (x -- )
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Pop()

			case opcodes.OP_DUP:

				// (x -- x x)
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Push(vch)

			case opcodes.OP_NIP:

				// (x1 x2 -- x2)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.RemoveAt(stack.Size() - 2)

			case opcodes.OP_OVER:

				// (x1 x2 -- x1 x2 x1)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-2)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Push(vch)

			case opcodes.OP_PICK:
				fallthrough
			case opcodes.OP_ROLL:

				// (xn ... x2 x1 x0 n - xn ... x2 x1 x0 xn)
				// (xn ... x2 x1 x0 n - ... x2 x1 x0 xn)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				scriptNum, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err

				}
				n := scriptNum.ToInt32()
				stack.Pop()
				if n < 0 || n >= int32(stack.Size()) {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vchn := stack.Top(int(-n - 1))
				if vchn == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				if e.OpValue == opcodes.OP_ROLL {
					stack.RemoveAt(stack.Size() - int(n) - 1)
				}
				stack.Push(vchn)

			case opcodes.OP_ROT:

				// (x1 x2 x3 -- x2 x3 x1)
				//  x2 x1 x3  after first swap
				//  x2 x3 x1  after second swap
				if stack.Size() < 3 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Swap(stack.Size()-3, stack.Size()-2)
				stack.Swap(stack.Size()-2, stack.Size()-1)

			case opcodes.OP_SWAP:

				// (x1 x2 -- x2 x1)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				stack.Swap(stack.Size()-2, stack.Size()-1)

			case opcodes.OP_TUCK:

				// (x1 x2 -- x2 x1 x2)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				if !stack.Insert(stack.Size()-2, vch) {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

			case opcodes.OP_SIZE:

				// (in -- in size)
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				size := len(vch.([]byte))
				bn := script.NewScriptNum(int64(size))
				stack.Push(bn.Serialize())

				//
				// Bitwise logic
				//
			case opcodes.OP_AND:
				fallthrough
			case opcodes.OP_OR:
				fallthrough
			case opcodes.OP_XOR:
				// (x1 x2 - out)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-2)
				vch2 := stack.Top(-1)
				vch1Bytes := vch1.([]byte)
				vch2Bytes := vch2.([]byte)
				lenVch1 := len(vch1Bytes)
				lenVch2 := len(vch2Bytes)
				// Inputs must be the same size
				if lenVch1 != lenVch2 {
					log.Debug("ScriptErrInvalidOperandSize")
					return errcode.New(errcode.ScriptErrInvalidOperandSize)
				}

				// To avoid allocating, we modify vch1 in place.
				switch e.OpValue {
				case opcodes.OP_AND:
					for i := 0; i < len(vch1.([]byte)); i++ {
						vch1Bytes[i] &= vch2Bytes[i]
					}
				case opcodes.OP_OR:
					for i := 0; i < len(vch1.([]byte)); i++ {
						vch1Bytes[i] |= vch2Bytes[i]
					}
				case opcodes.OP_XOR:
					for i := 0; i < len(vch1.([]byte)); i++ {
						vch1Bytes[i] ^= vch2Bytes[i]
					}
				default:
				}
				// And pop vch2.
				stack.Pop()
				stack.Pop()
				stack.Push(vch1Bytes)
			case opcodes.OP_EQUAL:
				fallthrough
			case opcodes.OP_EQUALVERIFY:
				// case opcodes.OP_NOTEQUAL: // use opcodes.OP_NUMNOTEQUAL
				// (x1 x2 - bool)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-2)
				if vch1 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-1)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				fEqual := bytes.Equal(vch1.([]byte), vch2.([]byte))
				//fEqual := reflect.DeepEqual(vch1, vch2)
				// opcodes.OP_NOTEQUAL is disabled because it would be too
				// easy to say something like n != 1 and have some
				// wiseguy pass in 1 with extra zero bytes after it
				// (numerically, 0x01 == 0x0001 == 0x000001)
				// if (opcode == opcodes.OP_NOTEQUAL)
				//    fEqual = !fEqual;
				stack.Pop()
				stack.Pop()
				if fEqual {
					stack.Push(bnTrue.Serialize())
				} else {
					stack.Push(bnFalse.Serialize())
				}
				if e.OpValue == opcodes.OP_EQUALVERIFY {
					if fEqual {
						stack.Pop()
					} else {
						log.Debug("ScriptErrInvalidStackOperation")
						return errcode.New(errcode.ScriptErrEqualVerify)
					}
				}
				//Numeric
			case opcodes.OP_1ADD:
				fallthrough
			case opcodes.OP_1SUB:
				fallthrough
			case opcodes.OP_NEGATE:
				fallthrough
			case opcodes.OP_ABS:
				// (in -- out)
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				bn, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				switch e.OpValue {
				case opcodes.OP_1ADD:
					bn.Value += bnOne.Value
				case opcodes.OP_1SUB:
					bn.Value -= bnOne.Value
				case opcodes.OP_NEGATE:
					bn.Value = -bn.Value
				case opcodes.OP_ABS:
					if bn.Value < 0 {
						bn.Value = -bn.Value
					}
				default:
					log.Debug("ScriptErrInvalidOpCode")
					return errcode.New(errcode.ScriptErrInvalidOpCode)
				}
				stack.Pop()
				stack.Push(bn.Serialize())

			case opcodes.OP_NOT:
				fallthrough
			case opcodes.OP_0NOTEQUAL:
				// (in -- out)
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				bn, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				var fValue script.ScriptNum
				switch e.OpValue {
				case opcodes.OP_NOT:
					if bn.Value == bnZero.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_0NOTEQUAL:
					if bn.Value != bnZero.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				default:
					log.Debug("ScriptErrInvalidOpCode")
					return errcode.New(errcode.ScriptErrInvalidOpCode)
				}
				stack.Pop()
				stack.Push(fValue.Serialize())
			case opcodes.OP_ADD:
				fallthrough
			case opcodes.OP_SUB:
				fallthrough
			case opcodes.OP_DIV:
				fallthrough
			case opcodes.OP_MOD:
				fallthrough
			case opcodes.OP_MIN:
				fallthrough
			case opcodes.OP_MAX:
				// (x1 x2 -- out)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-2)
				if vch1 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-1)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				bn1, err := script.GetScriptNum(vch1.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				bn2, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				bn := script.NewScriptNum(0)
				switch e.OpValue {
				case opcodes.OP_ADD:
					bn.Value = bn1.Value + bn2.Value
				case opcodes.OP_SUB:
					bn.Value = bn1.Value - bn2.Value
				case opcodes.OP_DIV:
					// denominator must not be 0
					if bn2.Value == 0 {
						log.Debug("ScriptErrDivByZero")
						return errcode.New(errcode.ScriptErrDivByZero)
					}
					bn.Value = bn1.Value / bn2.Value

				case opcodes.OP_MOD:
					// divisor must not be 0
					if bn2.Value == 0 {
						log.Debug("ScriptErrModByZero")
						return errcode.New(errcode.ScriptErrModByZero)
					}
					bn.Value = bn1.Value % bn2.Value

				case opcodes.OP_MIN:
					if bn1.Value < bn2.Value {
						bn = bn1
					} else {
						bn = bn2
					}
				case opcodes.OP_MAX:
					if bn1.Value > bn2.Value {
						bn = bn1
					} else {
						bn = bn2
					}
				default:
					log.Debug("ScriptErrInvalidOpCode")
					return errcode.New(errcode.ScriptErrInvalidOpCode)
				}
				stack.Pop()
				stack.Pop()
				stack.Push(bn.Serialize())

			case opcodes.OP_BOOLAND:
				fallthrough
			case opcodes.OP_BOOLOR:
				fallthrough
			case opcodes.OP_NUMEQUAL:
				fallthrough
			case opcodes.OP_NUMEQUALVERIFY:
				fallthrough
			case opcodes.OP_NUMNOTEQUAL:
				fallthrough
			case opcodes.OP_LESSTHAN:
				fallthrough
			case opcodes.OP_GREATERTHAN:
				fallthrough
			case opcodes.OP_LESSTHANOREQUAL:
				fallthrough
			case opcodes.OP_GREATERTHANOREQUAL:

				// (x1 x2 -- out)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-2)
				if vch1 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-1)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				bn1, err := script.GetScriptNum(vch1.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				bn2, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				var fValue script.ScriptNum
				switch e.OpValue {
				case opcodes.OP_BOOLAND:
					if bn1.Value != bnZero.Value && bn2.Value != bnZero.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_BOOLOR:
					if bn1.Value != bnZero.Value || bn2.Value != bnZero.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_NUMEQUAL:
					if bn1.Value == bn2.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_NUMEQUALVERIFY:
					if bn1.Value == bn2.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_NUMNOTEQUAL:
					if bn1.Value != bn2.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_LESSTHAN:
					if bn1.Value < bn2.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_GREATERTHAN:
					if bn1.Value > bn2.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_LESSTHANOREQUAL:
					if bn1.Value <= bn2.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				case opcodes.OP_GREATERTHANOREQUAL:
					if bn1.Value >= bn2.Value {
						fValue = bnTrue
					} else {
						fValue = bnFalse
					}
				default:
					log.Debug("ScriptErrInvalidOpCode")
					return errcode.New(errcode.ScriptErrInvalidOpCode)
				}
				stack.Pop()
				stack.Pop()
				stack.Push(fValue.Serialize())

				if e.OpValue == opcodes.OP_NUMEQUALVERIFY {
					vch := stack.Top(-1)
					fValue := script.BytesToBool(vch.([]byte))
					if fValue {
						stack.Pop()
					} else {
						log.Debug("ScriptErrNumEqualVerify")
						return errcode.New(errcode.ScriptErrNumEqualVerify)
					}
				}

			case opcodes.OP_WITHIN:
				// (x min max -- out)
				if stack.Size() < 3 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-3)
				if vch1 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-2)
				if vch2 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch3 := stack.Top(-1)
				if vch3 == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				bn1, err := script.GetScriptNum(vch1.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				bn2, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				bn3, err := script.GetScriptNum(vch3.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					return err
				}
				var fValue script.ScriptNum
				if bn2.Value <= bn1.Value && bn1.Value < bn3.Value {
					fValue = bnTrue
				} else {
					fValue = bnFalse
				}
				stack.Pop()
				stack.Pop()
				stack.Pop()
				stack.Push(fValue.Serialize())
				// Crypto
			case opcodes.OP_RIPEMD160:
				fallthrough
			case opcodes.OP_SHA1:
				fallthrough
			case opcodes.OP_SHA256:
				fallthrough
			case opcodes.OP_HASH160:
				fallthrough
			case opcodes.OP_HASH256:

				// (in -- GetHash)
				var vchHash []byte
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch := stack.Top(-1)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				switch e.OpValue {
				case opcodes.OP_RIPEMD160:
					vchHash = util.Ripemd160(vch.([]byte))
				case opcodes.OP_SHA1:
					result := util.Sha1(vch.([]byte))
					vchHash = append(vchHash, result[:]...)
				case opcodes.OP_SHA256:
					vchHash = util.Sha256Bytes(vch.([]byte))
				case opcodes.OP_HASH160:
					vchHash = util.Hash160(vch.([]byte))
				case opcodes.OP_HASH256:
					vchHash = util.DoubleSha256Bytes(vch.([]byte))
				}
				stack.Pop()
				stack.Push(vchHash)

			case opcodes.OP_CODESEPARATOR:

				// Hash starts after the code separator
				beginCodeHash = i

			case opcodes.OP_CHECKSIG:
				fallthrough
			case opcodes.OP_CHECKSIGVERIFY:

				// (sig pubkey -- bool)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vchSig := stack.Top(-2)
				if vchSig == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vchPubkey := stack.Top(-1)
				if vchPubkey == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				vchSigBytes := vchSig.([]byte)
				err := script.CheckSignatureEncoding(vchSigBytes, flags)
				if err != nil {
					return err
				}
				err = script.CheckPubKeyEncoding(vchPubkey.([]byte), flags)
				if err != nil {
					return err
				}
				// signature is DER format, the second byte + 2 indicates the len of signature
				// 0x30 if the first byte that indicates the beginning of signature
				//vchSigBytes = vchSigBytes[:vchSigBytes[1]+2]
				// Subset of script starting at the most recent codeSeparator
				scriptCode := script.NewScriptOps(s.ParsedOpCodes[beginCodeHash:])

				// Remove the signature since there is no way for a signature
				// to sign itself.

				/*var vchScript = script.NewEmptyScript()
				vchScript.PushSingleData(vchSigBytes)
				scriptCode.FindAndDelete(vchScript)*/
				scriptCode = scriptCode.RemoveOpcodeByData(vchSigBytes)

				fSuccess, err := scriptChecker.CheckSig(transaction, vchSigBytes, vchPubkey.([]byte), scriptCode, nIn, money, flags)
				if err != nil {
					return err
				}

				if !fSuccess &&
					(flags&script.ScriptVerifyNullFail == script.ScriptVerifyNullFail) &&
					len(vchSig.([]byte)) > 0 {
					log.Debug("ScriptErrSigNullFail")
					return errcode.New(errcode.ScriptErrSigNullFail)
				}

				stack.Pop()
				stack.Pop()
				if fSuccess {
					stack.Push(bnTrue.Serialize())
				} else {
					stack.Push(bnFalse.Serialize())
				}
				if e.OpValue == opcodes.OP_CHECKSIGVERIFY {
					if fSuccess {
						stack.Pop()
					} else {
						log.Debug("ScriptErrCheckSigVerify")
						return errcode.New(errcode.ScriptErrCheckSigVerify)
					}
				}

			case opcodes.OP_CHECKDATASIG:
				fallthrough
			case opcodes.OP_CHECKDATASIGVERIFY:
				// Make sure thie remains an error before activation
				if (flags & script.ScriptEnableCheckDataSig) == 0 {
					log.Debug("ScriptErrorBadOpcode")
					return errcode.New(errcode.ScriptErrBadOpCode)
				}

				// (sig message pubkey -- bool)
				if stack.Size() < 3 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				vchSig := stack.Top(-3)
				vchMessage := stack.Top(-2)
				vchPubKey := stack.Top(-1)
				if vchSig == nil || vchMessage == nil || vchPubKey == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				vchSigBytes := vchSig.([]byte)

				ok, err := script.CheckDataSignatureEncoding(vchSigBytes, flags)
				if !ok {
					return err
				}
				//
				//if err := script.CheckSignatureEncoding(vchSigBytes, flags); err != nil {
				//	log.Debug("Script check signature encoding error:%v", err)
				//	return err
				//}

				if err := script.CheckPubKeyEncoding(vchPubKey.([]byte), flags); err != nil {
					log.Debug("Script check signature encoding error:%v", err)
					return err
				}

				success := false
				if len(vchSigBytes) > 0 {
					vchHashs := util.Sha256Hash(vchMessage.([]byte))
					success = tx.CheckSig(vchHashs, vchSigBytes, vchPubKey.([]byte))
				}

				if !success && ((flags & script.ScriptVerifyNullFail) == script.ScriptVerifyNullFail) && len(
					vchSigBytes) > 0 {
					return errcode.New(errcode.ScriptErrSigNullFail)
				}

				stack.Pop()
				stack.Pop()
				stack.Pop()
				if success {
					stack.Push(bnTrue.Serialize())
				} else {
					stack.Push(bnFalse.Serialize())
				}

				if e.OpValue == opcodes.OP_CHECKDATASIGVERIFY {
					if success {
						stack.Pop()
					} else {
						return errcode.New(errcode.ScriptErrCheckDataSigVerify)
					}
				}

			case opcodes.OP_CHECKMULTISIG:
				fallthrough
			case opcodes.OP_CHECKMULTISIGVERIFY:

				// ([sig ...] num_of_signatures [pubkey ...]
				// num_of_pubkeys -- bool)
				i := 1
				if stack.Size() < i {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				vch := stack.Top(-i)
				if vch == nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				// ScriptSig1 ScriptSig2...ScriptSigM M PubKey1 PubKey2...PubKey N
				pubKeysNum, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					//log.Debug("ScriptErrInvalidStackOperation")
					return err
				}
				pubKeysCount := pubKeysNum.ToInt32()
				if pubKeysCount < 0 || pubKeysCount > script.MaxPubKeysPerMultiSig {
					log.Debug("ScriptErrOpCount")
					return errcode.New(errcode.ScriptErrPubKeyCount)
				}
				nOpCount += int(pubKeysCount)
				if nOpCount > script.MaxOpsPerScript {
					log.Debug("ScriptErrOpCount")
					return errcode.New(errcode.ScriptErrOpCount)
				}
				// skip N
				i++
				// PubKey start position
				iPubKey := i
				// iKey2 is the position of last non-signature item in
				// the stack. Top stack item = 1. With
				// ScriptVerifyNullFail, this is used for cleanup if
				// operation fails.
				iKey2 := pubKeysCount + 2
				i += int(pubKeysCount)
				if stack.Size() < i {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				sigsNumVch := stack.Top(-i)
				if err != nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				nSigsNum, err := script.GetScriptNum(sigsNumVch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					//log.Debug("ScriptErrInvalidStackOperation")
					return err
				}
				nSigsCount := nSigsNum.ToInt32()
				if nSigsCount < 0 || nSigsCount > pubKeysCount {
					log.Debug("ScriptErrSigCount")
					return errcode.New(errcode.ScriptErrSigCount)
				}
				i++
				/// Sig start position
				iSig := i
				i += int(nSigsCount)
				if stack.Size() < i {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				// Subset of script starting at the most recent codeSeparator
				scriptCode := script.NewScriptOps(s.ParsedOpCodes[beginCodeHash:])

				// Drop the signature in pre-segwit scripts but not segwit scripts
				for k := 0; k < int(nSigsCount); k++ {
					vchSig := stack.Top(-iSig - k)
					if vchSig == nil {
						log.Debug("ScriptErrInvalidStackOperation")
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					scriptCode = scriptCode.RemoveOpcodeByData(vchSig.([]byte))
				}
				fSuccess := true
				for fSuccess && nSigsCount > 0 {
					vchSig := stack.Top(-iSig)
					if vchSig == nil {
						log.Debug("ScriptErrInvalidStackOperation")
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchPubkey := stack.Top(-iPubKey)
					if vchPubkey == nil {
						log.Debug("ScriptErrInvalidStackOperation")
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					// Note how this makes the exact order of
					// pubkey/signature evaluation distinguishable by
					// CHECKMULTISIG NOT if the STRICTENC flag is set.
					// See the script_(in)valid tests for details.
					err := script.CheckSignatureEncoding(vchSig.([]byte), flags)
					if err != nil {
						return err
					}
					err = script.CheckPubKeyEncoding(vchPubkey.([]byte), flags)
					if err != nil {
						return err
					}
					fOk, err := scriptChecker.CheckSig(transaction, vchSig.([]byte), vchPubkey.([]byte), scriptCode, nIn, money, flags)
					if err != nil {
						return err
					}
					if fOk {
						iSig++
						nSigsCount--
					}
					iPubKey++
					pubKeysCount--
					// If there are more signatures left than keys left,
					// then too many signatures have failed. Exit early,
					// without checking any further signatures.
					if nSigsCount > pubKeysCount {
						fSuccess = false
					}
				}
				// Clean up stack of actual arguments
				for i > 1 {
					// If the operation failed, we require that all
					// signatures must be empty vector
					if !fSuccess && (flags&script.ScriptVerifyNullFail == script.ScriptVerifyNullFail) &&
						iKey2 == 0 && len(stack.Top(-1).([]byte)) > 0 {
						log.Debug("ScriptErrSigNullFail")
						return errcode.New(errcode.ScriptErrSigNullFail)

					}
					if iKey2 > 0 {
						iKey2--
					}
					stack.Pop()
					i--
				}
				// A bug causes CHECKMULTISIG to consume one extra
				// argument whose contents were not checked in any way.
				//
				// Unfortunately this is a potential source of
				// mutability, so optionally verify it is exactly equal
				// to zero prior to removing it from the stack.
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				if flags&script.ScriptVerifyNullDummy == script.ScriptVerifyNullDummy &&
					len(stack.Top(-1).([]byte)) > 0 {
					log.Debug("ScriptErrSigNullDummy")
					return errcode.New(errcode.ScriptErrSigNullDummy)

				}

				//pop bug byte op_0, format: op0 sig1 sig2 m pubkey1 pubk2 pubkey3 n checkmultisig
				stack.Pop()
				if fSuccess {
					stack.Push(bnTrue.Serialize())
				} else {
					stack.Push(bnFalse.Serialize())
				}
				if e.OpValue == opcodes.OP_CHECKMULTISIGVERIFY {
					if fSuccess {
						stack.Pop()
					} else {
						log.Debug("ScriptErrCheckMultiSigVerify")
						return errcode.New(errcode.ScriptErrCheckMultiSigVerify)
					}
				}
			case opcodes.OP_CAT:
				// (x1 x2 -- out)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch1 := stack.Top(-2)
				vch2 := stack.Top(-1)
				vch1Bytes := vch1.([]byte)
				vch2Bytes := vch2.([]byte)
				lenVch1 := len(vch1Bytes)
				lenVch2 := len(vch2Bytes)
				if lenVch1+lenVch2 > script.MaxScriptElementSize {
					log.Debug("ScriptErrPushSize")
					return errcode.New(errcode.ScriptErrPushSize)
				}
				stack.Pop()
				stack.Pop()
				var vch3Bytes []byte
				vch3Bytes = append(vch3Bytes, vch1Bytes...)
				vch3Bytes = append(vch3Bytes, vch2Bytes...)
				stack.Push(vch3Bytes)

			case opcodes.OP_SPLIT:
				// (in position -- x1 x2)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				vch1 := stack.Top(-2)
				vch2 := stack.Top(-1)
				scriptNum, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				position := scriptNum.Value
				// Make sure the split point is appropriate.
				if uint64(position) > uint64(len(vch1.([]byte))) {
					log.Debug("ScriptErrInvalidSplitRange")
					return errcode.New(errcode.ScriptErrInvalidSplitRange)
				}

				// Prepare the results in their own buffer as `data`
				// will be invalidated.
				vch3 := vch1.([]byte)[:position]
				vch4 := vch1.([]byte)[position:]

				// Replace existing stack values by the new values.
				stack.Pop()
				stack.Pop()
				stack.Push(vch3)
				stack.Push(vch4)
				//
				// Conversion operations
				//
			case opcodes.OP_NUM2BIN:
				// (in size -- out)
				if stack.Size() < 2 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				vch2 := stack.Top(-1)
				scriptNum, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
				if err != nil {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}
				size := scriptNum.Value
				if size > script.MaxScriptElementSize || size < 0 {
					log.Debug("ScriptErrPushSize")
					return errcode.New(errcode.ScriptErrPushSize)
				}

				stack.Pop()

				vch1 := stack.Top(-1)
				// Try to see if we can fit that number in the number of
				// byte requested.
				vchEncode := script.MinimallyEncode(vch1.([]byte))
				vchEncodeLen := len(vchEncode)
				if int64(vchEncodeLen) > size {
					// We definitively cannot.
					log.Debug("ScriptErrImpossibleEncoding")
					return errcode.New(errcode.ScriptErrImpossibleEncoding)
				}

				// We already have an element of the right size, we
				// don't need to do anything.
				if int64(vchEncodeLen) == size {
					stack.Pop()
					stack.Push(vchEncode)
					break
				}

				var signbit uint8 = 0x00
				if vchEncodeLen > 0 {
					signbit = vchEncode[vchEncodeLen-1] & 0x80
					vchEncode[vchEncodeLen-1] &= 0x7f
				}

				for i := vchEncodeLen; int64(i) < size-1; i++ {
					vchEncode = append(vchEncode, 0)
				}
				vchEncode = append(vchEncode, signbit)
				stack.Pop()
				stack.Push(vchEncode)
			case opcodes.OP_BIN2NUM:
				// (in -- out)
				if stack.Size() < 1 {
					log.Debug("ScriptErrInvalidStackOperation")
					return errcode.New(errcode.ScriptErrInvalidStackOperation)
				}

				vch := stack.Top(-1)
				vchEncode := script.MinimallyEncode(vch.([]byte))

				// The resulting number must be a valid number.
				if !script.IsMinimallyEncoded(vchEncode, script.DefaultMaxNumSize) {
					log.Debug("ScriptErrInvalidNumberRange")
					return errcode.New(errcode.ScriptErrInvalidNumberRange)
				}
				stack.Pop()
				stack.Push(vchEncode)
			default:
				return errcode.New(errcode.ScriptErrBadOpCode)
			}
		}
		if stack.Size()+stackAlt.Size() > 1000 {
			log.Debug("ScriptErrStackSize")
			return errcode.New(errcode.ScriptErrStackSize)
		}
	}

	if !stackExec.Empty() {
		log.Debug("ScriptErrUnbalancedConditional")
		return errcode.New(errcode.ScriptErrUnbalancedConditional)
	}

	return nil
}
