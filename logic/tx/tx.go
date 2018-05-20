package tx

import (
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"

	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/model/opcodes"
	"bytes"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/util/amount"
)

func CheckRegularTransaction(tx *tx.Tx, allowLargeOpReturn bool) error {
	return nil
}

func CheckBlockCoinBaseTransaction(tx *tx.Tx, allowLargeOpReturn bool) error {
	return nil
}

func CheckBlockRegularTransaction(tx *tx.Tx, allowLargeOpReturn bool) error {
	tempCoinsMap :=  utxo.NewEmptyCoinsMap()

	err := checkInputs(tx, tempCoinsMap, 1)
	if err != nil{
		return err
	}

	return nil
}

func SubmitTransaction(txs []*tx.Tx) bool {
	return true
}
/*
func UndoTransaction(txs []*txundo.TxUndo) bool {
	return true
}
*/

func checkInputs(tx *tx.Tx, tempCoinMap *utxo.CoinsMap, flags uint32) error {
	ins := tx.GetIns()
	for i, in := range ins {
		coin := tempCoinMap.GetCoin(in.PreviousOutPoint)
		if coin == nil {
			return errcode.New(errcode.ErrorNoPreviousOut)
		}
		scriptPubKey := coin.GetTxOut().GetScriptPubKey()
		scriptSig := in.GetScriptSig()
		if flags & script.ScriptEnableSigHashForkId != 0 {
			flags |= script.ScriptVerifyStrictEnc
		}
		if flags & script.ScriptVerifySigPushOnly != 0 && !scriptSig.IsPushOnly() {
			return errcode.New(errcode.ScriptErrSigPushOnly)
		}
		stack := util.NewStack()
		err := EvalScript(stack, scriptSig, tx, i, coin.GetAmount(), flags)
		if err != nil {
			return err
		}

		err = EvalScript(stack, scriptPubKey, tx, i, coin.GetAmount(), flags)
		if err != nil {
			return err
		}
		if stack.Empty() {
			return errcode.New(errcode.ScriptErrEvalFalse)
		}

	}
	return nil
}

