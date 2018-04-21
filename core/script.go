package core

import (
	"bytes"
	"encoding/binary"
	"errors"
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
	SCRIPT_VERIFY_NONE = 0

	// Evaluate P2SH subscripts (softfork safe, BIP16).
	SCRIPT_VERIFY_P2SH = (1 << 0)

	// Passing a non-strict-DER signature or one with undefined hashtype to a
	// checksig operation causes script failure. Evaluating a pubkey that is not
	// (0x04 + 64 bytes) or (0x02 or 0x03 + 32 bytes) by checksig causes script
	// failure.
	SCRIPT_VERIFY_STRICTENC = (1 << 1)

	// Passing a non-strict-DER signature to a checksig operation causes script
	// failure (softfork safe, BIP62 rule 1)
	SCRIPT_VERIFY_DERSIG = (1 << 2)

	// Passing a non-strict-DER signature or one with S > order/2 to a checksig
	// operation causes script failure
	// (softfork safe, BIP62 rule 5).
	SCRIPT_VERIFY_LOW_S = (1 << 3)

	// verify dummy stack item consumed by CHECKMULTISIG is of zero-length
	// (softfork safe, BIP62 rule 7).
	SCRIPT_VERIFY_NULLDUMMY = (1 << 4)

	// Using a non-push operator in the scriptSig causes script failure
	// (softfork safe, BIP62 rule 2).
	SCRIPT_VERIFY_SIGPUSHONLY = (1 << 5)

	// Require minimal encodings for all push operations (OP_0... OP_16,
	// OP_1NEGATE where possible, direct pushes up to 75 bytes, OP_PUSHDATA up
	// to 255 bytes, OP_PUSHDATA2 for anything larger). Evaluating any other
	// push causes the script to fail (BIP62 rule 3). In addition, whenever a
	// stack element is interpreted as a number, it must be of minimal length
	// (BIP62 rule 4).
	// (softfork safe)
	SCRIPT_VERIFY_MINIMALDATA = (1 << 6)

	// Discourage use of NOPs reserved for upgrades (NOP1-10)
	//
	// Provided so that nodes can avoid accepting or mining transactions
	// containing executed NOP's whose meaning may change after a soft-fork,
	// thus rendering the script invalid; with this flag set executing
	// discouraged NOPs fails the script. This verification flag will never be a
	// mandatory flag applied to scripts in a block. NOPs that are not executed,
	// e.g.  within an unexecuted IF ENDIF block, are *not* rejected.
	SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_NOPS = (1 << 7)

	// Require that only a single stack element remains after evaluation. This
	// changes the success criterion from "At least one stack element must
	// remain, and when interpreted as a boolean, it must be true" to "Exactly
	// one stack element must remain, and when interpreted as a boolean, it must
	// be true".
	// (softfork safe, BIP62 rule 6)
	// Note: CLEANSTACK should never be used without P2SH or WITNESS.
	SCRIPT_VERIFY_CLEANSTACK = (1 << 8)

	// Verify CHECKLOCKTIMEVERIFY
	//
	// See BIP65 for details.
	SCRIPT_VERIFY_CHECKLOCKTIMEVERIFY = (1 << 9)

	// support CHECKSEQUENCEVERIFY opcode
	//
	// See BIP112 for details
	SCRIPT_VERIFY_CHECKSEQUENCEVERIFY = (1 << 10)

	// Making v1-v16 witness program non-standard
	//
	SCRIPT_VERIFY_DISCOURAGE_UPGRADABLE_WITNESS_PROGRAM = (1 << 12)

	// Segwit script only: Require the argument of OP_IF/NOTIF to be exactly
	// 0x01 or empty vector
	//
	SCRIPT_VERIFY_MINIMALIF = (1 << 13)

	// Signature(s) must be empty vector if an CHECK(MULTI)SIG operation failed
	//
	SCRIPT_VERIFY_NULLFAIL = (1 << 14)

	// Public keys in scripts must be compressed
	//
	SCRIPT_VERIFY_COMPRESSED_PUBKEYTYPE = (1 << 15)

	// Do we accept signature using SIGHASH_FORKID
	//
	SCRIPT_ENABLE_SIGHASH_FORKID = (1 << 16)

	// Do we accept activate replay protection using a different fork id.
	//
	SCRIPT_ENABLE_REPLAY_PROTECTION = (1 << 17)

	// Enable new opcodes.
	//
	SCRIPT_ENABLE_MONOLITH_OPCODES = (1 << 18)
)

