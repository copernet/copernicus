package core

import (
	"encoding/binary"
	"errors"
	"bytes"
	"io"
	"copernicus/utils"
)

const (
	DefaultSize = 28

	// MaxPubKeysPerMultiSig :  maximum number of public keys per multiSig
	MaxPubKeysPerMultiSig = 20

	// LockTimeThreshold threshold for nLockTime: below this value it is interpreted as block number,
	// otherwise as UNIX timestamp. Threshold is Tue Nov 5 00:53:20 1985 UTC
	LockTimeThreshold = 500000000

	// SequenceFinal setting sequence to this value for every input in a transaction
	// disables nLockTime.
	SequenceFinal = 0xffffffff

	MaxScriptSize        = 10000
	MaxScriptElementSize = 520
	MaxScriptOpCodes     = 201
	MaxOpsPerScript      = 201
)


/** Script verification flags */
const (
	ScriptVerifyNone = 0

	// Evaluate P2SH subscripts (softfork safe, BIP16).
	ScriptVerifyP2SH = (1 << 0)

	// Passing a non-strict-DER signature or one with undefined hashtype to a
	// checksig operation causes script failure. Evaluating a pubkey that is not
	// (0x04 + 64 bytes) or (0x02 or 0x03 + 32 bytes) by checksig causes script
	// failure.
	ScriptVerifyStrictEnc = (1 << 1)

	// Passing a non-strict-DER signature to a checksig operation causes script
	// failure (softfork safe, BIP62 rule 1)
	ScriptVerifyDersig = (1 << 2)

	// Passing a non-strict-DER signature or one with S > order/2 to a checksig
	// operation causes script failure
	// (softfork safe, BIP62 rule 5).
	ScriptVerifyLowS = (1 << 3)

	// verify dummy stack item consumed by CHECKMULTISIG is of zero-length
	// (softfork safe, BIP62 rule 7).
	ScriptVerifyNullDummy = (1 << 4)

	// Using a non-push operator in the scriptSig causes script failure
	// (softfork safe, BIP62 rule 2).
	ScriptVerifySigPushOnly = (1 << 5)

	// Require minimal encodings for all push operations (OP_0... OP_16,
	// OP_1NEGATE where possible, direct pushes up to 75 bytes, OP_PUSHDATA up
	// to 255 bytes, OP_PUSHDATA2 for anything larger). Evaluating any other
	// push causes the script to fail (BIP62 rule 3). In addition, whenever a
	// stack element is interpreted as a number, it must be of minimal length
	// (BIP62 rule 4).
	// (softfork safe)
	ScriptVerifyMinmalData = (1 << 6)

	// Discourage use of NOPs reserved for upgrades (NOP1-10)
	//
	// Provided so that nodes can avoid accepting or mining transactions
	// containing executed NOP's whose meaning may change after a soft-fork,
	// thus rendering the script invalid; with this flag set executing
	// discouraged NOPs fails the script. This verification flag will never be a
	// mandatory flag applied to scripts in a block. NOPs that are not executed,
	// e.g.  within an unexecuted IF ENDIF block, are *not* rejected.
	ScriptVerifyDiscourageUpgradableNops = (1 << 7)

	// Require that only a single stack element remains after evaluation. This
	// changes the success criterion from "At least one stack element must
	// remain, and when interpreted as a boolean, it must be true" to "Exactly
	// one stack element must remain, and when interpreted as a boolean, it must
	// be true".
	// (softfork safe, BIP62 rule 6)
	// Note: CLEANSTACK should never be used without P2SH or WITNESS.
	ScriptVerifyCleanStack = (1 << 8)

	// Verify CHECKLOCKTIMEVERIFY
	//
	// See BIP65 for details.
	ScriptVerifyCheckLockTimeVerify = (1 << 9)

	// support CHECKSEQUENCEVERIFY opcode
	//
	// See BIP112 for details
	ScriptVerifyCheckSequenceVerify = (1 << 10)

	// Making v1-v16 witness program non-standard
	//
	ScriptVerifyDiscourageUpgradableWitnessProgram = (1 << 12)

	// Segwit script only: Require the argument of OP_IF/NOTIF to be exactly
	// 0x01 or empty vector
	//
	ScriptVerifyMinmalIf = (1 << 13)

	// Signature(s) must be empty vector if an CHECK(MULTI)SIG operation failed
	//
	ScriptVerifyNullFail = (1 << 14)

	// Public keys in scripts must be compressed
	//
	ScriptVerifyCompressedPubkeyType = (1 << 15)

	// Do we accept signature using SIGHASH_FORKID
	//
	ScriptEnableSighashForkid = (1 << 16)

	// Do we accept activate replay protection using a different fork id.
	//
	ScriptEnableReplayProtection = (1 << 17)

	// Enable new opcodes.
	//
	ScriptEnableMonolithOpcodes = (1 << 18)
)

