package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/btcboost/copernicus/consensus"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/utils"
)

type Block struct {
	Header BlockHeader
	Txs    []*Tx
}

func (bl *Block) GetBlockHeader() BlockHeader {
	return bl.Header
}

func (bl *Block) SetNull() {
	bl.Header.SetNull()
	bl.Txs = nil
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
	if err := bl.Header.Serialize(w); err != nil {
		return err
	}
	if err := utils.WriteVarInt(w, uint64(len(bl.Txs))); err != nil {
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
	if err := bl.Header.Deserialize(r); err != nil {
		return err
	}
	ntx, err := utils.ReadVarInt(r)
	if err != nil {
		return err
	}
	if ntx > consensus.MaxTxCount {
		return errors.New(fmt.Sprintf("recv %d transactions, but allow max %d", ntx, consensus.MaxTxCount))
	}
	for i := 0; i < ntx; i++ {
		if tx, err := DeserializeTx(r); err != nil {
			return err
		}
		bl.Txs = append(bl.Txs, tx)
	}
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
	return &Block{}
}
