package model

import (
	"bytes"
	"encoding/binary"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

var EmptyByte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

type Block struct {
	Raw          []byte
	Hash         utils.Hash
	Height       int32
	Transactions []*Tx
	Version      uint32
	MerkleRoot   utils.Hash
	BlockTime    uint32
	Bits         uint32
	Nonce        uint32
	Size         uint32
	TotalBTC     uint64
	BlockReward  float64
	PrevBlock    utils.Hash
	NextBlock    utils.Hash
}

func ParseBlock(raw []byte) (block *Block, err error) {
	block = new(Block)
	block.Raw = raw
	block.Hash = core.DoubleSha256Hash(raw[:80])
	block.Version = binary.LittleEndian.Uint32(raw[0:4])
	if !bytes.Equal(raw[4:36], EmptyByte) {
		block.PrevBlock = core.DoubleSha256Hash(raw[4:36])
	}
	block.MerkleRoot = core.DoubleSha256Hash(raw[36:68])
	//block.BlockTime = binary.LittleEndian.Uint32(txRaw[68:72])
	block.Bits = binary.LittleEndian.Uint32(raw[72:76])
	block.Nonce = binary.LittleEndian.Uint32(raw[76:80])
	block.Size = uint32(len(raw))
	//txs, _ := ParseTranscation(txRaw[80:])
	//block.Transactions = txs
	return
}