const (
	ScriptNonStandard = iota
	// 'standard' transaction types:
	ScriptPubkey
	ScriptPubkeyHash
	ScriptHash
	ScriptMultiSig
	ScriptNullData

	MaxOpReturnRelay uint = 83
	MaxOpReturnRelayLarge uint = 223
)

type Script struct {
	byteCodes         []byte
	ParsedOpCodes []ParsedOpCode
}

func (s *Script)SetByteCodes(bc []byte){
	s.byteCodes = bc
	
}
func (s *Script)GetByteCodes() []byte{
	return s.byteCodes
}
func NewScriptRaw(bytes []byte) *Script {
	script := Script{bytes: bytes}
	script.convertOPS()
	return &script
}

func NewScriptOps(parsedOpCodes []ParsedOpCode) *Script {
	script := Script{ParsedOpCodes: parsedOpCodes}
	script.convertRaw()
	return &script
}

func (script *Script) convertRaw() {
	script.bytes = make([]byte, 0)
	for _, e := range script.ParsedOpCodes {
		script.bytes = append(script.bytes, e.OpValue)
		if e.OpValue == OP_PUSHDATA1 {
			script.bytes = append(script.bytes, byte(e.Length))
		} else if e.OpValue == OP_PUSHDATA2 {
			b := make([]byte, 2)
			binary.LittleEndian.PutUint16(b, uint16(e.Length))
			script.bytes = append(script.bytes, b...)
		} else if e.OpValue == OP_PUSHDATA4 {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, uint32(e.Length))
			script.bytes = append(script.bytes, b...)
		} else {
			if e.OpValue < OP_PUSHDATA1 && e.Length > 0 {
				script.bytes = append(script.bytes, byte(e.Length))
				script.bytes = append(script.bytes, e.Data...)
			}
		}

	}

}

func (script *Script) convertOPS() error {
	script.ParsedOpCodes = make([]ParsedOpCode, 0)
	scriptLen := len(script.bytes)

	for i := 0; i < scriptLen; i++ {
		var nSize int
		opcode := script.bytes[i]
		parsedopCode := ParsedOpCode{OpValue: opcode}

		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
			if scriptLen - i < nSize {
				return errors.New("OP has no enough data")
			}
			parsedopCode.Data = script.bytes[i + 1: i + 1 + nSize]
		} else if opcode == OP_PUSHDATA1 {
			if scriptLen - i < 1 {
				return errors.New("OP_PUSHDATA1 has no enough data")
			}

			nSize = int(script.bytes[i + 1])
			if scriptLen - i - 1 < nSize {
				return errors.New("OP_PUSHDATA1 has no enough data")
			}
			parsedopCode.Data = script.bytes[i + 2: i + 2 + nSize]
			i++
		} else if opcode == OP_PUSHDATA2 {
			if scriptLen - i < 2 {
				return errors.New("OP_PUSHDATA2 has no enough data")
			}
			nSize = int(binary.LittleEndian.Uint16(script.bytes[i + 1: i + 3]))
			if scriptLen - i - 3 < nSize {
				return errors.New("OP_PUSHDATA2 has no enough data")
			}
			parsedopCode.Data = script.bytes[i + 3: i + 3 + nSize]
			i += 2
		} else if opcode == OP_PUSHDATA4 {
			if scriptLen - i < 4 {
				return errors.New("OP_PUSHDATA4 has no enough data")

			}
			nSize = int(binary.LittleEndian.Uint32(script.bytes[i + 1: i + 5]))
			parsedopCode.Data = script.bytes[i + 5: i + 5 + nSize]
			i += 4
		}
		if scriptLen - i < 0 || (scriptLen - i) < nSize {
			return errors.New("size is wrong")

		}
		parsedopCode.Length = nSize

		script.ParsedOpCodes = append(script.ParsedOpCodes, parsedopCode)

		i += nSize
	}
	return nil
}