func EvalScript(stack *util.Stack, s *script.Script, transaction *tx.Tx, nIn int,
	money amount.Amount, flags uint32) error {
	nOpCount := 0

	bnZero :=  script.ScriptNum{0}
	bnOne :=  script.ScriptNum{1}
	bnFalse :=  script.ScriptNum{0}
	bnTrue :=  script.ScriptNum{1}

	beginCodeHash := 0

	for i, e := range s.ParsedOpCodes {
		if len(e.Data) > script.MaxScriptElementSize {
			return errcode.New(errcode.ScriptErrPushSize)
		}
		nOpCount++
		// Note how OP_RESERVED does not count towards the opCode limit.
		if e.OpValue > opcodes.OP_16 && nOpCount > script.MaxOpsPerScript {
			return errcode.New(errcode.ScriptErrOpCount)
		}

		if e.OpValue == opcodes.OP_CAT || e.OpValue == opcodes.OP_SUBSTR || e.OpValue == opcodes.OP_LEFT ||
			e.OpValue == opcodes.OP_RIGHT || e.OpValue == opcodes.OP_INVERT || e.OpValue == opcodes.OP_AND ||
			e.OpValue == opcodes.OP_OR || e.OpValue == opcodes.OP_XOR || e.OpValue == opcodes.OP_2MUL ||
			e.OpValue == opcodes.OP_2DIV || e.OpValue == opcodes.OP_MUL || e.OpValue == opcodes.OP_DIV ||
			e.OpValue == opcodes.OP_MOD || e.OpValue == opcodes.OP_LSHIFT ||
			e.OpValue == opcodes.OP_RSHIFT {
			// Disabled opcodes.
			return errcode.New(errcode.ScriptErrDisabledOpCode)
		}

		//if fExec && 0 <= e.OpValue && e.OpValue <= opcodes.OP_PUSHDATA4 {
		if 0 <= e.OpValue && e.OpValue <= opcodes.OP_PUSHDATA4 {
			//if fRequireMinimal && !e.CheckMinimalDataPush() {
			//	return errcode.New(errcode.ScriptErrMinimalData)
			//}
			stack.Push(e.Data)
		//} else if fExec || (opcodes.OP_IF <= e.OpValue && e.OpValue <= opcodes.OP_ENDIF) {
		} else if (opcodes.OP_IF <= e.OpValue && e.OpValue <= opcodes.OP_ENDIF) {
			switch e.OpValue {
			//
			// Push value
			//
			case opcodes.OP_1NEGATE:
			case opcodes.OP_1:
			case opcodes.OP_2:
			case opcodes.OP_3:
			case opcodes.OP_4:
			case opcodes.OP_5:
			case opcodes.OP_6:
			case opcodes.OP_7:
			case opcodes.OP_8:
			case opcodes.OP_9:
			case opcodes.OP_10:
			case opcodes.OP_11:
			case opcodes.OP_12:
			case opcodes.OP_13:
			case opcodes.OP_14:
			case opcodes.OP_15:
			case opcodes.OP_16:
				{
					// ( -- value)
					bn := script.NewScriptNum(int64(e.OpValue) - int64(opcodes.OP_1 - 1))
					stack.Push(bn.Serialize())
					// The result of these opcodes should always be the
					// minimal way to push the data they push, so no need
					// for a CheckMinimalPush here.
					break
				}
				//
				// Control
				//
			case opcodes.OP_NOP:
				break
			case opcodes.OP_CHECKLOCKTIMEVERIFY:
				{
					if flags & script.ScriptVerifyCheckLockTimeVerify == script.ScriptVerifyCheckLockTimeVerify {
						// not enabled; treat as a NOP2
						if flags & script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
							return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)
						}
						break
					}
					if stack.Size() < 1 {
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
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					//nLocktime, err := script.GetScriptNum(topBytes.([]byte), fRequireMinimal, 5)
					nLocktime, err := script.GetScriptNum(topBytes.([]byte), true, 5)
					if err != nil {
						return err
					}
					// In the rare event that the argument may be < 0 due to
					// some arithmetic being done first, you can always use
					// 0 MAX CHECKLOCKTIMEVERIFY.
					if nLocktime.Value < 0 {
						return errcode.New(errcode.ScriptErrNegativeLockTime)
					}
					// Actually compare the specified lock time with the
					// transaction.
					if !checkLockTime(nLocktime.Value, int64(transaction.GetLockTime()), transaction.GetIns()[nIn].Sequence) {
						return errcode.New(errcode.ScriptErrUnsatisfiedLockTime)
					}
					break
				}
			case opcodes.OP_CHECKSEQUENCEVERIFY:
				{
					if flags & script.ScriptVerifyCheckSequenceVerify == script.ScriptVerifyCheckSequenceVerify {
						// not enabled; treat as a NOP3
						if flags & script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
							return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)

						}
						break
					}
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					// nSequence, like nLockTime, is a 32-bit unsigned
					// integer field. See the comment in checkLockTimeVerify
					// regarding 5-byte numeric operands.
					topBytes := stack.Top(-1)
					if topBytes == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					//nSequence, err := script.GetScriptNum(topBytes.([]byte), fRequireMinimal, 5)
					nSequence, err := script.GetScriptNum(topBytes.([]byte), true, 5)
					if err != nil {
						return err
					}
					// In the rare event that the argument may be < 0 due to
					// some arithmetic being done first, you can always use
					// 0 MAX checkSequenceVerify.
					if nSequence.Value < 0 {
						return errcode.New(errcode.ScriptErrNegativeLockTime)
					}

					// To provide for future soft-fork extensibility, if the
					// operand has the disabled lock-time flag set,
					// checkSequenceVerify behaves as a NOP.
					if nSequence.Value & script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
						break
					}
					if !checkSequence(nSequence.Value, int64(transaction.GetIns()[nIn].Sequence), uint32(transaction.GetVersion())) {
						return errcode.New(errcode.ScriptErrUnsatisfiedLockTime)
					}
					break
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
				{
					if flags & script.ScriptVerifyDiscourageUpgradableNops == script.ScriptVerifyDiscourageUpgradableNops {
						return errcode.New(errcode.ScriptErrDiscourageUpgradableNops)
					}
					break
				}
			case opcodes.OP_IF:
				fallthrough
			case opcodes.OP_NOTIF:
				{
					// <expression> if [statements] [else [statements]]
					// endif
					fValue := false
					//if fExec {
						if stack.Size() < 1 {
							return errcode.New(errcode.ScriptErrUnbalancedConditional)
						}
						vch := stack.Top(-1)
						if vch == nil {
							return errcode.New(errcode.ScriptErrUnbalancedConditional)
						}
						vchBytes := vch.([]byte)
						if flags & script.ScriptVerifyMinimalIf == script.ScriptVerifyMinimalIf {
							if len(vchBytes) > 1 {
								return errcode.New(errcode.ScriptErrMinimalIf)
							}
							if len(vchBytes) == 1 && vchBytes[0] != 1 {
								return errcode.New(errcode.ScriptErrMinimalIf)
							}
						}
						fValue = script.BytesToBool(vchBytes)
						if e.OpValue == opcodes.OP_NOTIF {
							fValue = !fValue
						}
						stack.Pop()
					//}
					//vfExec.PushBack(fValue)
					break
				}
			case opcodes.OP_ELSE:
				//{
				//	if vfExec.Empty() {
				//		return errcode.New(errcode.ScriptErrUnbalancedConditional)
				//	}
				//	vfBack := !vfExec.Back().(bool)
				//	vfExec.SetBack(vfBack)
				//	break
				//}
			case opcodes.OP_ENDIF:
				//{
				//	if vfExec.Empty() {
				//		return errcode.New(errcode.ScriptErrUnbalancedConditional)
				//	}
				//	vfExec.PopBack()
				//	break
				//}
			case opcodes.OP_VERIFY:
				{
					// (true -- ) or
					// (false -- false) and return
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchBytes := vch.([]byte)
					fValue := script.BytesToBool(vchBytes)
					if fValue {
						stack.Pop()
					} else {
						return errcode.New(errcode.ScriptErrVerify)
					}
					break
				}
			case opcodes.OP_RETURN:
				{
					return errcode.New(errcode.ScriptErrOpReturn)
				}
				//
				// Stack ops
				//
			case opcodes.OP_TOALTSTACK:
				{
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					//altstack.Push(vch)
					stack.Pop()
					break
				}
			case opcodes.OP_2DROP:
				{
					// (x1 x2 -- )
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Pop()
					stack.Pop()
					break
				}
			case opcodes.OP_2DUP:
				{
					// (x1 x2 -- x1 x2 x1 x2)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidAltStackOperation)
					}
					vch1 := stack.Top(-2)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidAltStackOperation)
					}
					vch2 := stack.Top(-1)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidAltStackOperation)
					}
					stack.Push(vch1)
					stack.Push(vch2)
					break
				}
			case opcodes.OP_3DUP:
				{
					// (x1 x2 x3 -- x1 x2 x3 x1 x2 x3)
					if stack.Size() < 3 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-3)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-2)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch3 := stack.Top(-1)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Push(vch1)
					stack.Push(vch2)
					stack.Push(vch3)
					break
				}
			case opcodes.OP_2OVER:
				{
					// (x1 x2 x3 x4 -- x1 x2 x3 x4 x1 x2)
					if stack.Size() < 4 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-4)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-3)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Push(vch1)
					stack.Push(vch2)
					break
				}
			case opcodes.OP_2ROT:
				{
					// (x1 x2 x3 x4 x5 x6 -- x3 x4 x5 x6 x1 x2)
					if stack.Size() < 6 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-6)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-5)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Erase(stack.Size()-6, stack.Size()-4)
					stack.Push(vch1)
					stack.Push(vch2)
					break
				}
			case opcodes.OP_2SWAP:
				{
					// (x1 x2 x3 x4 -- x3 x4 x1 x2)
					if stack.Size() < 4 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Swap(stack.Size()-4, stack.Size()-2)
					stack.Swap(stack.Size()-3, stack.Size()-1)
					break
				}
			case opcodes.OP_IFDUP:
				{
					// (x - 0 | x x)
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchBytes := vch.([]byte)
					if script.BytesToBool(vchBytes) {
						stack.Push(vch)
					}
					break
				}
			case opcodes.OP_DEPTH:
				{
					// -- stacksize
					bn := script.NewScriptNum(int64(stack.Size()))
					stack.Push(bn.Serialize())
					break
				}
			case opcodes.OP_DROP:
				{
					// (x -- )
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Pop()
					break
				}
			case opcodes.OP_DUP:
				{
					// (x -- x x)
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Push(vch)
					break
				}
			case opcodes.OP_NIP:
				{
					// (x1 x2 -- x2)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.RemoveAt(stack.Size() - 2)
					break
				}
			case opcodes.OP_OVER:
				{
					// (x1 x2 -- x1 x2 x1)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-2)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Push(vch)
					break
				}
			case opcodes.OP_PICK:
				fallthrough
			case opcodes.OP_ROLL:
				{
					// (xn ... x2 x1 x0 n - xn ... x2 x1 x0 xn)
					// (xn ... x2 x1 x0 n - ... x2 x1 x0 xn)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					//scriptNum, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					scriptNum, err := script.GetScriptNum(vch.([]byte), true, script.DefaultMaxNumSize)
					if err != nil {
						return err

					}
					n := scriptNum.ToInt32()
					stack.Pop()
					if n < 0 || n >= int32(stack.Size()) {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchn := stack.Top(int(-n - 1))
					if vchn == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					if e.OpValue == opcodes.OP_ROLL {
						stack.RemoveAt(stack.Size() - int(n) - 1)
					}
					stack.Push(vchn)
					break
				}
			case opcodes.OP_ROT:
				{
					// (x1 x2 x3 -- x2 x3 x1)
					//  x2 x1 x3  after first swap
					//  x2 x3 x1  after second swap
					if stack.Size() < 3 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Swap(stack.Size() - 3, stack.Size() - 2)
					stack.Swap(stack.Size() - 2, stack.Size() - 1)
					break
				}
			case opcodes.OP_SWAP:
				{
					// (x1 x2 -- x2 x1)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					stack.Swap(stack.Size() - 2, stack.Size() - 1)
					break
				}
			case opcodes.OP_TUCK:
				{
					// (x1 x2 -- x2 x1 x2)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					if !stack.Insert(stack.Size()-2, vch) {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					break
				}
			case opcodes.OP_SIZE:
				{
					// (in -- in size)
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					size := len(vch.([]byte))
					bn := script.NewScriptNum(int64(size))
					stack.Push(bn.Serialize())
					break
				}
				//
				// Bitwise logic
				//
			case opcodes.OP_EQUAL:
				fallthrough
			case opcodes.OP_EQUALVERIFY:
				// case opcodes.OP_NOTEQUAL: // use opcodes.OP_NUMNOTEQUAL
				{
					// (x1 x2 - bool)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-2)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-1)
					if vch2 == nil {
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
						stack.Push(bnTrue.Value)
					} else {
						stack.Push(bnFalse.Value)
					}
					if e.OpValue == opcodes.OP_EQUALVERIFY {
						if fEqual {
							stack.Pop()
						} else {
							return errcode.New(errcode.ScriptErrEqualVerify)
						}
					}
					break

				}
				//Numeric
			case opcodes.OP_1ADD:
				fallthrough
			case opcodes.OP_1SUB:
				fallthrough
			case opcodes.OP_NEGATE:
				fallthrough
			case opcodes.OP_ABS:
				fallthrough
			case opcodes.OP_NOT:
				fallthrough
			case opcodes.OP_0NOTEQUAL:
				{
					// (in -- out)
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					//bn, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					bn, err := script.GetScriptNum(vch.([]byte), true, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					switch e.OpValue {
					case opcodes.OP_1ADD:
						bn.Value += bnOne.Value
						break
					case opcodes.OP_1SUB:
						bn.Value -= bnOne.Value
						break
					case opcodes.OP_NEGATE:
						bn.Value = -bn.Value
						break
					case opcodes.OP_ABS:
						if bn.Value < 0 {
							bn.Value = -bn.Value
						}
						break
					case opcodes.OP_NOT:
						if bn.Value == bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_0NOTEQUAL:
						if bn.Value != bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					default:
						return errcode.New(errcode.ScriptErrInvalidOpCode)
					}
					stack.Pop()
					stack.Push(bn.Serialize())
					break
				}
			case opcodes.OP_ADD:
				fallthrough
			case opcodes.OP_SUB:
				fallthrough
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
				fallthrough
			case opcodes.OP_MIN:
				fallthrough
			case opcodes.OP_MAX:
				{
					// (x1 x2 -- out)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-2)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-1)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					//bn1, err := script.GetScriptNum(vch1.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					bn1, err := script.GetScriptNum(vch1.([]byte), true, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					//bn2, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					bn2, err := script.GetScriptNum(vch2.([]byte), true, script.DefaultMaxNumSize)
					if err != nil {
						return err
					}
					bn := script.NewScriptNum(0)
					switch e.OpValue {
					case opcodes.OP_ADD:
						bn.Value = bn1.Value + bn2.Value
						break
					case opcodes.OP_SUB:
						bn.Value = bn1.Value - bn2.Value
						break
					case opcodes.OP_BOOLAND:
						if bn1.Value != bnZero.Value && bn2.Value != bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_BOOLOR:
						if bn1.Value != bnZero.Value || bn2.Value != bnZero.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_NUMEQUAL:
						if bn1.Value == bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_NUMEQUALVERIFY:
						if bn1.Value == bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_NUMNOTEQUAL:
						if bn1.Value != bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_LESSTHAN:
						if bn1.Value < bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_GREATERTHAN:
						if bn1.Value > bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_LESSTHANOREQUAL:
						if bn1.Value <= bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_GREATERTHANOREQUAL:
						if bn1.Value >= bn2.Value {
							bn.Value = 1
						} else {
							bn.Value = 0
						}
						break
					case opcodes.OP_MIN:
						if bn1.Value < bn2.Value {
							bn = bn1
						} else {
							bn = bn2
						}
						break
					case opcodes.OP_MAX:
						if bn1.Value > bn2.Value {
							bn = bn1
						} else {
							bn = bn2
						}
						break
					default:
						return errcode.New(errcode.ScriptErrInvalidOpCode)
					}
					stack.Pop()
					stack.Pop()
					stack.Push(bn.Serialize())

					if e.OpValue == opcodes.OP_NUMEQUALVERIFY {
						vch := stack.Top(-1)
						if vch == nil {
							return errcode.New(errcode.ScriptErrInvalidStackOperation)
						}
						if script.BytesToBool(vch.([]byte)) {
							stack.Pop()
						} else {
							return errcode.New(errcode.ScriptErrNumEqualVerify)
						}
					}
					break
				}

			case opcodes.OP_WITHIN:
				{
					// (x min max -- out)
					if (stack.Size() < 3) {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch1 := stack.Top(-3)
					if vch1 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch2 := stack.Top(-2)
					if vch2 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch3 := stack.Top(-1)
					if vch3 == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					//bn1, err := script.GetScriptNum(vch1.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					//if err != nil {
					//	return err
					//}
					//bn2, err := script.GetScriptNum(vch2.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					//if err != nil {
					//	return err
					//}
					//bn3, err := script.GetScriptNum(vch3.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					//if err != nil {
					//	return err
					//}
					//var fValue int = 0
					//if bn2.Value <= bn1.Value && bn1.Value < bn3.Value {
					//	fValue = 1
					//} else {
					//	fValue = 0
					//}
					stack.Pop()
					stack.Pop()
					stack.Pop()
					//stack.Push(fValue)
					break;
				}
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
				{
					// (in -- txHash)
					var vchHash []byte
					if stack.Size() < 1 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vch := stack.Top(-1)
					if vch == nil {
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
						vchHash = util.Sha256Bytes(vch.([]byte))
					}
					stack.Pop()
					stack.Push(vchHash)
					break
				}
			case opcodes.OP_CODESEPARATOR:
				{
					// Hash starts after the code separator
					beginCodeHash = i
					break
				}
			case opcodes.OP_CHECKSIG:
				fallthrough
			case opcodes.OP_CHECKSIGVERIFY:
				{
					// (sig pubkey -- bool)
					if stack.Size() < 2 {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchSig := stack.Top(-2)
					if vchSig == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					vchPubkey := stack.Top(-1)
					if vchPubkey == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					vchSigBytes := vchSig.([]byte)
					checkSig, err := crypto.CheckSignatureEncoding(vchSigBytes, flags)
					if err != nil {
						return err
					}
					checkPubKey, err := crypto.CheckPubKeyEncoding(vchPubkey.([]byte), flags)
					if err != nil {
						return err
					}
					if !checkPubKey || !checkSig {
						return errcode.New(errcode.ScriptErrInValidPubKeyOrSig)
					}

					hashType := vchSigBytes[len(vchSigBytes) - 1]
					// signature is DER format, the second byte + 2 indicates the len of signature
					// 0x30 if the first byte that indicates the beginning of signature
					vchSigBytes = vchSigBytes[:vchSigBytes[1] + 2]
					// Subset of script starting at the most recent codeSeparator
					scriptCode := script.NewScriptOps(s.ParsedOpCodes[beginCodeHash:])

					// Remove the signature since there is no way for a signature
					// to sign itself.
					scriptCode = scriptCode.RemoveOpcodeByData(vchSigBytes)

					txHash, err := tx.SignatureHash(transaction, scriptCode, uint32(hashType), nIn, money, flags)
					if err != nil {
						return err
					}
					fSuccess := tx.CheckSig(txHash, vchSigBytes, vchPubkey.([]byte))
					if !fSuccess &&
						(flags & script.ScriptVerifyNullFail == script.ScriptVerifyNullFail) &&
						len(vchSig.([]byte)) > 0 {
						return errcode.New(errcode.ScriptErrSigNullFail)

					}

					stack.Pop()
					stack.Pop()
					if fSuccess {
						stack.Push(true)
					} else {
						stack.Push(false)
					}
					if e.OpValue == opcodes.OP_CHECKSIGVERIFY {
						if fSuccess {
							stack.Pop()
						} else {
							return errcode.New(errcode.ScriptErrCheckSigVerify)
						}
					}
				}
			case opcodes.OP_CHECKMULTISIG:
				fallthrough
			case opcodes.OP_CHECKMULTISIGVERIFY:
				/*{
					// ([sig ...] num_of_signatures [pubkey ...]
					// num_of_pubkeys -- bool)
					i := 1
					if stack.Size() < i {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}

					vch := stack.Top(-i)
					if vch == nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					nKeysNum, err := script.GetScriptNum(vch.([]byte), fRequireMinimal, script.DefaultMaxNumSize)
					if err != nil {
						return errcode.New(errcode.ScriptErrInvalidStackOperation)
					}
					nKeysCount := nKeysNum.Int32()
					if nKeysCount < 0 || nKeysCount > MaxOpsPerScript {
						return false, crypto.ScriptErr(crypto.ScriptErrOpCount)
					}
					i++
					iKey := i
					// ikey2 is the position of last non-signature item in
					// the stack. Top stack item = 1. With
					// ScriptVerifyNullFail, this is used for cleanup if
					// operation fails.
					iKey2 := nKeysCount + 2
					i += int(nKeysCount)
					if stack.Size() < i {
						return false, crypto.ScriptErr(crypto.ScriptErrInvalidStackOperation)
					}
					sigsVch, err := stack.Top(-i)
					if err != nil {
						return false, err
					}
					nSigsNum, err := GetCScriptNum(sigsVch.([]byte), fRequireMinimal, DefaultMaxNumSize)
					if err != nil {
						return false, err
					}
					nSigsCount := nSigsNum.Int32()
					if nSigsCount < 0 || nSigsCount > nKeysCount {
						return false, crypto.ScriptErr(crypto.ScriptErrSigCount)
					}
					i++
					isig := i
					i += int(nSigsCount)
					if stack.Size() < i {
						return false, crypto.ScriptErr(crypto.ScriptErrInvalidStackOperation)
					}

					// Subset of script starting at the most recent
					// codeseparator
					scriptCode := NewScriptRaw(s.GetByteCodes()[pbegincodehash:])

					// Drop the signature in pre-segwit scripts but not
					// segwit scripts
					for k := 0; k < int(nSigsCount); k++ {
						vchSig, err := stack.Top(-isig - k)
						if err != nil {
							return false, err
						}
						CleanupScriptCode(scriptCode, vchSig.([]byte), flags)
					}
					fSuccess := true
					for fSuccess && nSigsCount > 0 {
						vchSig, err := stack.Top(-isig)
						if err != nil {
							return false, err
						}
						vchPubkey, err := stack.Top(-iKey)
						if err != nil {
							return false, err
						}
						// Note how this makes the exact order of
						// pubkey/signature evaluation distinguishable by
						// CHECKMULTISIG NOT if the STRICTENC flag is set.
						// See the script_(in)valid tests for details.
						checkSig, err := crypto.CheckSignatureEncoding(vchSig.([]byte), flags)
						if err != nil {
							return false, err
						}
						checkPubKey, err := crypto.CheckPubKeyEncoding(vchPubkey.([]byte), flags)
						if err != nil {
							return false, err
						}
						if !checkSig || !checkPubKey {
							return false, errors.New("check sig or public key failed")
						}
						txHash, err := SignatureHash(tx, scriptCode, flags, nIn)
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
						vch, err := stack.Top(-isig)
						if err != nil {
							return false, err
						}
						if !fSuccess &&
							(flags&crypto.ScriptVerifyNullFail == crypto.ScriptVerifyNullFail) &&
							iKey2 == 0 && len(vch.([]byte)) > 0 {
							return false, crypto.ScriptErr(crypto.ScriptErrSigNullFail)

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
					if stack.Size() > 1 {
						return false, crypto.ScriptErr(crypto.ScriptErrInvalidStackOperation)
					}
					vch, err = stack.Top(-isig)
					if err != nil {
						return false, err
					}
					if flags&crypto.ScriptVerifyNullFail == crypto.ScriptVerifyNullFail && len(vch.([]byte)) > 0 {
						return false, err
					}
					stack.Pop()
					if fSuccess {
						stack.Push(vchTrue)
					} else {
						stack.Push(vchFalse)
					}
					if e.OpValue == opcodes.OP_CHECKMULTISIGVERIFY {
						if fSuccess {
							stack.Pop()
						} else {
							return nil
						}
					}
					if stack.Size()+altstack.Size() > 1000 {
						return nil
					}
				}*/
			}
		}
	}
	return nil
}

