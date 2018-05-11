package block

import (
	"io"
	"unsafe"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/util"
)

type Block struct {
	Header BlockHeader
	Txs    []*tx.Tx
}

func (bl *Block) GetBlockHeader() BlockHeader {
	return bl.Header
}

func (bl *Block) SetNull() {
	bl.Header.SetNull()
	bl.Txs = nil
}

func (bl *Block) Serialize(w io.Writer) error {
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

func (bl *Block) Unserialize(r io.Reader) error {

	if err := bl.Header.Unserialize(r); err != nil {
		return err
	}

	return nil
}

func (bl *Block) SerializeSize() uint {
	size := uint(unsafe.Sizeof(BlockHeader{}))
	for _, Tx := range bl.Txs {
		size += Tx.SerializeSize()
	}
	return size
}

func NewBlock() *Block {
	return &Block{}
}