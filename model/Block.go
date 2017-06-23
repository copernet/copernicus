package model

import (
	"bytes"
	"copernicus/utils"
	"encoding/binary"
	"encoding/hex"
	"time"
	
	"copernicus/crypto"
)

var EmptyByte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

type Block struct {
	Raw          []byte
	Hash         crypto.Hash
	Height       int32
	Transactions []*Transaction
	Version      uint32
	MerkleRoot   string
	BlockTime    time.Time
	Bits         uint32
	Nonce        uint32
	Size         uint32
	TotalBTC     uint64
	BlockReward  float64
	PrevBlock    string
	NextBlock    string
}

func ParseBlock(raw []byte) (block *Block, err error) {
	block = new(Block)
	block.Raw = raw
	block.Hash = crypto.Hash(raw[:80])
	block.Version = binary.LittleEndian.Uint32(raw[0:4])
	if !bytes.Equal(raw[4:36], EmptyByte) {
		block.PrevBlock = utils.ToHash256String(raw[4:36])
	}
	block.MerkleRoot = hex.EncodeToString(raw[36:68])
	//block.BlockTime = binary.LittleEndian.Uint32(raw[68:72])
	block.Bits = binary.LittleEndian.Uint32(raw[72:76])
	block.Nonce = binary.LittleEndian.Uint32(raw[76:80])
	block.Size = uint32(len(raw))
	//txs, _ := ParseTranscation(raw[80:])
	//block.Transactions = txs
	return
}
