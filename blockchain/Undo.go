package blockchain

import (
	"errors"
	"fmt"
	"io"

	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

type DisconnectResult int

const (
	// DisconnectOk All good.
	DisconnectOk DisconnectResult = iota
	// DisconnectUnclean Rolled back, but UTXO set was inconsistent with block.
	DisconnectUnclean
	// DisconnectFailed Something else went wrong.
	DisconnectFailed
)

type TxUndo struct {
	prevout []*utxo.Coin
}

func (tu *TxUndo) Serialize(w io.Writer) error {
	err := utils.WriteVarInt(w, uint64(len(tu.prevout)))
	if err != nil {
		return err
	}
	for _, coin := range tu.prevout {
		err = coin.Serialize(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeserializeTxUndo(r io.Reader) (*TxUndo, error) {
	tu := &TxUndo{
		prevout: make([]*utxo.Coin, 0),
	}
	num, err := utils.ReadVarInt(r)
	if err != nil {
		return nil, err
	}
	for {
		coin, err := utxo.DeserializeCoin(r)
		if err == io.EOF {
			if len(tu.prevout) != int(num) {
				return nil, errors.New("the coins number incorrect")
			}
			return tu, io.EOF
		}
		if err != nil && err != io.EOF {
			return nil, err
		}
		tu.prevout = append(tu.prevout, coin)
	}
}

func UndoCoinSpend(coin *utxo.Coin, cache *utxo.CoinsViewCache, out *model.OutPoint) DisconnectResult {
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

func ApplyBlockUndo(undo *BlockUndo, block *model.Block, index *BlockIndex, cache *utxo.CoinsViewCache) DisconnectResult {
	clean := true
	if len(undo.txundo)+1 != len(block.Transactions) {
		fmt.Println("DisconnectBlock(): block and undo data inconsistent")
		return DisconnectFailed
	}

	// Undo transactions in reverse order.
	i := len(block.Transactions)
	for i > 0 {
		i--
		tx := block.Transactions[i]
		txid := tx.Hash

		// Check that all outputs are available and match the outputs in the
		// block itself exactly.
		for j := 0; j < len(tx.Outs); j++ {
			if tx.Outs[j].Script.IsUnspendable() {
				continue
			}

			out := model.NewOutPoint(txid, uint32(j))
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
			if len(txundo.prevout) != len(tx.Ins) {
				fmt.Println("DisconnectBlock(): transaction and undo data inconsistent")
				return DisconnectFailed
			}

			for k := len(tx.Ins); k > 0; {
				k--
				outpoint := tx.Ins[k].PreviousOutPoint
				c := txundo.prevout[k]
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

func UpdateCoins(tx *model.Tx, inputs *utxo.CoinsViewCache, undo *TxUndo, height int) {
	// Mark inputs spent.
	if !(tx.IsCoinBase()) {
		for _, txin := range tx.Ins {
			undo.prevout = append(undo.prevout, utxo.NewEmptyCoin())

			isSpent := inputs.SpendCoin(txin.PreviousOutPoint, undo.prevout[len(undo.prevout)-1])
			if !isSpent {
				panic("the coin is spent ..")
			}
		}
	}

	// Add outputs.
	utxo.AddCoins(*inputs, *tx, height)
}

func newTxUndo() *TxUndo {
	return &TxUndo{
		prevout: make([]*utxo.Coin, 0),
	}
}
