package scripts

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