func (script *Script) Eval() (int, error) {
	return 0, nil
}

func ReadScript(reader io.Reader, maxAllowed uint32, fieldName string) (signScript []byte, err error) {
	count, err := utils.ReadVarInt(reader)
	if err != nil {
		return
	}
	if count > uint64(maxAllowed) {
		err = errors.Errorf("readScript %s is larger than the max allowed size [count %d,max %d]", fieldName, count, maxAllowed)
		return
	}
	buf := scriptPool.Borrow(count)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		scriptPool.Return(buf)
		return
	}
	return buf, nil

}

/*
func (script *Script) IsCommitment(data []byte) bool {
	if len(data) > 64 || script.Size() != len(data)+2 {
		return false
	}

	if script.byteCodes[0] != OP_RETURN || int(script.byteCodes[1]) != len(data) {
		return false
	}

	for i := 0; i < len(data); i++ {
		if script.byteCodes[i+2] != data[i] {
			return false
		}
	}

	return true
}
*/

func (script *Script) CheckScriptPubKey(state *ValidationState) (succeed bool, pubKeyType int) {
	//p2sh scriptPubKey
	if script.IsPayToScriptHash() {
		return true, ScriptHash
	}
	// Provably prunable, data-carrying output
	//
	// So long as script passes the IsUnspendable() test and all but the first
	// byte passes the IsPushOnly() test we don't care what exactly is in the
	// script.
	len := len(script.ParsedOpCodes)
	if len == 0 {
		return false, ScriptNonStandard
	}
	parsedOpCode0 := script.ParsedOpCodes[0]
	opValue0 := parsedOpCode0.OpValue

	// OP_RETURN
	if len == 1 {
		if parsedOpCode0.OpValue == OP_RETURN {
			return true, ScriptNullData
		}
		return false, ScriptNonStandard
	}

	// OP_RETURN and DATA
	if parsedOpCode0.OpValue == OP_RETURN {
		tempScript := NewScriptOps(script.ParsedOpCodes[1:])
		if tempScript.IsPushOnly() {
			return true, ScriptNullData
		}
		return false, ScriptNonStandard
	}

	//PUBKEY OP_CHECKSIG
	if len == 2 {
		if opValue0 > OP_PUSHDATA4 || parsedOpCode0.Length < 33 ||
			parsedOpCode0.Length > 65 || script.ParsedOpCodes[1].OpValue != OP_CHECKSIG {
			return false, ScriptNonStandard
		}
		return true, ScriptPubkey
	}

	//OP_DUP OP_HASH160 OP_PUBKEYHASH OP_EQUALVERIFY OP_CHECKSIG
	if opValue0 == OP_DUP {
		if script.ParsedOpCodes[1].OpValue != OP_HASH160 ||
			script.ParsedOpCodes[2].OpValue != OP_PUBKEYHASH ||
			script.ParsedOpCodes[2].Length != 20 ||
			script.ParsedOpCodes[3].OpValue != OP_EQUALVERIFY ||
			script.ParsedOpCodes[4].OpValue != OP_CHECKSIG {
			return false, ScriptNonStandard
		}
		return true, ScriptPubkeyHash
	}

	//m pubkey1 pubkey2...pubkeyn n OP_CHECKMULTISIG
	if opValue0 == OP_0 || (opValue0 >= OP_1 && opValue0 <= OP_16) {
		opM, _ := DecodeOPN(opValue0)
		i := 1
		pubKeyCount := 0
		for script.ParsedOpCodes[i].Length >= 33 && script.ParsedOpCodes[i].Length <= 65 {
			pubKeyCount++
			i++
		}
		opValueI := script.ParsedOpCodes[i].OpValue
		if opValueI == OP_0 || (opValue0 >= OP_1 && opValue0 <= OP_16) {
			opN, _ := DecodeOPN(opValueI)
			// Support up to x-of-3 multisig txns as standard
			if opM < 1 || opN < 1 || opN > 3 || opM > opN || opN != pubKeyCount {
				return false, ScriptNonStandard
			}
			i++
		} else {
			return false, ScriptNonStandard
		}
		if script.ParsedOpCodes[i].OpValue != OP_CHECKMULTISIG {
			return false, ScriptNonStandard
		}
		return true, ScriptMultiSig
	}

	return false, ScriptNonStandard
}

