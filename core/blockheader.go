package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
)

type BlockHeader struct {
	Version       int32
	HashPrevBlock utils.Hash
	MerkleRoot    utils.Hash
	TimeStamp     uint32
	Bits          uint32
	Nonce         uint32
}

const blockHeaderLength = 16 + utils.Hash256Size*2

func NewBlockHeader() *BlockHeader {
	bh := BlockHeader{}
	bh.SetNull()
	return &bh
}

func (bh *BlockHeader) IsNull() bool {
	return bh.Bits == 0
}

func (bh *BlockHeader) GetBlockTime() uint32 {
	return bh.TimeStamp
}

func (bh *BlockHeader) GetHash() (utils.Hash, error) {
	buf := bytes.NewBuffer(make([]byte, 0, blockHeaderLength))
	err := bh.Serialize(buf)
	return crypto.DoubleSha256Hash(buf.Bytes()), err
}

func (bh *BlockHeader) SetNull() {
	bh.Version = 0
	bh.TimeStamp = 0
	bh.Bits = 0
	bh.Nonce = 0
}

func (bh *BlockHeader) Serialize(writer io.Writer) (err error) {
	if err = binary.Write(writer, binary.LittleEndian, bh.Version); err != nil {
		return
	}
	if _, err = writer.Write(bh.HashPrevBlock.GetCloneBytes()); err != nil {
		return
	}
	if _, err = writer.Write(bh.MerkleRoot.GetCloneBytes()); err != nil {
		return
	}
	if err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, bh.TimeStamp); err != nil {
		return
	}
	if err = utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, bh.Bits); err != nil {
		return
	}
	return utils.BinarySerializer.PutUint32(writer, binary.LittleEndian, bh.Nonce)
}

func (bh *BlockHeader) Deserialize(r io.Reader) (err error) {
	if err = binary.Read(r, binary.LittleEndian, &bh.Version); err != nil {
		return
	}
	if _, err = io.ReadFull(r, bh.HashPrevBlock[:]); err != nil {
		return
	}
	if _, err = io.ReadFull(r, bh.MerkleRoot[:]); err != nil {
		return
	}
	if bh.TimeStamp, err = utils.BinarySerializer.Uint32(r, binary.LittleEndian); err != nil {
		return
	}
	if bh.Bits, err = utils.BinarySerializer.Uint32(r, binary.LittleEndian); err != nil {
		return
	}
	bh.Nonce, err = utils.BinarySerializer.Uint32(r, binary.LittleEndian)

	return
}

func (bh *BlockHeader) ToString() string {
	var hash utils.Hash
	var err error
	if hash, err = bh.GetHash(); err != nil {
		return ""
	}
	return fmt.Sprintf("Block version : %d, hashPrevBlock : %s, hashMerkleRoot : %s,"+
		"Time : %d, Bits : %d, nonce : %d, BlockHash : %s\n", bh.Version, bh.HashPrevBlock.ToString(),
		bh.MerkleRoot.ToString(), bh.TimeStamp, bh.Bits, bh.Nonce, hash.ToString())
}
