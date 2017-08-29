package model

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	DEFAULT_SIZE = 28

	// MAX_PUBKEYS_PER_MULTISIG :  Maximum number of public keys per multisig
	MAX_PUBKEYS_PER_MULTISIG = 20

	// LOCKTIME_THRESHOLD Threshold for nLockTime: below this value it is interpreted as block number,
	// otherwise as UNIX timestamp. Thresold is Tue Nov 5 00:53:20 1985 UTC
	LOCKTIME_THRESHOLD = 500000000

	// SEQUENCE_FINAL Setting nSequence to this value for every input in a transaction
	// disables nLockTime.
	SEQUENCE_FINAL = 0xffffffff

	MAX_SCRIPT_SIZE         = 10000
	MAX_SCRIPT_ELEMENT_SIZE = 520
	MAX_SCRIPT_OPCODES      = 201
	MAX_OPS_PER_SCRIPT      = 201
)

type Script struct {
	bytes         []byte
	ParsedOpCodes []ParsedOpCode
}

func (script *Script) ConvertRaw() {
	script.bytes = make([]byte, 0)
	for i := 0; i < len(script.ParsedOpCodes); i++ {
		parsedOpcode := script.ParsedOpCodes[i]
		script.bytes = append(script.bytes, parsedOpcode.opValue)
		script.bytes = append(script.bytes, parsedOpcode.data...)
	}

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
	size := len(script.bytes)
	return size == 23 &&
		script.bytes[0] == OP_HASH160 &&
		script.bytes[1] == 0x14 &&
		script.bytes[22] == OP_EQUAL

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

func NewScriptRaw(bytes []byte) *Script {
	script := Script{bytes: bytes}
	script.ConvertOPS()
	return &script
}

func (script *Script) PushInt64(n int64) {

	if n == -1 || (n >= 1 && n <= 16) {
		script.bytes[len(script.bytes)-1] = byte(n + (OP_1 - 1))
	} else if n == 0 {
		script.bytes[len(script.bytes)-1] = OP_0
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
		data[dataLen-1] = byte(dataLen)
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

func (script *Script) ParseScript() (stk []ParsedOpCode, err error) {
	stk = make([]ParsedOpCode, 0)
	scriptLen := len(script.bytes)

	for i := 0; i < scriptLen; i++ {
		var nSize int
		opcode := script.bytes[i]
		parsedopCode := ParsedOpCode{opValue: opcode}
		stk = append(stk, parsedopCode)
		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
			fmt.Println(i, i+nSize, len(script.bytes))
			parsedopCode.data = script.bytes[i : i+nSize]

		} else if opcode == OP_PUSHDATA1 {
			if scriptLen-i < 1 {
				err = errors.New("OP_PUSHDATA1 has no enough data")
				return
			}
			nSize = i + 1
			i++
			nSize = int(script.bytes[i+1])
			parsedopCode.data = script.bytes[i+2 : i+2+nSize]
		} else if opcode == OP_PUSHDATA2 {
			if scriptLen-i < 2 {
				err = errors.New("OP_PUSHDATA2 has no enough data")
				return
			}
			nSize = int(binary.LittleEndian.Uint16(script.bytes[:0]))
			parsedopCode.data = script.bytes[i+2 : i+2+nSize]
			i += 2
		} else if opcode == OP_PUSHDATA4 {
			if scriptLen-i < 4 {
				err = errors.New("OP_PUSHDATA4 has no enough data")
				return
			}
			parsedopCode.data = script.bytes[i+4 : i+4+nSize]
			nSize = int(binary.LittleEndian.Uint32(script.bytes[:0]))
			i += 4
		}
		if scriptLen-i < 0 || (scriptLen-i) < nSize {
			err = errors.New("size is wrong")
			return
		}
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
	script.bytes = make([]byte, 0)

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
				n += MAX_PUBKEYS_PER_MULTISIG
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
	return len(script.bytes)
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

func NewScriptWithRaw(bytes []byte) *Script {
	script := Script{bytes: bytes}
	script.ConvertOPS()
	return &script
}

func NewScript(parsedOpCodes []ParsedOpCode) *Script {
	script := Script{ParsedOpCodes: parsedOpCodes}
	script.ConvertRaw()
	return &script
}