func (script *Script) CheckScriptSig(state *ValidationState) bool{
	if script.Size() > 1650 {
		state.Dos(100, false, RejectInvalid, "bad-tx-input-script-size", false, "")
		return false
	}
	if !script.IsPushOnly() {
		state.Dos(100, false, RejectInvalid, "bad-tx-input-script-not-pushonly", false, "")
		return false
	}

	return true
}

func (script *Script) IsPayToScriptHash() bool {
	size := len(script.byteCodes)
	return size == 23 &&
		script.byteCodes[0] == OP_HASH160 &&
		script.byteCodes[1] == 0x14 &&
		script.byteCodes[22] == OP_EQUAL
}

func (script *Script) IsUnspendable() bool {
	return script.Size() > 0 &&
		script.ParsedOpCodes[0].OpValue == OP_RETURN ||
		script.Size() > MaxScriptSize
}
/*
func CheckMinimalPush(data []byte, opcode int32) bool {
	dataLen := len(data)
	if dataLen == 0 {
		// Could have used OP_0.
		return opcode == OP_0
	}
	if dataLen == 1 && data[0] >= 1 && data[0] <= 16 {
		// Could have used OP_1 .. OP_16.
		return opcode == (OP_1 + int32(data[0]-1))
	}
	if dataLen == 1 && data[0] == 0x81 {
		return opcode == OP_1NEGATE
	}
	if dataLen <= 75 {
		// Could have used a direct push (opcode indicating number of byteCodes
		// pushed + those byteCodes).
		return opcode == int32(dataLen)
	}
	if dataLen <= 255 {
		// Could have used OP_PUSHDATA.
		return opcode == OP_PUSHDATA1
	}
	if dataLen <= 65535 {
		// Could have used OP_PUSHDATA2.
		return opcode == OP_PUSHDATA2
	}
	return true
}
*/
/*
func (script *Script) GetOp(index *int, opCode *byte, data *[]byte) bool {

	opcode := byte(OP_INVALIDOPCODE)
	tmpIndex := *index
	tmpData := make([]byte, 0)
	if tmpIndex >= script.Size() {
		return false
	}

	// Read instruction
	if script.Size() - tmpIndex < 1 {
		return false
	}

	opcode = script.byteCodes[tmpIndex]
	tmpIndex++

	// Immediate operand
	if opcode <= OP_PUSHDATA4 {
		nSize := 0
		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
		} else if opcode == OP_PUSHDATA1 {
			if script.Size() - tmpIndex < 1 {
				return false
			}
			nSize = int(script.byteCodes[*index])
			tmpIndex++
		} else if opcode == OP_PUSHDATA2 {
			if script.Size() - tmpIndex < 2 {
				return false
			}
			nSize = int(binary.LittleEndian.Uint16(script.byteCodes[tmpIndex : tmpIndex+2]))
			tmpIndex += 2
		} else if opcode == OP_PUSHDATA4 {
			if script.Size() - tmpIndex < 4 {
				return false
			}
			nSize = int(binary.LittleEndian.Uint32(script.byteCodes[tmpIndex : tmpIndex+4]))
			tmpIndex += 4
		}
		if script.Size()-tmpIndex < 0 || script.Size()-tmpIndex < nSize {
			return false
		}
		tmpData = append(tmpData, script.byteCodes[tmpIndex:tmpIndex+nSize]...)
		tmpIndex += nSize
	}

	*data = tmpData
	*opCode = opcode
	*index = tmpIndex
	return true
}*/

