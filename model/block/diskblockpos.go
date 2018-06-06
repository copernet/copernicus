package block

import (
	"fmt"
	"io"

	"github.com/copernet/copernicus/util"
)

type DiskBlockPos struct {
	File int32
	Pos  uint32
}

type DiskTxPos struct {
	BlockIn    *DiskBlockPos
	TxOffsetIn uint32
}


func (diskBlockPos *DiskBlockPos) Serialize(writer io.Writer) error {
	return  util.WriteElements(writer, &diskBlockPos.File, &diskBlockPos.Pos)
}

func (diskTxPos *DiskTxPos) Serialize(writer io.Writer) error {
	err := diskTxPos.BlockIn.Serialize(writer)
	if err != nil {
		return err
	}
	return util.WriteElements(writer, diskTxPos.TxOffsetIn)
}

func (dbp *DiskBlockPos) Unserialize(reader io.Reader) (error) {
	return util.ReadElements(reader, &dbp.File, &dbp.Pos)
	
}

func (dtp *DiskTxPos) Unserialize(reader io.Reader) (error) {
	dbp := new(DiskBlockPos)
	err := dbp.Unserialize(reader)
	if err != nil{
		return err
	}
	dtp.BlockIn = dbp
	return util.ReadElements(reader, &dtp.TxOffsetIn)
}

func (diskBlockPos *DiskBlockPos) SetNull() {
	diskBlockPos.File = -1
	diskBlockPos.Pos = 0
}

func (diskBlockPos *DiskBlockPos) Equal(other *DiskBlockPos) bool {
	return diskBlockPos.Pos == other.Pos && diskBlockPos.File == other.File

}

func (diskBlockPos *DiskBlockPos) IsNull() bool {
	return diskBlockPos.File == -1
}

func (diskBlockPos *DiskBlockPos) String() string {
	return fmt.Sprintf("BlcokDiskPos(File=%d, Pos=%d)", diskBlockPos.File, diskBlockPos.Pos)
}

func NewDiskBlockPos(file int32, pos uint32) *DiskBlockPos {
	diskBlockPos := DiskBlockPos{File: file, Pos: pos}
	return &diskBlockPos
}

func NewDiskTxPos(blockIn *DiskBlockPos, offsetIn uint32) *DiskTxPos {
	diskTxPos := &DiskTxPos{
		BlockIn:    blockIn,
		TxOffsetIn: offsetIn,
	}
	return diskTxPos
}
