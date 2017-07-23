package scripts

//todo as the same assign values

type Script struct {
	raw      []byte
	opsWords [][]byte
	//todo add IsPayToScriptHash,IsPayToWitnessScriptHash
}

func (script *Script) ConvertRaw() {

}

func (script *Script) ConvertOPS() {

}

func (script *Script) Check() bool {
	return false
}
func (script *Script) IsPayToScriptHash() bool {
	size := len(script.raw)
	return size == 23 &&
		script.raw[0] == OP_HASH160 &&
		script.raw[1] == 0x14 &&
		script.raw[22] == OP_EQUAL

}

func NewScriptWithRaw(bytes []byte) *Script {
	script := Script{raw: bytes}
	script.ConvertOPS()
	return &script
}

func NewScript(bytes [][]byte) *Script {
	script := Script{opsWords: bytes}
	script.ConvertRaw()
	return &script
}
