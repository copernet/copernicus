package scripts

import (
	"encoding/binary"
	"errors"
)

const (
	DEFAULT_SIZE = 28
)

type CScript struct {
	bytes []byte
}

func (script *CScript) PushInt64(n int64) {

	if n == -1 || (n >= 1 && n <= 16) {
		script.bytes[len(script.bytes)-1] = byte(n + (OP_1 - 1))
	} else if n == 0 {
		script.bytes[len(script.bytes)-1] = OP_0
	} else {
		scriptNum := NewCScriptNum(n)
		script.bytes = append(script.bytes, scriptNum.Serialize()...)
	}
}

func (script *CScript) PushOpCode(opcode int) error {
	if opcode < 0 || opcode > 0xff {
		return errors.New("push opcode failed :invalid opcode")
	}
	script.bytes = append(script.bytes, byte(opcode))
	return nil
}

func (script *CScript) PushScriptNum(scriptNum *CScriptNum) {
	script.bytes = append(script.bytes, scriptNum.Serialize()...)
}

func (script *CScript) PushData(data []byte) {
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
