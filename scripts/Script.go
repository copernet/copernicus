package scripts

type Script struct {
	bytes    []byte
	opsWords [][]byte
}

func (script *Script) Serialize() {

}

func (script *Script) deSerialize() {

}

func (script *Script) Check() (bool) {
	return false
}

func Construct(bytes []byte) (*Script) {
	return nil
}
