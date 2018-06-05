package outpoint

import (
	"encoding/binary"
	"fmt"

	//"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/util"
	"io"
	"math"
)

type OutPoint struct {
	Hash  util.Hash
	Index uint32
}

func NewOutPoint(hash util.Hash, index uint32) *OutPoint {
	outPoint := OutPoint{
		Hash:  hash,
		Index: index,
	}
	return &outPoint
}

func (outPoint *OutPoint) SerializeSize() uint32 {
	return outPoint.EncodeSize()
}

func (outPoint *OutPoint) Serialize(writer io.Writer) error {
	return outPoint.Encode(writer)
}

func (outPoint *OutPoint) Unserialize(reader io.Reader) (err error) {
	return outPoint.Decode(reader)
}

func (outPoint *OutPoint) EncodeSize() uint32 {
	return outPoint.Hash.EncodeSize() + 4
}

func (outPoint *OutPoint) Encode(writer io.Writer) error {
	_, err := writer.Write(outPoint.Hash[:])
	if err != nil {
		return err
	}
	return util.BinarySerializer.PutUint32(writer, binary.LittleEndian, outPoint.Index)
}

func (outPoint *OutPoint) Decode(reader io.Reader) (err error) {
	_, err = io.ReadFull(reader, outPoint.Hash[:])
	if err != nil {
		return
	}
	outPoint.Index, err = util.BinarySerializer.Uint32(reader, binary.LittleEndian)
	//log.Debug("outpoint: prehash:%v, index:%d", outPoint.Hash[:], outPoint.Index)
	return
}

func (outPoint *OutPoint) String() string {
	return fmt.Sprintf("OutPoint ( hash:%s index: %d)", outPoint.Hash.String(), outPoint.Index)
}

func (outPoint *OutPoint) IsNull() bool {
	if outPoint == nil {
		return true
	}
	if outPoint.Index != math.MaxUint32 {
		return false
	}
	return outPoint.Hash.IsEqual(&util.HashZero)
}
