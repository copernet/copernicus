package model

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

type BlockHeader struct {
	Version        int32
	HashPrevBlock  utils.Hash
	HashMerkleRoot utils.Hash
	Time           uint32
	Bits           uint32
	Nonce          uint32
}

const blockHeaderLenth = 16 + utils.HashSize*32

func NewBlockHeader() *BlockHeader {
	blHe := BlockHeader{}
	blHe.SetNull()
	return &blHe
}

func (blHe *BlockHeader) IsNull() bool {
	return blHe.Bits == 0
}

func (blHe *BlockHeader) GetBlockTime() uint32 {
	return blHe.Time
}

func (blHe *BlockHeader) GetHash() (utils.Hash, error) {
	buf := bytes.NewBuffer(make([]byte, 0, blockHeaderLenth))
	err := blHe.Serialize(buf)
	return core.DoubleSha256Hash(buf.Bytes()), err
}

func (blHe *BlockHeader) SetNull() {
	blHe.Version = 0
	blHe.HashPrevBlock = utils.HashZero
	blHe.HashMerkleRoot = utils.HashZero
	blHe.Time = 0
	blHe.Bits = 0
	blHe.Nonce = 0
}

func (blHe *BlockHeader) Serialize(writer io.Writer) error {
	if err := binary.Write(writer, binary.LittleEndian, blHe.Version); err != nil {
		return err
	}
	if _, err := writer.Write(blHe.HashPrevBlock.GetCloneBytes()); err != nil {
		return err
	}
	if _, err := writer.Write(blHe.HashMerkleRoot.GetCloneBytes()); err != nil {
		return err
	}
	if err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, blHe.Time); err != nil {
		return err
	}
	if err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, blHe.Bits); err != nil {
		return err
	}
	if err := utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, blHe.Nonce); err != nil {
		return err
	}
	return nil
}

func (blHe *BlockHeader) Deserialize(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &blHe.Version); err != nil {
		return err
	}
	if _, err := io.ReadFull(r, blHe.HashPrevBlock[:]); err != nil {
		return err
	}
	if _, err := io.ReadFull(r, blHe.HashMerkleRoot[:]); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &blHe.Time); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &blHe.Bits); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &blHe.Nonce); err != nil {
		return err
	}

	return nil
}

func (blHe *BlockHeader) ToString() string {
	var hash utils.Hash
	var err error
	if hash, err = blHe.GetHash(); err != nil {
		return ""
	}
	return fmt.Sprintf("Block version : %d, hashPrevBlock : %s, hashMerkleRoot : %s,"+
		"Time : %d, Bits : %d, nonce : %d, BlockHash : %s\n", blHe.Version, blHe.HashPrevBlock.ToString(),
		blHe.HashMerkleRoot.ToString(), blHe.Time, blHe.Bits, blHe.Nonce, hash.ToString())
}
