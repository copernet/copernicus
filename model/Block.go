package model

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
)

var EmptyByte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

type Block struct {
	Raw          []byte
	Hash         utils.Hash
	BlockHeader  BlockHeader
	Height       int32
	Transactions []*Tx
	txNum        uint32
	Size         uint32
	TotalBTC     uint64
	BlockReward  float64
	NextBlock    utils.Hash
	fChecked     bool
}

func ParseBlock(raw []byte) (block *Block, err error) {
	block = new(Block)
	block.Raw = raw
	block.Hash = core.DoubleSha256Hash(raw[:80])
	block.BlockHeader.Version = int32(binary.LittleEndian.Uint32(raw[0:4]))
	if !bytes.Equal(raw[4:36], EmptyByte) {
		block.BlockHeader.HashMerkleRoot = core.DoubleSha256Hash(raw[4:36])
	}
	block.BlockHeader.HashMerkleRoot = core.DoubleSha256Hash(raw[36:68])
	//block.BlockTime = binary.LittleEndian.Uint32(txRaw[68:72])
	block.BlockHeader.Bits = binary.LittleEndian.Uint32(raw[72:76])
	block.BlockHeader.Nonce = binary.LittleEndian.Uint32(raw[76:80])
	block.Size = uint32(len(raw))
	//txs, _ := ParseTranscation(txRaw[80:])
	//block.Transactions = txs
	return
}

func (bl *Block) GetBlockHeader() BlockHeader {
	return bl.BlockHeader
}

func (bl *Block) SetNull() {
	bl.BlockHeader.SetNull()
	bl.Transactions = nil
	bl.fChecked = false
}

func (bl *Block) Serialize(w io.Writer) error {
	if err := bl.BlockHeader.Serialize(w); err != nil {
		return err
	}
	for _, v := range bl.Transactions {
		if err := v.Serialize(w); err != nil {
			return err
		}
	}

	return nil
}

func (bl *Block) Deserialize(r io.Reader) error {
	bl.BlockHeader.Deserialize(r)
	for i := uint32(0); i < bl.txNum; i++ {
		if tx, err := DeserializeTx(r); err != nil {
			return err
		} else {
			bl.Transactions = append(bl.Transactions, tx)
		}
	}

	return nil
}

func NewBlock() *Block {
	bl := Block{}
	bl.SetNull()
	return &bl
}
