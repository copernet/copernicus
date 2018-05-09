package block

import (
	"bytes"
	"io"
	"fmt"
)

type BlockHeader struct {
	Version       int32
	HashPrevBlock utils.Hash
	MerkleRoot    utils.Hash
	Time          uint32
	Bits          uint32
	Nonce         uint32
}

const blockHeaderLength = 16 + utils.Hash256Size*2

func NewBlockHeader() *BlockHeader {
	return &BlockHeader{}
}

func (bh *BlockHeader) IsNull() bool {
	return bh.Bits == 0
}

func (bh *BlockHeader) GetBlockTime() int64 {
	return int64(bh.Time)
}

func (bh *BlockHeader) GetHash() utils.Hash {
	buf := bytes.NewBuffer(make([]byte, 0, blockHeaderLength))
	bh.Serialize(buf)
	return crypto.DoubleSha256Hash(buf.Bytes())
}

func (bh *BlockHeader) SetNull() {
	*bh = BlockHeader{}
}

func (bh *BlockHeader) Serialize(w io.Writer) error {
	return utils.WriteElements(w, bh.Version, &bh.HashPrevBlock, &bh.MerkleRoot, bh.Time, bh.Bits, bh.Nonce)
}

func (bh *BlockHeader) Deserialize(r io.Reader) error {
	return utils.ReadElements(r, &bh.Version, &bh.HashPrevBlock, &bh.MerkleRoot, &bh.Time, &bh.Bits, &bh.Nonce)
}

func (bh *BlockHeader) String() string {
	return fmt.Sprintf("Block version : %d, hashPrevBlock : %s, hashMerkleRoot : %s,"+
		"Time : %d, Bits : %d, nonce : %d, BlockHash : %s\n", bh.Version, bh.HashPrevBlock,
		bh.MerkleRoot, bh.Time, bh.Bits, bh.Nonce, bh.GetHash())
}