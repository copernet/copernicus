package block

import (
	"bytes"
	"fmt"
	"io"

	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lmerkleroot"
	"github.com/copernet/copernicus/model/consensus"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
)

type Block struct {
	Header        BlockHeader
	Txs           []*tx.Tx
	serializesize int
	Checked       bool
	encodeSize    int
}

const MinBlocksToKeep = int32(288)

func (bl *Block) GetBlockHeader() BlockHeader {
	return bl.Header
}

func (bl *Block) SetNull() {
	bl.Header.SetNull()
	bl.Txs = nil
}

func (bl *Block) Serialize(w io.Writer) error {
	log.Trace("block Serialize....")
	if err := bl.Header.Serialize(w); err != nil {
		return err
	}
	if err := util.WriteVarInt(w, uint64(len(bl.Txs))); err != nil {
		return err
	}
	for _, Tx := range bl.Txs {
		if err := Tx.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

func (bl *Block) SerializeSize() int {
	if bl.serializesize != 0 {
		return bl.serializesize
	}

	buf := bytes.NewBuffer(nil)
	bl.Serialize(buf)
	bl.serializesize = buf.Len()
	return bl.serializesize
}

func (bl *Block) Encode(w io.Writer) error {
	return bl.Serialize(w)
}

func (bl *Block) Decode(r io.Reader) error {
	return bl.Unserialize(r)
}

func (bl *Block) EncodeSize() int {
	if bl.encodeSize != 0 {
		return bl.encodeSize
	}
	return bl.SerializeSize()
}

func (bl *Block) Unserialize(r io.Reader) error {
	log.Trace("block Unserialize.... ")
	if err := bl.Header.Unserialize(r); err != nil {
		return err
	}
	ntx, err := util.ReadVarInt(r)
	if err != nil {
		return err
	}
	if ntx > consensus.MaxTxCount {
		return fmt.Errorf("recv %d transactions, but allow max %d", ntx, consensus.MaxTxCount)
	}
	bl.Txs = make([]*tx.Tx, ntx)
	for i := 0; i < int(ntx); i++ {
		tx := tx.NewTx(0, tx.DefaultVersion)
		if err := tx.Unserialize(r); err != nil {
			return err
		}
		bl.Txs[i] = tx
	}
	return nil
}

func (bl *Block) GetHash() util.Hash {
	bh := bl.Header
	return bh.GetHash()
}

func NewBlock() *Block {
	return &Block{}
}

func NewGenesisBlock() *Block {
	block := &Block{}
	block.Txs = []*tx.Tx{tx.NewGenesisCoinbaseTx()}
	block.Header = BlockHeader{
		Version:       1,
		HashPrevBlock: *util.HashFromString("0000000000000000000000000000000000000000000000000000000000000000"),
		//2009-01-03 18:15:05 +0000 UTC
		Time: 1231006505,
		//Time: uint32(1231006505),
		//486604799  [00000000ffff0000000000000000000000000000000000000000000000000000]
		Bits:  0x1d00ffff,
		Nonce: 2083236893,
	}
	block.Header.MerkleRoot = lmerkleroot.BlockMerkleRoot(block.Txs, nil)

	return block
}

func NewTestNetGenesisBlock() *Block {
	block := &Block{}
	block.Txs = []*tx.Tx{tx.NewGenesisCoinbaseTx()}
	block.Header = BlockHeader{
		Version:       1,
		HashPrevBlock: *util.HashFromString("0000000000000000000000000000000000000000000000000000000000000000"),
		Time:          1296688602,
		Bits:          0x1d00ffff,
		Nonce:         414098458,
	}
	block.Header.MerkleRoot = lmerkleroot.BlockMerkleRoot(block.Txs, nil)

	return block
}

func NewRegTestGenesisBlock() *Block {
	block := &Block{}
	block.Txs = []*tx.Tx{tx.NewGenesisCoinbaseTx()}
	block.Header = BlockHeader{
		Version:       1,
		HashPrevBlock: *util.HashFromString("0000000000000000000000000000000000000000000000000000000000000000"),
		Time:          1296688602,
		Bits:          0x207fffff,
		Nonce:         2,
	}
	block.Header.MerkleRoot = lmerkleroot.BlockMerkleRoot(block.Txs, nil)

	return block
}
