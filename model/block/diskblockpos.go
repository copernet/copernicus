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

func (dbp *DiskBlockPos) Serialize(writer io.Writer) error {
	return util.WriteElements(writer, &dbp.File, &dbp.Pos)
}

func (dtp *DiskTxPos) Serialize(writer io.Writer) error {
	err := dtp.BlockIn.Serialize(writer)
	if err != nil {
		return err
	}
	return util.WriteElements(writer, dtp.TxOffsetIn)
}

func (dbp *DiskBlockPos) Unserialize(reader io.Reader) error {
	return util.ReadElements(reader, &dbp.File, &dbp.Pos)

}

func (dtp *DiskTxPos) Unserialize(reader io.Reader) error {
	dbp := new(DiskBlockPos)
	err := dbp.Unserialize(reader)
	if err != nil {
		return err
	}
	dtp.BlockIn = dbp
	return util.ReadElements(reader, &dtp.TxOffsetIn)
}

func (dbp *DiskBlockPos) SetNull() {
	dbp.File = -1
	dbp.Pos = 0
}

func (dbp *DiskBlockPos) Equal(other *DiskBlockPos) bool {
	return dbp.Pos == other.Pos && dbp.File == other.File

}

func (dbp *DiskBlockPos) IsNull() bool {
	return dbp.File == -1
}

func (dbp *DiskBlockPos) String() string {
	return fmt.Sprintf("BlcokDiskPos(File=%d, Pos=%d)", dbp.File, dbp.Pos)
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
