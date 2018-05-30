package block

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/util"
)

type Block struct {
	Header        BlockHeader
	Txs           []*tx.Tx
	serializesize int
	Checked       bool
	encodeSize    int
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

func (bl *Block) EncodeSize() int {
	if bl.encodeSize != 0 {
		return bl.encodeSize
	}
	return bl.SerializeSize()
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
	bl.Txs = make([]*tx.Tx, ntx)
	for i := 0; i < int(ntx); i++ {
		var txn tx.Tx
		txnp := &txn
		if err := txnp.Unserialize(r); err != nil {
			return err
		}
		bl.Txs[i] = txnp
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
