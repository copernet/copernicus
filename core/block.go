package core

import (
	"bytes"
	"encoding/binary"
	"io"
	"unsafe"

	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
)

var emptyByte = bytes.Repeat([]byte{0}, 32)

type Block struct {
	Raw         []byte
	Hash        *utils.Hash
	BlockHeader BlockHeader
	Height      int32
	Txs         []*Tx
	Size        uint32
	TotalBTC    uint64
	BlockReward float64
	NextBlock   utils.Hash
	Checked     bool
}

func ParseBlock(raw []byte) (block *Block, err error) {
	block = new(Block)
	block.Raw = raw
	hash := crypto.DoubleSha256Hash(raw[:80])
	block.Hash = &hash
	block.BlockHeader.Version = int32(binary.LittleEndian.Uint32(raw[0:4]))
	if !bytes.Equal(raw[4:36], emptyByte) {
		block.BlockHeader.MerkleRoot = crypto.DoubleSha256Hash(raw[4:36])
	}
	// block.BlockTime = binary.LittleEndian.Uint32(txRaw[68:72])
	block.BlockHeader.Bits = binary.LittleEndian.Uint32(raw[72:76])
	block.BlockHeader.Nonce = binary.LittleEndian.Uint32(raw[76:80])
	block.Size = uint32(len(raw))
	// txs, _ := ParseTransaction(txRaw[80:])
	// block.Transactions = txs
	return
}

func (bl *Block) GetBlockHeader() BlockHeader {
	return bl.BlockHeader
}

func (bl *Block) SetNull() {
	bl.BlockHeader.SetNull()
	bl.Txs = nil
	bl.Hash = &utils.HashZero
	bl.Checked = false
}

func (bl *Block) UpdateTime(indexPrev *BlockIndex) int64 {
	oldTime := int64(bl.BlockHeader.Time)
	var newTime int64
	mt := indexPrev.GetMedianTimePast() + 1
	at := utils.GetAdjustedTime()
	if mt > at {
		newTime = mt
	} else {
		newTime = at
	}
	if oldTime < newTime {
		bl.BlockHeader.Time = uint32(newTime)
	}

	// Updating time can change work required on testnet:
	// if params.FPowAllowMinDifficultyBlocks {
	// 	pow := blockchain.Pow{}
	// 	bl.BlockHeader.Bits = pow.GetNextWorkRequired(indexPrev, &bl.BlockHeader, params)
	// }

	return newTime - oldTime
}

func (bl *Block) Serialize(w io.Writer) error {
	if err := bl.BlockHeader.Serialize(w); err != nil {
		return err
	}
	for _, tx := range bl.Txs {
		if err := tx.Serialize(w); err != nil {
			return err
		}
	}

	return nil
}

func (bl *Block) Deserialize(r io.Reader) error {
	bl.BlockHeader.Deserialize(r)
	for i := 0; i < len(bl.Txs); i++ {
		if tx, err := DeserializeTx(r); err != nil {
			bl.Txs = append(bl.Txs, tx)
			return err
		}
	}
	hash, err := bl.BlockHeader.GetHash()
	if err != nil {
		return err
	}
	bl.Hash = &hash
	return nil
}

func (bl *Block) SerializeSize() int {
	size := int(unsafe.Sizeof(BlockHeader{}))
	for _, tx := range bl.Txs {
		size += tx.SerializeSize()
	}
	return size
}

func NewBlock() *Block {
	bl := Block{}
	bl.SetNull()
	return &bl
}
