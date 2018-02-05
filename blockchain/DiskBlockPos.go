package blockchain

import (
	"fmt"
	"io"
	"github.com/btcboost/copernicus/utils"
)

type DiskBlockPos struct {
	File int
	Pos  int
}

func (diskBlockPos *DiskBlockPos) Serialize(writer io.Writer) error {
	err := utils.WriteVarInt(writer, uint64(diskBlockPos.File))
	if err != nil {
		return err
	}
	return utils.WriteVarInt(writer, uint64(diskBlockPos.Pos))
}

func DeserializeDiskBlock(reader io.Reader) (*DiskBlockPos, error) {
	file, err := utils.ReadVarInt(reader)
	if err != nil {
		return nil, err
	}
	pos, err := utils.ReadVarInt(reader)
	
	if err != nil {
		return nil, err
	}
	diskBlockPos := DiskBlockPos{File: int(file), Pos: int(pos)}
	return &diskBlockPos, nil
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

func (diskBlockPos *DiskBlockPos) ToString() string {
	return fmt.Sprintf("BlcokDiskPos(File=%i, Pos=%i)", diskBlockPos.File, diskBlockPos.Pos)
}

func NewDiskBlockPos(file int, pos int) *DiskBlockPos {
	diskBlockPos := DiskBlockPos{File: file, Pos: pos}
	return &diskBlockPos
}
