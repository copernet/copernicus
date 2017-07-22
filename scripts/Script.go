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

func NewScriptWithRaw(bytes []byte) *Script {
	return nil
}

func NewScript(bytes [][]byte) *Script {
	return nil
}
