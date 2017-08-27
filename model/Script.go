package model

//todo as the same assign values

type Script struct {
	Raw      []byte
	OpsWords [][]byte
	//todo add IsPayToScriptHash,IsPayToWitnessScriptHash
}

func (script *Script) ConvertRaw() {

}

func (script *Script) ConvertOPS() {

}

func (script *Script) Size() int {
	return 0
}

func (script *Script) Check() bool {
	return false
}
func (script *Script) IsPayToScriptHash() bool {
	size := len(script.Raw)
	return size == 23 &&
		script.Raw[0] == OP_HASH160 &&
		script.Raw[1] == 0x14 &&
		script.Raw[22] == OP_EQUAL

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

func (script *Script) IsPushOnly() bool {
	return true
}

func NewScript(bytes [][]byte) *Script {
	script := Script{OpsWords: bytes}
	script.ConvertRaw()
	return &script
}
func NewScriptRaw(bytes []byte) *Script {
	script := Script{Raw: bytes}
	script.ConvertOPS()
	return &script
}