type Script struct {
	bytes         []byte
	ParsedOpCodes []ParsedOpCode
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
		script.bytes = append(script.bytes, e.opValue)
		script.bytes = append(script.bytes, e.data...)
	}

}

func (script *Script) convertOPS() error {
	/*stk, err := script.ParseScript()
	if err != nil {
		return err
	}
	script.ParsedOpCodes = stk
	return nil*/
	script.ParsedOpCodes = make([]ParsedOpCode, 0)
	scriptLen := len(script.bytes)

	for i := 0; i < scriptLen; i++ {
		var nSize int
		opcode := script.bytes[i]
		parsedopCode := ParsedOpCode{opValue: opcode}

		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
			if scriptLen - i < nSize {
				return errors.New("OP has no enough data")
			}
			parsedopCode.data = script.bytes[i + 1: i + 1 + nSize]
		} else if opcode == OP_PUSHDATA1 {
			if scriptLen - i < 1 {
				return errors.New("OP_PUSHDATA1 has no enough data")
			}

			nSize = int(script.bytes[i + 1])
			if scriptLen - i - 1 < nSize {
				return errors.New("OP_PUSHDATA1 has no enough data")
			}
			parsedopCode.data = script.bytes[i + 2: i + 2 + nSize]
			i++
		} else if opcode == OP_PUSHDATA2 {
			if scriptLen - i < 2 {
				return errors.New("OP_PUSHDATA2 has no enough data")
			}
			nSize = int(binary.LittleEndian.Uint16(script.bytes[i + 1: i + 3]))
			if scriptLen - i - 3 < nSize {
				return errors.New("OP_PUSHDATA2 has no enough data")
			}
			parsedopCode.data = script.bytes[i + 3: i + 3 + nSize]
			i += 2
		} else if opcode == OP_PUSHDATA4 {
			if scriptLen - i < 4 {
				return errors.New("OP_PUSHDATA4 has no enough data")

			}
			nSize = int(binary.LittleEndian.Uint32(script.bytes[i + 1: i + 5]))
			parsedopCode.data = script.bytes[i + 5: i + 5 + nSize]
			i += 4
		}
		if scriptLen - i < 0 || (scriptLen - i) < nSize {
			return errors.New("size is wrong")

		}

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

	if script.bytes[0] != OP_RETURN || int(script.bytes[1]) != len(data) {
		return false
	}

	for i := 0; i < len(data); i++ {
		if script.bytes[i+2] != data[i] {
			return false
		}
	}

	return true
}
*/

func (script *Script) Check() bool {
	return false
}

func (script *Script) IsPayToScriptHash() bool {
	size := len(script.bytes)
	return size == 23 &&
		script.bytes[0] == OP_HASH160 &&
		script.bytes[1] == 0x14 &&
		script.bytes[22] == OP_EQUAL
}

func (script *Script) IsUnspendable() bool {
	//err := script.ConvertOPS()
	//if err != nil {
	//	return false
	//}

	return script.Size() > 0 &&
		script.ParsedOpCodes[0].opValue == OP_RETURN ||
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
		// Could have used a direct push (opcode indicating number of bytes
		// pushed + those bytes).
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

	opcode = script.bytes[tmpIndex]
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
			nSize = int(script.bytes[*index])
			tmpIndex++
		} else if opcode == OP_PUSHDATA2 {
			if script.Size() - tmpIndex < 2 {
				return false
			}
			nSize = int(binary.LittleEndian.Uint16(script.bytes[tmpIndex : tmpIndex+2]))
			tmpIndex += 2
		} else if opcode == OP_PUSHDATA4 {
			if script.Size() - tmpIndex < 4 {
				return false
			}
			nSize = int(binary.LittleEndian.Uint32(script.bytes[tmpIndex : tmpIndex+4]))
			tmpIndex += 4
		}
		if script.Size()-tmpIndex < 0 || script.Size()-tmpIndex < nSize {
			return false
		}
		tmpData = append(tmpData, script.bytes[tmpIndex:tmpIndex+nSize]...)
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
		script.bytes = append(script.bytes, byte(n+(OP_1-1)))
	} else if n == 0 {
		script.bytes = append(script.bytes, byte(OP_0))
	} else {
		scriptNum := NewCScriptNum(n)
		script.bytes = append(script.bytes, scriptNum.Serialize()...)
	}
}

