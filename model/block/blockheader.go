package block

import (
	"bytes"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/persist/db"
)

type BlockHeader struct {
	db.Serializer
	Version       int32
	HashPrevBlock util.Hash
	MerkleRoot    util.Hash
	Time          uint32
	Bits          uint32
	Nonce         uint32
}

const blockHeaderLength = 16 + util.Hash256Size*2

func NewBlockHeader() *BlockHeader {
	return &BlockHeader{}
}

func (bh *BlockHeader) IsNull() bool {
	return bh.Bits == 0
}

func (bh *BlockHeader) GetBlockTime() int64 {
	return int64(bh.Time)
}

func (bh *BlockHeader) GetHash() util.Hash {
	buf := bytes.NewBuffer(make([]byte, 0, blockHeaderLength))
	bh.SerializeHeader(buf)
	return crypto.DoubleSha256Hash(buf.Bytes())
}

func (bh *BlockHeader) SetNull() {
	*bh = BlockHeader{}
}

func (bh *BlockHeader) SerializeHeader(w io.Writer) error {
	return util.WriteElements(w, bh.Version, &bh.HashPrevBlock, &bh.MerkleRoot, bh.Time, bh.Bits, bh.Nonce)
}

func (bh *BlockHeader) UnserializeHeader(r io.Reader) error {
	return util.ReadElements(r, &bh.Version, &bh.HashPrevBlock, &bh.MerkleRoot, &bh.Time, &bh.Bits, &bh.Nonce)
}

func (bh *BlockHeader) String() string {
	return fmt.Sprintf("Block version : %d, hashPrevBlock : %s, hashMerkleRoot : %s,"+
		"Time : %d, Bits : %d, nonce : %d, BlockHash : %s\n", bh.Version, bh.HashPrevBlock,
		bh.MerkleRoot, bh.Time, bh.Bits, bh.Nonce, bh.GetHash())
}
func (bh *BlockHeader)GetSerializeList()[]string{
	dump_list := []string{"Version","HashPrevBlock", "MerkleRoot", "Time", "Bits","Nonce"}
	return dump_list
}