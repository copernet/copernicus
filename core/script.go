package core

import (
	"encoding/binary"
	"errors"
	"bytes"
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
func (script *Script) Eval() (int, error) {
	return 0, nil
}

func (script *Script) ConvertRaw() {
	script.byteCodes = make([]byte, 0)
	for i := 0; i < len(script.ParsedOpCodes); i++ {
		parsedOpcode := script.ParsedOpCodes[i]
		script.byteCodes = append(script.byteCodes, parsedOpcode.opValue)
		script.byteCodes = append(script.byteCodes, parsedOpcode.data...)
	}

}

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

func (script *Script) ConvertOPS() error {
	stk, err := script.ParseScript()
	if err != nil {
		return err
	}
	script.ParsedOpCodes = stk
	return nil
}

func (script *Script) Check() bool {
	return false
}
func (script *Script) IsPayToScriptHash() bool {
	size := len(script.byteCodes)
	return size == 23 &&
		script.byteCodes[0] == OP_HASH160 &&
		script.byteCodes[1] == 0x14 &&
		script.byteCodes[22] == OP_EQUAL
}

func (script *Script) IsUnspendable() bool {
	err := script.ConvertOPS()
	if err != nil {
		return false
	}

	return script.Size() > 0 &&
		script.ParsedOpCodes[0].opValue == OP_RETURN ||
		script.Size() > MaxScriptSize
}

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

func NewScriptRaw(byteCodes []byte) *Script {
	script := Script{byteCodes: byteCodes}
	script.ConvertOPS()
	return &script
}

func (script *Script) GetOp(index *int, opCode *byte, data *[]byte) bool {

	opcode := byte(OP_INVALIDOPCODE)
	tmpIndex := *index
	tmpData := make([]byte, 0)
	if tmpIndex >= script.Size() {
		return false
	}

	// Read instruction
	if script.Size()-tmpIndex < 1 {
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
			if script.Size()-tmpIndex < 1 {
				return false
			}
			nSize = int(script.byteCodes[*index])
			tmpIndex++
		} else if opcode == OP_PUSHDATA2 {
			if script.Size()-tmpIndex < 2 {
				return false
			}
			nSize = int(binary.LittleEndian.Uint16(script.byteCodes[tmpIndex : tmpIndex+2]))
			tmpIndex += 2
		} else if opcode == OP_PUSHDATA4 {
			if script.Size()-tmpIndex < 4 {
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
}

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

func (script *Script) FindAndDelete(b *Script) (bool, error) {
	orginalParseCodes, err := script.ParseScript()
	if err != nil {
		return false, err
	}
	paramScript, err := b.ParseScript()
	if err != nil {
		return false, err
	}
	script.byteCodes = make([]byte, 0)

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

func (script *Script) Find(opcode int) bool {
	stk, err := script.ParseScript()
	if err != nil {
		return false
	}
	for i := 0; i < len(stk); i++ {
		if int(stk[i].opValue) == opcode {
			return true
		}
	}
	return false
}

func (script *Script) IsPushOnly() bool {
	stk, err := script.ParseScript()
	if err != nil {
		return false
	}
	if len(stk) == 0 {
		return false
	}
	for i := 0; i < len(stk); i++ {
		if stk[i].opValue > OP_16 {
			return false
		}
	}
	return true

}
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

func (script *Script) GetScriptByte() []byte {
	scriptByte := make([]byte, 0)
	scriptByte = append(scriptByte, script.byteCodes...)
	return scriptByte
}

func (script *Script) GetSigOpCountWithAccurate(accurate bool) (int, error) {
	n := 0
	stk, err := script.ParseScript()
	if err != nil {
		return n, err
	}
	var lastOpcode int
	for i := 0; i < len(stk); i++ {
		opcode := stk[i].opValue
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

func DecodeOPN(opcode int) (int, error) {
	if opcode < OP_0 || opcode > OP_16 {
		return 0, errors.New(" DecodeOPN opcode is out of bounds")
	}
	if opcode == OP_0 {
		return 0, nil
	}
	return opcode - (OP_1 - 1), nil
}

func (script *Script) Size() int {
	return len(script.byteCodes)
}

func (script *Script) IsEqual(script2 *Script) bool {
	if script.Size() != script2.Size() {
		return false
	}

	return bytes.Equal(script.byteCodes, script2.byteCodes)
}

func EncodeOPN(n int) (int, error) {
	if n < 0 || n > 16 {
		return 0, errors.New("EncodeOPN n is out of bounds")
	}
	if n == 0 {
		return OP_0, nil
	}
	return OP_1 + n - 1, nil
}

func NewScript(parsedOpCodes []ParsedOpCode) *Script {
	script := Script{ParsedOpCodes: parsedOpCodes}
	script.ConvertRaw()
	return &script
}
