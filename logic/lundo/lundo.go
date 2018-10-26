package lundo

import (
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
)

func ApplyBlockUndo(blockUndo *undo.BlockUndo, blk *block.Block, cm *utxo.CoinsMap) undo.DisconnectResult {
	clean := true

	if !checkUndoData(blockUndo, blk) {
		return undo.DisconnectFailed
	}

	// Undo transactions in reverse order.
	for i := len(blk.Txs) - 1; i >= 0; i-- {
		tx := blk.Txs[i]

		if !clearTxOuts(tx, cm) {
			clean = false
		}

		if i < 1 {
			continue // Skip the coinbase
		}

		if !restoreTxInputs(tx, blockUndo.GetTxundo()[i-1], cm) {
			clean = false
		}
	}

	if !clean {
		log.Error("ApplyBlockUndo unclean, block: %s", blk.GetHash())
		return undo.DisconnectUnclean
	}

	log.Debug("ApplyBlockUndo: success. block: %s", blk.GetHash())
	return undo.DisconnectOk
}

func clearTxOuts(tx *tx.Tx, cm *utxo.CoinsMap) bool {
	clean := true

	for j := 0; j < tx.GetOutsCount(); j++ {
		if !tx.GetTxOut(j).IsSpendable() {
			continue
		}

		coin := cm.SpendGlobalCoin(outpoint.NewOutPoint(tx.GetHash(), uint32(j)))

		if coin == nil {
			clean = false
		} else if coinOut := coin.GetTxOut(); !tx.GetTxOut(j).IsEqual(&coinOut) {
			clean = false
		}
	}

	return clean
}

func restoreTxInputs(tx *tx.Tx, txundo *undo.TxUndo, cm *utxo.CoinsMap) bool {
	clean := true
	ins := tx.GetIns()

	for k := len(ins) - 1; k >= 0; k-- {
		coin := txundo.GetUndoCoins()[k]
		out := ins[k].PreviousOutPoint

		if cm.FetchCoin(out) != nil {
			clean = false // Overwriting transaction output.
		}

		cm.AddCoin(out, coin, coin.IsCoinBase())
	}

	return clean
}

func checkUndoData(blockUndo *undo.BlockUndo, blk *block.Block) bool {
	txUndos := blockUndo.GetTxundo()

	if len(txUndos)+1 != len(blk.Txs) {
		log.Error("checkUndoData: block(%d) and undo(%d) data inconsistent", len(blk.Txs), len(txUndos)+1)
		return false
	}

	for i := 1; i < len(blk.Txs); i++ {
		txundo := txUndos[i-1]
		insLen := len(blk.Txs[i].GetIns())

		if len(txundo.GetUndoCoins()) != insLen {
			log.Error("checkUndoData: tx(%d) and undo data(%d) inconsistent", len(txundo.GetUndoCoins()), insLen)
			return false
		}
	}

	return true
}
