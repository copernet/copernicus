package model

type ScriptFreeList chan []byte

const (
	freeListMaxScriptSize = 512
)

func (scriptFreeList ScriptFreeList) Borrow(size uint64) []byte {
	if size > freeListMaxScriptSize {
		return make([]byte, size)
	}
	var buf []byte
	select {
	case buf = <-scriptFreeList:
	default:
		buf = make([]byte, freeListMaxScriptSize)
	}
	return buf[:size]
}
func (scriptFreeList ScriptFreeList) Return(buf []byte) {
	if cap(buf) != freeListMaxScriptSize {
		return
	}
	select {
	case scriptFreeList <- buf:
	default:
	}

}