/*
func (script *Script) PushInt64(n int64) {

	if n == -1 || (n >= 1 && n <= 16) {
		script.byteCodes = append(script.byteCodes, byte(n+(OP_1-1)))
	} else if n == 0 {
		script.byteCodes = append(script.byteCodes, byte(OP_0))
	} else {
		scriptNum := NewCScriptNum(n)
		script.byteCodes = append(script.byteCodes, scriptNum.Serialize()...)
	}
}

func (script *Script) PushOpCode(opcode int) error {
	if opcode < 0 || opcode > 0xff {
		return errors.New("push opcode failed :invalid opcode")
	}
	script.byteCodes = append(script.byteCodes, byte(opcode))
	return nil
}

func (script *Script) PushScriptNum(scriptNum *CScriptNum) {
	script.byteCodes = append(script.byteCodes, scriptNum.Serialize()...)
}

func (script *Script) PushData(data []byte) {
	dataLen := len(data)
	if dataLen < OP_PUSHDATA1 {
		script.byteCodes = append(script.byteCodes, byte(dataLen))
	} else if dataLen <= 0xff {
		script.byteCodes = append(script.byteCodes, OP_PUSHDATA1)
		script.byteCodes = append(script.byteCodes, byte(dataLen))
	} else if dataLen <= 0xffff {
		script.byteCodes = append(script.byteCodes, OP_PUSHDATA2)
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(dataLen))
		script.byteCodes = append(script.byteCodes, buf...)

	} else {
		script.byteCodes = append(script.byteCodes, OP_PUSHDATA4)
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(script.byteCodes, uint32(dataLen))
		script.byteCodes = append(script.byteCodes, buf...)
	}
	script.byteCodes = append(script.byteCodes, data...)
}
*/
/*
func (script *Script) ParseScript() (stk []ParsedOpCode, err error) {
	stk = make([]ParsedOpCode, 0)
	scriptLen := len(script.byteCodes)

	for i := 0; i < scriptLen; i++ {
		var nSize int
		opcode := script.byteCodes[i]
		parsedopCode := ParsedOpCode{opValue: opcode}

		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
			parsedopCode.data = script.byteCodes[i+1 : i+1+nSize]

		} else if opcode == OP_PUSHDATA1 {
			if scriptLen-i < 1 {
				err = errors.New("OP_PUSHDATA1 has no enough data")
				return
			}
			nSize = int(script.byteCodes[i+1])
			parsedopCode.data = script.byteCodes[i+2 : i+2+nSize]
			i++

		} else if opcode == OP_PUSHDATA2 {
			if scriptLen-i < 2 {
				err = errors.New("OP_PUSHDATA2 has no enough data")
				return
			}
			nSize = int(binary.LittleEndian.Uint16(script.byteCodes[i+1 : i+3]))
			parsedopCode.data = script.byteCodes[i+3 : i+3+nSize]
			i += 2
		} else if opcode == OP_PUSHDATA4 {
			if scriptLen-i < 4 {
				err = errors.New("OP_PUSHDATA4 has no enough data")
				return
			}
			nSize = int(binary.LittleEndian.Uint32(script.byteCodes[i+1 : i+5]))
			parsedopCode.data = script.byteCodes[i+5 : i+5+nSize]
			i += 4
		}
		if scriptLen-i < 0 || (scriptLen-i) < nSize {
			err = errors.New("size is wrong")
			return
		}

		stk = append(stk, parsedopCode)
		i += nSize
	}

	return
}
*/
/*
func (script *Script) FindAndDelete(b *Script) (bool, error) {
	//orginalParseCodes, err := script.ParseScript()
	//if err != nil {
	//	return false, err
	//}
	//paramScript, err := b.ParseScript()
	//if err != nil {
	//	return false, err
	//}
	//script.bytes = make([]byte, 0)

	for i := 0; i < len(orginalParseCodes); i++ {
		isDelete := false
		parseCode := orginalParseCodes[i]
		for j := 0; j < len(paramScript); j++ {
			parseCodeOther := paramScript[j]
			if parseCode.opValue == parseCodeOther.opValue {
				isDelete = true
			}
		}
		if !isDelete {
			script.byteCodes = append(script.byteCodes, parseCode.opValue)
			script.byteCodes = append(script.byteCodes, parseCode.data...)
		}
	}

	return true, nil
}
*/
/*
func (script *Script) Find(opcode int) bool {
	//stk, err := script.ParseScript()
	//if err != nil {
	//	return false
	//}
	for _, ops := range script.ParsedOpCodes {
		if int(ops.opValue) == opcode {
			return true
		}
	}
	return false
}
*/

