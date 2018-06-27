package block

import (
	"bytes"
	"fmt"
	"io"

	"github.com/copernet/copernicus/util"
)

type BlockHeader struct {
	Version       int32     `json:"version"`
	HashPrevBlock util.Hash `json:"previousblockhash, string"`
	MerkleRoot    util.Hash `json:"merkleroot, string"`
	Time          uint32    `json:"time"`
	Bits          uint32    `json:"bits"`
	Nonce         uint32    `json:"nonce"`
	encodeSize    int
	serializeSize int
	hash          util.Hash `json:"hash"`
}

const blockHeaderLength = 16 + util.Hash256Size*2

func NewBlockHeader() *BlockHeader {
	return &BlockHeader{}
}

func (bh *BlockHeader) IsNull() bool {
	return bh.Bits == 0
}

func (bh *BlockHeader) GetBlockTime() uint32 {
	return bh.Time
}

func (bh *BlockHeader) GetHash() util.Hash {
	if !bh.hash.IsNull() {
		return bh.hash
	}
	buf := bytes.NewBuffer(make([]byte, 0, blockHeaderLength))
	bh.SerializeHeader(buf)
	bh.hash = util.DoubleSha256Hash(buf.Bytes())
	return bh.hash
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

func (bh *BlockHeader) Encode(w io.Writer) error {
	return bh.Serialize(w)
}

func (bh *BlockHeader) EncodeSize() int {
	if bh.encodeSize > 0 {
		return bh.encodeSize
	}
	buf := bytes.NewBuffer(nil)
	bh.Encode(buf)
	bh.encodeSize = buf.Len()
	return bh.encodeSize
}
func (bh *BlockHeader) SerializeSize() int {
	if bh.serializeSize > 0 {
		return bh.serializeSize
	}
	buf := bytes.NewBuffer(nil)
	bh.Serialize(buf)
	bh.serializeSize = buf.Len()
	return bh.serializeSize
}
func (bh *BlockHeader) Decode(r io.Reader) error {
	return bh.UnserializeHeader(r)
}

func (bh *BlockHeader) Serialize(w io.Writer) error {
	return util.WriteElements(w, bh.Version, &bh.HashPrevBlock, &bh.MerkleRoot, bh.Time, bh.Bits, bh.Nonce)
}

func (bh *BlockHeader) Unserialize(r io.Reader) error {
	return util.ReadElements(r, &bh.Version, &bh.HashPrevBlock, &bh.MerkleRoot, &bh.Time, &bh.Bits, &bh.Nonce)
}

func (bh *BlockHeader) String() string {
	return fmt.Sprintf("Block version : %d, hashPrevBlock : %s, hashMerkleRoot : %s,"+
		"Time : %d, Bits : %d, nonce : %d, BlockHash : %s\n", bh.Version, bh.HashPrevBlock,
		bh.MerkleRoot, bh.Time, bh.Bits, bh.Nonce, bh.GetHash())
}
func (bh *BlockHeader) GetSerializeList() []string {
	dump_list := []string{"Version", "HashPrevBlock", "MerkleRoot", "Time", "Bits", "Nonce"}
	return dump_list
}