func checkLockTime(lockTime int64, txLockTime int64, sequence uint32) bool {
	// There are two kinds of nLockTime: lock-by-blockheight and
	// lock-by-blocktime, distinguished by whether nLockTime <
	// LOCKTIME_THRESHOLD.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nLockTime being tested is the same as the nLockTime in the
	// transaction.
	if !((txLockTime < script.LockTimeThreshold && lockTime < script.LockTimeThreshold) ||
		(txLockTime >= script.LockTimeThreshold && lockTime >= script.LockTimeThreshold)) {
		return false
	}

	// Now that we know we're comparing apples-to-apples, the comparison is a
	// simple numeric one.
	if lockTime > txLockTime {
		return false
	}
	// Finally the nLockTime feature can be disabled and thus
	// checkLockTimeVerify bypassed if every txIN has been finalized by setting
	// nSequence to maxInt. The transaction would be allowed into the
	// blockChain, making the opCode ineffective.
	//
	// Testing if this vin is not final is sufficient to prevent this condition.
	// Alternatively we could test all inputs, but testing just this input
	// minimizes the data required to prove correct checkLockTimeVerify
	// execution.
	if script.SequenceFinal == sequence {
		return false
	}
	return true
}

func checkSequence(sequence int64, txToSequence int64, txVersion uint32) bool {
	// Fail if the transaction's version number is not set high enough to
	// trigger BIP 68 rules.
	if txVersion < 2 {
		return false
	}
	// Sequence numbers with their most significant bit set are not consensus
	// constrained. Testing that the transaction's sequence number do not have
	// this bit set prevents using this property to get around a
	// checkSequenceVerify check.
	if txToSequence & script.SequenceLockTimeDisableFlag == script.SequenceLockTimeDisableFlag {
		return false
	}
	// Mask off any bits that do not have consensus-enforced meaning before
	// doing the integer comparisons
	nLockTimeMask := script.SequenceLockTimeTypeFlag | script.SequenceLockTimeMask
	txToSequenceMasked := txToSequence & int64(nLockTimeMask)
	nSequenceMasked := sequence & int64(nLockTimeMask)

	// There are two kinds of nSequence: lock-by-blockHeight and
	// lock-by-blockTime, distinguished by whether nSequenceMasked <
	// CTxIn::SEQUENCE_LOCKTIME_TYPE_FLAG.
	//
	// We want to compare apples to apples, so fail the script unless the type
	// of nSequenceMasked being tested is the same as the nSequenceMasked in the
	// transaction.
	if !((txToSequenceMasked < script.SequenceLockTimeTypeFlag && nSequenceMasked < script.SequenceLockTimeTypeFlag) ||
		(txToSequenceMasked >= script.SequenceLockTimeTypeFlag && nSequenceMasked >= script.SequenceLockTimeTypeFlag)) {
		return false
	}
	if nSequenceMasked > txToSequenceMasked {
		return false
	}
	return true
}
