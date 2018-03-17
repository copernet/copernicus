package blockchain

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

type DiskBlockIndex struct {
	*core.BlockIndex
	hashPrev utils.Hash
}

func (diskBlockIndex *DiskBlockIndex) Serialize(writer io.Writer) error {
	err := utils.WriteVarInt(writer, uint64(diskBlockIndex.Height))
	if err != nil {
		return err
	}
	err = utils.WriteVarInt(writer, uint64(diskBlockIndex.Status))
	if err != nil {
		return err
	}
	err = utils.WriteVarInt(writer, uint64(diskBlockIndex.TxCount))
	if err != nil {
		return err
	}
	if diskBlockIndex.Status&(core.BlockHaveData|core.BlockHaveUndo) != 0 {
		err = utils.WriteVarInt(writer, uint64(diskBlockIndex.File))
		if err != nil {
			return err
		}
	}
	if diskBlockIndex.Status&core.BlockHaveData != 0 {
		err = utils.WriteVarInt(writer, uint64(diskBlockIndex.DataPos))
		if err != nil {
			return err
		}
	}
	if diskBlockIndex.Status&core.BlockHaveUndo != 0 {
		err = utils.WriteVarInt(writer, uint64(diskBlockIndex.UndoPos))
		if err != nil {
			return err
		}
	}
	err = binary.Write(writer, binary.LittleEndian, diskBlockIndex.Version)
	if err != nil {
		return err
	}
	_, err = writer.Write(diskBlockIndex.hashPrev.GetCloneBytes())
	if err != nil {
		return err
	}
	_, err = writer.Write(diskBlockIndex.MerkleRoot.GetCloneBytes())
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, diskBlockIndex.Time)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, diskBlockIndex.Bits)
	if err != nil {
		return err
	}
	err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, diskBlockIndex.Nonce)
	return err
}

func (diskBlockIndex *DiskBlockIndex) ToString(writer io.Writer) string {
	str := "DiskBlockIndex("
	str += diskBlockIndex.BlockIndex.ToString()
	str += fmt.Sprintf("\n\thashBlock=%s, hashPrev=%s)", diskBlockIndex.BlockHash.ToString(), diskBlockIndex.hashPrev.ToString())
	return str
}

func NewDiskBlockIndex(bl *core.BlockIndex) *DiskBlockIndex {
	dbi := DiskBlockIndex{
		BlockIndex: bl,
	}
	if bl.Prev == nil {
		dbi.hashPrev = utils.HashZero
	} else {
		dbi.hashPrev = *bl.Prev.GetBlockHash()
	}
	return nil
}
