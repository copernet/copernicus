package blockchain

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
	
	"github.com/btcboost/copernicus/utils"
)

type BlockFileInfo struct {
	Blocks      uint32 // number of blocks stored in file
	Size        uint32 // number of used bytes of block file
	UndoSize    uint32 // number of used bytes in the undo file
	HeightFirst uint32 // lowest height of block in file
	HeightLast  uint32 // highest height of block in file
	timeFirst   uint64 // earliest time of block in file
	timeLast    uint64 // latest time of block in file
	
	index uint32
}

func (bfi *BlockFileInfo) SetNull() {
	bfi.Size = 0
	bfi.timeFirst = 0
	bfi.UndoSize = 0
	bfi.timeLast = 0
	bfi.HeightLast = 0
	bfi.HeightFirst = 0
	bfi.Blocks = 0
}

func (bfi *BlockFileInfo) Serialize(writer io.Writer) error {
	err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, bfi.Blocks)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, bfi.Size)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, bfi.UndoSize)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint64(writer, binary.LittleEndian, bfi.timeFirst)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, bfi.HeightFirst)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, bfi.HeightLast)
	if err != nil {
		return err
	}
	return utils.BinarySerializer.PutUint64(writer, binary.LittleEndian, bfi.timeLast)
}

func DeserializeBlockFileInfo(reader io.Reader) (*BlockFileInfo, error) {
	blockFileInfo := new(BlockFileInfo)
	blocks, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	size, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	undoSize, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	heightFirst, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	heightLast, err := utils.BinarySerializer.Uint32(reader, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	
	timeFirst, err := utils.BinarySerializer.Uint64(reader, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	timeLast, err := utils.BinarySerializer.Uint64(reader, binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	blockFileInfo.Blocks = blocks
	blockFileInfo.Size = size
	blockFileInfo.UndoSize = undoSize
	blockFileInfo.HeightFirst = heightFirst
	blockFileInfo.HeightLast = heightLast
	blockFileInfo.timeFirst = timeFirst
	blockFileInfo.timeLast = timeLast
	return blockFileInfo, nil
	
}

func (bfi *BlockFileInfo) AddBlock(nHeightIn uint32, timeIn uint64) {
	if bfi.Blocks == 0 || bfi.HeightFirst > nHeightIn {
		bfi.HeightFirst = nHeightIn
	}
	if bfi.Blocks == 0 || bfi.timeFirst > timeIn {
		bfi.timeFirst = timeIn
	}
	bfi.Blocks++
	if nHeightIn > bfi.HeightLast {
		bfi.HeightLast = nHeightIn
	}
	if timeIn > bfi.timeLast {
		bfi.timeLast = timeIn
	}
}

func (bfi *BlockFileInfo) ToString() string {
	return fmt.Sprintf("CBlockFileInfo(blocks=%d, size=%d, heights=%d...%d, time=%s...%s)",
		bfi.Blocks, bfi.Size, bfi.HeightFirst, bfi.HeightLast,
		time.Unix(int64(bfi.timeFirst), 0).Format(time.RFC3339),
		time.Unix(int64(bfi.timeLast), 0).Format(time.RFC3339))
}

func NewBlockFileInfo() *BlockFileInfo {
	bfi := new(BlockFileInfo)
	return bfi
}
