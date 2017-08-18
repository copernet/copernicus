package model

import (
	"github.com/btcboost/copernicus/utils"
	"github.com/pkg/errors"
	"io"
)

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

func ReadScript(reader io.Reader, pver uint32, maxAllowed uint32, fieldName string) (signScript []byte, err error) {
	count, err := utils.ReadVarInt(reader, pver)
	if err != nil {
		return
	}
	if count > uint64(maxAllowed) {
		err = errors.Errorf("readScript %s is larger than the max allowed size [count %d,max %d]", fieldName, count, maxAllowed)
		return
	}
	buf := scriptPool.Borrow(count)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		scriptPool.Return(buf)
		return
	}
	return buf, nil

}
