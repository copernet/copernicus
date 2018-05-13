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

// isDisabled returns whether or not the opCode is disabled and thus is always
// bad to see in the instruction stream (even if turned off by a conditional).
func (parsedOpCode *ParsedOpCode) isDisabled() bool {
	switch parsedOpCode.OpValue {
	case OP_CAT:
		return true
	case OP_SUBSTR:
		return true
	case OP_LEFT:
		return true
	case OP_RIGHT:
		return true
	case OP_INVERT:
		return true
	case OP_AND:
		return true
	case OP_OR:
		return true
	case OP_XOR:
		return true
	case OP_2MUL:
		return true
	case OP_2DIV:
		return true
	case OP_MUL:
		return true
	case OP_DIV:
		return true
	case OP_MOD:
		return true
	case OP_LSHIFT:
		return true
	case OP_RSHIFT:
		return true
	default:
		return false
	}
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

func (parsedOpCode *ParsedOpCode) checkMinimalDataPush() error {
	data := parsedOpCode.Data
	dataLen := len(data)
	opcode := parsedOpCode.OpValue
	if dataLen == 0 && opcode != OP_0 {
		return errors.Errorf(
			"zero length data push is encode with op code %d instead of OP_0",
			parsedOpCode.OpValue)
	} else if dataLen == 1 {
		if data[0] >= 1 && data[0] <= 16 {
			if opcode != OP_1+data[0]-1 {
				// Should have used OP_1 .. OP_16
				return errors.Errorf(
					"data push of the value %d encoded with opCode %d instead of op_%d",
					data[0], parsedOpCode.OpValue, data[0])
			}
		} else if data[0] == 0x81 {
			if opcode != OP_1NEGATE {
				return errors.Errorf(
					"data push of the value -1 encoded with opCode %d instead of OP_1NEGATE",
					parsedOpCode.OpValue)
			}
		}
	} else if dataLen <= 75 {
		if int(opcode) != dataLen {
			return errors.Errorf(
				"data push of %d bytes encoded with opCode %d instead of op_data_%d",
				dataLen, parsedOpCode.OpValue, dataLen)
		}
	} else if dataLen <= 255 {
		if opcode != OP_PUSHDATA1 {
			return errors.Errorf(
				" data push of %d bytes encoded with opCode %d instead of OP_PUSHDATA1",
				dataLen, parsedOpCode.OpValue)
		}
	} else if dataLen <= 65535 {
		if opcode != OP_PUSHDATA2 {
			return errors.Errorf(
				"data push of %d bytes encoded with opCode %d instead of OP_PUSHDATA2",
				dataLen, parsedOpCode.OpValue)
		}
	}
	return nil
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