func (script *Script) PushOpCode(opcode int) error {
	if opcode < 0 || opcode > 0xff {
		return errors.New("push opcode failed :invalid opcode")
	}
	script.bytes = append(script.bytes, byte(opcode))
	return nil
}

func (script *Script) PushScriptNum(scriptNum *CScriptNum) {
	script.bytes = append(script.bytes, scriptNum.Serialize()...)
}

func (script *Script) PushData(data []byte) {
	dataLen := len(data)
	if dataLen < OP_PUSHDATA1 {
		script.bytes = append(script.bytes, byte(dataLen))
	} else if dataLen <= 0xff {
		script.bytes = append(script.bytes, OP_PUSHDATA1)
		script.bytes = append(script.bytes, byte(dataLen))
	} else if dataLen <= 0xffff {
		script.bytes = append(script.bytes, OP_PUSHDATA2)
		buf := make([]byte, 2)
		binary.LittleEndian.PutUint16(buf, uint16(dataLen))
		script.bytes = append(script.bytes, buf...)

	} else {
		script.bytes = append(script.bytes, OP_PUSHDATA4)
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(script.bytes, uint32(dataLen))
		script.bytes = append(script.bytes, buf...)
	}
	script.bytes = append(script.bytes, data...)
}
*/
/*
func (script *Script) ParseScript() (stk []ParsedOpCode, err error) {
	stk = make([]ParsedOpCode, 0)
	scriptLen := len(script.bytes)

	for i := 0; i < scriptLen; i++ {
		var nSize int
		opcode := script.bytes[i]
		parsedopCode := ParsedOpCode{opValue: opcode}

		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
			parsedopCode.data = script.bytes[i+1 : i+1+nSize]

		} else if opcode == OP_PUSHDATA1 {
			if scriptLen-i < 1 {
				err = errors.New("OP_PUSHDATA1 has no enough data")
				return
			}
			nSize = int(script.bytes[i+1])
			parsedopCode.data = script.bytes[i+2 : i+2+nSize]
			i++

		} else if opcode == OP_PUSHDATA2 {
			if scriptLen-i < 2 {
				err = errors.New("OP_PUSHDATA2 has no enough data")
				return
			}
			nSize = int(binary.LittleEndian.Uint16(script.bytes[i+1 : i+3]))
			parsedopCode.data = script.bytes[i+3 : i+3+nSize]
			i += 2
		} else if opcode == OP_PUSHDATA4 {
			if scriptLen-i < 4 {
				err = errors.New("OP_PUSHDATA4 has no enough data")
				return
			}
			nSize = int(binary.LittleEndian.Uint32(script.bytes[i+1 : i+5]))
			parsedopCode.data = script.bytes[i+5 : i+5+nSize]
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
			script.bytes = append(script.bytes, parseCode.opValue)
			script.bytes = append(script.bytes, parseCode.data...)
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
	/*stk, err := script.ParseScript()
	if err != nil {
		return false
	}
	if len(stk) == 0 {
		return false
	}
	*/

	for _, ops := range script.ParsedOpCodes {
		if ops.opValue > OP_16 {
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
	scriptByte = append(scriptByte, script.bytes...)
	return scriptByte
}
*/
func (script *Script) GetSigOpCount(accurate bool) (int, error) {
	n := 0
	//stk, err := script.ParseScript()
	//if err != nil {
	//	return n, err
	//}
	var lastOpcode int
	for _, e := range script.ParsedOpCodes {
		opcode := e.opValue
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
		lastOpcode = int(opcode)
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

func DecodeOPN(opcode int) (int, error) {
	if opcode < OP_0 || opcode > OP_16 {
		return 0, errors.New(" DecodeOPN opcode is out of bounds")
	}
	return opcode - (OP_1 - 1), nil
}

func (script *Script) Size() int {
	return len(script.bytes)
}

func (script *Script) IsEqual(script2 *Script) bool {
	/*if script.Size() != script2.Size() {
		return false
	}*/

	return bytes.Equal(script.bytes, script2.bytes)
}
