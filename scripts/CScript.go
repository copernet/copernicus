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

func (script *CScript) ParseScript() (stk [][]byte, err error) {
	stk = make([][]byte, 0, len(script.bytes))
	scriptLen := len(script.bytes)

	for i := 0; i < scriptLen; {
		var nSize int
		opcode := script.bytes[i]
		opcodeArray := make([]byte, 0, 1)
		opcodeArray = append(opcodeArray, opcode)
		stk = append(stk, opcodeArray)
		var opdata []byte
		if opcode < OP_PUSHDATA1 {
			nSize = int(opcode)
			opcodeArray = script.bytes[i : i+nSize]
		} else if opcode == OP_PUSHDATA1 {
			if scriptLen-i < 1 {
				err = errors.New("OP_PUSHDATA1 has no enough data")
				return
			}
			nSize = i + 1
		} else if opcode == OP_PUSHDATA2 {
			if scriptLen-i < 2 {
				err = errors.New("OP_PUSHDATA2 has no enough data")
				return
			}
			nSize = int(binary.LittleEndian.Uint16(script.bytes[:0]))
			opcodeArray = script.bytes[i+2 : i+2+nSize]
			i += 2
		} else if opcode == OP_PUSHDATA4 {
			if scriptLen-i < 4 {
				err = errors.New("OP_PUSHDATA4 has no enough data")
				return
			}
			opcodeArray = script.bytes[i+4 : i+4+nSize]
			nSize = int(binary.LittleEndian.Uint32(script.bytes[:0]))
			i += 4
		}
		stk = append(stk, opdata)
		if scriptLen-i < 0 || (scriptLen-i) < nSize {
			err = errors.New("size is wrong")
			return
		}
		i += nSize
	}
	return

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

func EncodeOPN(n int) (int, error) {
	if n < 0 || n > 16 {
		return 0, errors.New("EncodeOPN n is out of bounds")
	}
	if n == 0 {
		return OP_0, nil
	}
	return OP_1 + n - 1, nil
}
