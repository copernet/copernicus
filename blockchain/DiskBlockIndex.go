package blockchain

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/utils"
)

type DiskBlockIndex struct {
	BlockIndex
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
	err = utils.WriteVarInt(writer, uint64(diskBlockIndex.Txs))
	if err != nil {
		return err
	}
	if diskBlockIndex.Status&(BLOCK_HAVE_DATA|BLOCK_HAVE_UNDO) != 0 {
		err = utils.WriteVarInt(writer, uint64(diskBlockIndex.File))
		if err != nil {
			return err
		}
	}
	if diskBlockIndex.Status&BLOCK_HAVE_DATA != 0 {
		err = utils.WriteVarInt(writer, uint64(diskBlockIndex.DataPosition))
		if err != nil {
			return err
		}
	}
	if diskBlockIndex.Status&BLOCK_HAVE_UNDO != 0 {
		err = utils.WriteVarInt(writer, uint64(diskBlockIndex.UndoPosition))
		if err != nil {
			return err
		}
	}
	err = binary.Write(writer, binary.LittleEndian, diskBlockIndex.Version)
	if err != nil {
		return err
	}
	_, err = writer.Write(diskBlockIndex.PPrev.PHashBlock.GetCloneBytes())
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
	str += fmt.Sprintf("\n\thashBlock=%s, hashPrev=%s)", diskBlockIndex.PHashBlock.ToString(), diskBlockIndex.PPrev.PHashBlock.ToString())
	return str
}

func NewDiskBlockIndex(pindex *BlockIndex) *DiskBlockIndex {
	return nil
}
