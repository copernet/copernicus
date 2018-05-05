package blockchain

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

const MaxInputPerTx = core.MaxTxInPerMessage

type DisconnectResult int

const (
	// DisconnectOk All good.
	DisconnectOk DisconnectResult = iota
	// DisconnectUnclean Rolled back, but UTXO set was inconsistent with block.
	DisconnectUnclean
	// DisconnectFailed Something else went wrong.
	DisconnectFailed
)

type TxInUndo struct {
}

type TxUndo struct {
	PrevOut []*utxo.Coin
}

func (tu *TxUndo) Serialize(w io.Writer) error {
	err := utils.WriteVarInt(w, uint64(len(tu.PrevOut)))
	if err != nil {
		return err
	}
	for _, coin := range tu.PrevOut {
		err = coin.Serialize(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeserializeTxUndo(r io.Reader) (*TxUndo, error) {
	tu := &TxUndo{
		PrevOut: make([]*utxo.Coin, 0),
	}
	utils.BinarySerializer.Uint64(r, binary.LittleEndian)
	var count int
	for {
		coin, err := utxo.DeserializeCoin(r)
		if err == io.EOF {
			return tu, io.EOF
		}
		if err != nil && err != io.EOF {
			return nil, err
		}
		count++
		if count > MaxInputPerTx {
			panic("Too many input undo records")
		}
		tu.PrevOut = append(tu.PrevOut, coin)
	}
}

func UndoCoinSpend(coin *utxo.Coin, cache *utxo.CoinsViewCache, out *core.OutPoint) DisconnectResult {
	clean := true
	if cache.HaveCoin(out) {
		// Overwriting transaction output.
		clean = false
	}
	if coin.GetHeight() == 0 {
		// Missing undo metadata (height and coinbase). Older versions included
		// this information only in undo records for the last spend of a
		// transactions' outputs. This implies that it must be present for some
		// other output of the same tx.
		alternate := utxo.AccessByTxid(cache, &out.Hash)
		if alternate.IsSpent() {
			// Adding output for transaction without known metadata
			return DisconnectFailed
		}

		// This is somewhat ugly, but hopefully utility is limited. This is only
		// useful when working from legacy on disck data. In any case, putting
		// the correct information in there doesn't hurt.
		coin = utxo.NewCoin(coin.TxOut, alternate.GetHeight(), alternate.IsCoinBase())
	}
	cache.AddCoin(out, *coin, coin.IsCoinBase())
	if clean {
		return DisconnectOk
	}
	return DisconnectUnclean
}

type BlockUndo struct {
	txundo []*TxUndo
}

func NewBlockUndo() *BlockUndo {
	return &BlockUndo{
		txundo: make([]*TxUndo, 0),
	}
}

func (bu *BlockUndo) Serialize(w io.Writer) error {
	var err error
	for _, txundo := range bu.txundo {
		err = txundo.Serialize(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeserializeBlockUndo(r io.Reader) (*BlockUndo, error) {
	bu := &BlockUndo{
		txundo: make([]*TxUndo, 0),
	}
	for {
		tu, err := DeserializeTxUndo(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		bu.txundo = append(bu.txundo, tu)
	}
	return bu, nil
}

func ApplyBlockUndo(undo *BlockUndo, block *core.Block, index *core.BlockIndex,
	cache *utxo.CoinsViewCache) DisconnectResult {
	clean := true
	if len(undo.txundo)+1 != len(block.Txs) {
		fmt.Println("DisconnectBlock(): block and undo data inconsistent")
		return DisconnectFailed
	}

	// Undo transactions in reverse order.
	i := len(block.Txs)
	for i > 0 {
		i--
		tx := block.Txs[i]
		txid := tx.Hash

		// Check that all outputs are available and match the outputs in the
		// block itself exactly.
		for j := 0; j < len(tx.Outs); j++ {
			if tx.Outs[j].Script.IsUnspendable() {
				continue
			}

			out := core.NewOutPoint(txid, uint32(j))
			coin := utxo.NewEmptyCoin()
			isSpent := cache.SpendCoin(out, coin)
			if !isSpent || tx.Outs[0] != coin.TxOut {
				// transaction output mismatch
				clean = false
			}

			// Restore inputs
			if i < 1 {
				// Skip the coinbase
				continue
			}

			txundo := undo.txundo[i-1]
			if len(txundo.PrevOut) != len(tx.Ins) {
				fmt.Println("DisconnectBlock(): transaction and undo data inconsistent")
				return DisconnectFailed
			}

			for k := len(tx.Ins); k > 0; {
				k--
				outpoint := tx.Ins[k].PreviousOutPoint
				c := txundo.PrevOut[k]
				res := UndoCoinSpend(c, cache, outpoint)
				if res == DisconnectFailed {
					return DisconnectFailed
				}
				clean = clean && (res != DisconnectUnclean)
			}
		}
	}

	// Move best block pointer to previous block.
	cache.SetBestBlock(block.BlockHeader.HashPrevBlock)

	if clean {
		return DisconnectOk
	}
	return DisconnectUnclean
}

func newTxUndo() *TxUndo {
	return &TxUndo{
		PrevOut: make([]*utxo.Coin, 0),
	}
}
