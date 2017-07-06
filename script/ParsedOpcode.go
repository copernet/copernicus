package script

import (
	"fmt"
	"github.com/pkg/errors"
)

type ParsedOpCode struct {
	opValue byte
	name    string
	length  int
	data    []byte
	opFunc  OpFunc
}

// isDisabled returns whether or not the opcode is disabled and thus is always
// bad to see in the instruction stream (even if turned off by a conditional).
func (parsedOpCode *ParsedOpCode) isDisabled() bool {
	switch parsedOpCode.opValue {
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
	switch parsedOpCode.opValue {
	case OP_VERIF:
		return true
	case OP_VERNOTIF:
		return true
	default:
		return false
	}
}

func (parsedOpCode *ParsedOpCode) isConditional() bool {
	switch parsedOpCode.opValue {
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
	data := parsedOpCode.data
	dataLen := len(data)
	opcode := parsedOpCode.opValue
	if dataLen == 0 && opcode != OP_0 {
		return errors.Errorf(
			"zero length data push is encode with op code %s instead of OP_0",
			parsedOpCode.name)
	} else if dataLen == 1 {
		if data[0] >= 1 && data[0] <= 16 {
			if opcode != OP_1+data[0]-1 {
				// Should have used OP_1 .. OP_16
				return errors.Errorf(
					"data push of the value %d encoded with opcode %s instead of op_%d",
					data[0], parsedOpCode.name, data[0])
			}
		} else if data[0] == 0x81 {
			if opcode != OP_1NEGATE {
				return errors.Errorf(
					"data push of the value -1 encoded with opcode %s instend of OP_1NEGATE",
					parsedOpCode.name)
			}
		}
	} else if dataLen <= 75 {
		if int(opcode) != dataLen {
			return errors.Errorf(
				"data push of %d bytes encoded with opcode %s instead of op_data_%d",
				dataLen, parsedOpCode.name, dataLen)
		}
	} else if dataLen <= 255 {
		if opcode != OP_PUSHDATA1 {
			return errors.Errorf(
				" data push of %d bytes encoded with opcode %s instead of OP_PUSHDATA1",
				dataLen, parsedOpCode.name)
		}
	} else if dataLen <= 65535 {
		if opcode != OP_PUSHDATA2 {
			return errors.Errorf(
				"data push of %d bytes encoded with opcode %s instead of OP_PUSHDATA2",
				dataLen, parsedOpCode.name)
		}
	}
	return nil
}

func (parsedOpCode *ParsedOpCode) print(oneline bool) string {
	opcodeName := parsedOpCode.name
	if oneline {
		if replName, ok := OpcodeOnelineRepls[opcodeName]; ok {
			opcodeName = replName
		}
		if parsedOpCode.length == 1 {
			return opcodeName
		}
		return fmt.Sprintf("%x", parsedOpCode.data)
	}
	// Nothing more to do for non-data push opcodes.
	if parsedOpCode.length == 1 {
		return opcodeName
	}
	// Add length for the OP_PUSHDATA# opcodes.
	retString := opcodeName
	switch parsedOpCode.length {
	case -1:
		retString += fmt.Sprintf(" 0x%02x", len(parsedOpCode.data))
	case -2:
		retString += fmt.Sprintf(" 0x%04x", len(parsedOpCode.data))
	case -4:
		retString += fmt.Sprintf(" 0x%08x", len(parsedOpCode.data))
	}
	return fmt.Sprintf("%s 0x%02x", retString, parsedOpCode.data)

}
