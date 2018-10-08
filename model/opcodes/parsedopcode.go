package opcodes

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

type ParsedOpCode struct {
	OpValue byte

	Length int
	Data   []byte
}

// alwaysIllegal returns whether or not the opcode is always illegal when passed
// over by the program counter even if in a non-executed branch (it isn't a
// coincidence that they are conditionals).
func (parsedOpCode *ParsedOpCode) alwaysIllegal() bool {
	switch parsedOpCode.OpValue {
	case OP_VERIF:
		return true
	case OP_VERNOTIF:
		return true
	default:
		return false
	}
}

func (parsedOpCode *ParsedOpCode) isConditional() bool {
	switch parsedOpCode.OpValue {
	case OP_IF:
		return true
	case OP_NOTIF:
		return true
	case OP_ELSE:
		return true
	case OP_ENDIF:
		return true
	default:
		return false
	}
}

func (parsedOpCode *ParsedOpCode) CheckCompactDataPush() bool {
	dataLen := len(parsedOpCode.Data)
	opcode := parsedOpCode.OpValue
	if dataLen <= 75 {
		return int(opcode) == dataLen
	}
	if dataLen <= 255 {
		return opcode == OP_PUSHDATA1
	}
	if dataLen <= 65535 {
		return opcode == OP_PUSHDATA2
	}
	return opcode == OP_PUSHDATA4
}

func (parsedOpCode *ParsedOpCode) CheckMinimalDataPush() bool {
	data := parsedOpCode.Data
	dataLen := len(data)
	opcode := parsedOpCode.OpValue
	if dataLen == 0 {
		return opcode == OP_0
	}
	if dataLen == 1 {
		if data[0] >= 1 && data[0] <= 16 {
			if opcode != OP_1+data[0]-1 {
				return false
			}
		} else if data[0] == 0x81 {
			if opcode != OP_1NEGATE {
				return false
			}
		}
		return true
	}

	return parsedOpCode.CheckCompactDataPush()
}

func (parsedOpCode *ParsedOpCode) bytes() ([]byte, error) {
	var retBytes []byte
	if parsedOpCode.Length > 0 {
		retBytes = make([]byte, 1, parsedOpCode.Length)
	} else {
		retBytes = make([]byte, 1, 1+len(parsedOpCode.Data)-parsedOpCode.Length)
	}
	retBytes[0] = parsedOpCode.OpValue
	if parsedOpCode.Length == 1 {
		if len(parsedOpCode.Data) != 0 {
			return nil, errors.Errorf(
				"internal consistency error parsed opCode %d has data length %d when %d was expected",
				parsedOpCode.OpValue, len(parsedOpCode.Data), 0)
		}
		return retBytes, nil
	}
	nBytes := parsedOpCode.Length
	if parsedOpCode.Length < 0 {
		l := len(parsedOpCode.Data)
		switch parsedOpCode.Length {
		case -1:
			retBytes = append(retBytes, byte(l))
			nBytes = int(retBytes[1]) + len(retBytes)
		case -2:
			retBytes = append(retBytes, byte(l&0xff), byte(l>>8&0xff))
			nBytes = int(binary.LittleEndian.Uint16(retBytes[1:])) + len(retBytes)
		case -4:
			retBytes = append(retBytes, byte(l&0xff),
				byte((l>>8)&0xff), byte((l>>16)&0xff),
				byte((l>>24)&0xff))
			nBytes = int(binary.LittleEndian.Uint32(retBytes[1:])) +
				len(retBytes)

		}
	}
	retBytes = append(retBytes, parsedOpCode.Data...)
	if len(retBytes) != nBytes {
		return nil, errors.Errorf(
			"internal consistency error - parsed opCode %d has data length %d when %d was expected",
			parsedOpCode.OpValue, len(retBytes), nBytes)
	}
	return retBytes, nil
}

func NewParsedOpCode(opValue byte, length int, Data []byte) *ParsedOpCode {
	newParsedOpCodeData := make([]byte, len(Data))
	copy(newParsedOpCodeData, Data)
	return &ParsedOpCode{OpValue: opValue, Length: length, Data: newParsedOpCodeData}
}
