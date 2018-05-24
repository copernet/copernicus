package block

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/util"
)

type Block struct {
	Header BlockHeader
	Txs    []*tx.Tx
	size    uint
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
	ntx, err := util.ReadVarInt(r)
	if err != nil {
		return err
	}
	if ntx > consensus.MaxTxCount {
		return errors.New(fmt.Sprintf("recv %d transactions, but allow max %d", ntx, consensus.MaxTxCount))
	}
	txns := make([]tx.Tx, ntx)
	bl.Txs = make([]*tx.Tx, ntx)
	for i := 0; i < int(ntx); i++ {
		if err := txns[i].Unserialize(r); err != nil {
			return err
		}
		bl.Txs[i] = &txns[i]
	}
	return nil
}

func (blk *Block) GetHash() *util.Hash {
	buf := bytes.NewBuffer(nil)
	err := blk.Serialize(buf)
	if err != nil {
		log.Error("GetHash is err=====%#v", err)
		panic("GetHash is err=====")
	}
	h := util.Sha256Hash(buf.Bytes())
	return &h
}
func (bl *Block) SerializeSize() uint {
	if bl.size !=0{
		return bl.size
	}
	size := uint(unsafe.Sizeof(BlockHeader{}))
	for _, Tx := range bl.Txs {
		size += Tx.SerializeSize()
	}
	bl.size = size
	return size
}

func NewBlock() *Block {
	return &Block{}
}
