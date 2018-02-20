package blockchain

import (
	"encoding/binary"
	"io"

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

func (blockFileInfo *BlockFileInfo) Serialize(writer io.Writer) error {
	err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, blockFileInfo.Blocks)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, blockFileInfo.Size)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, blockFileInfo.UndoSize)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint64(writer, binary.LittleEndian, blockFileInfo.timeFirst)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, blockFileInfo.HeightFirst)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, blockFileInfo.HeightLast)
	if err != nil {
		return err
	}
	return utils.BinarySerializer.PutUint64(writer, binary.LittleEndian, blockFileInfo.timeLast)
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

func (blockFileInfo *BlockFileInfo) AddBlock(nHeightIn uint32, timeIn uint64) {
	if blockFileInfo.Blocks == 0 || blockFileInfo.HeightFirst > nHeightIn {
		blockFileInfo.HeightFirst = nHeightIn
	}
	if blockFileInfo.Blocks == 0 || blockFileInfo.timeFirst > timeIn {
		blockFileInfo.timeFirst = timeIn
	}
	blockFileInfo.Blocks++
	if nHeightIn > blockFileInfo.HeightLast {
		blockFileInfo.HeightLast = nHeightIn
	}
	if timeIn > blockFileInfo.timeLast {
		blockFileInfo.timeLast = timeIn
	}
}