func (script *Script) IsPushOnly() bool {
	for _, ops := range script.ParsedOpCodes {
		if ops.OpValue > OP_16 {
			return false
		}
	}
	return true

}
/*
func (script *Script) GetSigOpCount() (int, error) {
	if !script.IsPayToScriptHash() {
		return script.GetSigOpCountWithAccurate(true)
	}
	stk, err := script.ParseScript()
	if err != nil {
		return 0, err
	}
	if len(stk) == 0 {
		return 0, nil
	}
	for i := 0; i < len(stk); i++ {
		opcode := stk[i].opValue
		if opcode == OP_16 {
			return 0, nil
		}
	}
	return script.GetSigOpCountWithAccurate(true)
}

func (script *Script) GetSigOpCountFor(scriptSig *Script) (int, error) {
	if !script.IsPayToScriptHash() {
		return script.GetSigOpCountWithAccurate(true)
	}

	// This is a pay-to-script-hash scriptPubKey;
	// get the last item that the scriptSig
	// pushes onto the stack:
	var n = 0
	stk, err := scriptSig.ParseScript()
	if err != nil {
		return n, err
	}

	data := make([]byte, 0)
	for i := 0; i < len(stk); i++ {
		var opcode *byte
		if !scriptSig.GetOp(&i, opcode, &data) {
			return 0, nil
		}

		if *opcode > OP_16 {
			return 0, nil
		}
	}

	subScript := NewScriptRaw(data)
	return subScript.GetSigOpCountWithAccurate(true)
}
*/
/*
func (script *Script) GetScriptByte() []byte {
	scriptByte := make([]byte, 0)
	scriptByte = append(scriptByte, script.byteCodes...)
	return scriptByte
}
*/
func (script *Script) GetSigOpCount(accurate bool) (int, error) {
	n := 0
	//stk, err := script.ParseScript()
	//if err != nil {
	//	return n, err
	//}
	var lastOpcode byte
	for _, e := range script.ParsedOpCodes {
		opcode := e.OpValue
		if opcode == OP_CHECKSIG || opcode == OP_CHECKSIGVERIFY {
			n++
		} else if opcode == OP_CHECKMULTISIG || opcode == OP_CHECKMULTISIGVERIFY {
			if accurate && lastOpcode >= OP_1 && lastOpcode <= OP_16 {
				opn, err := DecodeOPN(lastOpcode)
				if err != nil {
					return 0, err

				}
				n += opn
			} else {
				n += MaxPubKeysPerMultiSig
			}
		}
		lastOpcode = opcode
	}
	return n, nil
}

func (script *Script) GetP2SHSigOpCount() (int, error) {
	// This is a pay-to-script-hash scriptPubKey;
	// get the last item that the scriptSig
	// pushes onto the stack:
	for _, e := range script.ParsedOpCodes {
		opcode := e.opValue
		if opcode > OP_16 {
			return 0, nil
		}
	}
	lastOps := script.ParsedOpCodes[len(script.ParsedOpCodes) - 1]
	tempScript := NewScriptRaw(lastOps.data)
	return tempScript.GetSigOpCount(true)

}


func EncodeOPN(n int) (int, error) {
	if n < 0 || n > 16 {
		return 0, errors.New("EncodeOPN n is out of bounds")
	}
	return OP_1 + n - 1, nil
}

func DecodeOPN(opcode byte) (int, error) {
	if opcode < OP_0 || opcode > OP_16 {
		return 0, errors.New(" DecodeOPN opcode is out of bounds")
	}
	return int(opcode) - int(OP_1 - 1), nil
}

func (script *Script) Size() int {
	return len(script.byteCodes)
}

func (script *Script) IsEqual(script2 *Script) bool {
	/*if script.Size() != script2.Size() {
		return false
	}*/

	return bytes.Equal(script.byteCodes, script2.byteCodes)
}
