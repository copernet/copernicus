package lundo

import (
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
)

func ApplyBlockUndo(blockUndo *undo.BlockUndo, blk *block.Block, cm *utxo.CoinsMap, height int32) undo.DisconnectResult {
	clean := true
	txUndos := blockUndo.GetTxundo()
	//check block undo
	if len(txUndos)+1 != len(blk.Txs) {
		log.Error("checkUndoData: block(%d) and undo(%d) data inconsistent", len(blk.Txs), len(txUndos)+1)
		return undo.DisconnectFailed
	}

	// First, restore inputs.
	for i := 1; i < len(blk.Txs); i++ {
		ptx := blk.Txs[i]
		txundo := txUndos[i-1]
		insLen := len(ptx.GetIns())

		if len(txundo.GetUndoCoins()) != insLen {
			log.Error("checkUndoData: tx(%d) and undo data(%d) inconsistent", len(txundo.GetUndoCoins()), insLen)
			return undo.DisconnectFailed
		}

		for j := 0; i < insLen; j++ {
			res := undoCoinSpend(ptx, txundo, cm)
			if res == undo.DisconnectFailed {
				return undo.DisconnectFailed
			}
			clean = clean && res != undo.DisconnectUnclean
		}
	}

	// Second, revert created outputs.
	for _, ptx := range blk.Txs {
		txID := ptx.GetHash()
		isCoinBase := ptx.IsCoinBase()

		// Check that all outputs are available and match the outputs in the
		// block itself exactly.
		for j := 0; j < ptx.GetOutsCount(); j++ {
			if !ptx.GetTxOut(j).IsSpendable() {
				continue
			}
			coin := cm.SpendGlobalCoin(outpoint.NewOutPoint(txID, uint32(j)))
			coinOut := coin.GetTxOut()
			if coin != nil || !ptx.GetTxOut(j).IsEqual(&coinOut) ||
				isCoinBase != coin.IsCoinBase() || height != coin.GetHeight() {
				// transaction output mismatch
				clean = false
			}
		}
	}

	cm.Flush(blk.GetBlockHeader().HashPrevBlock)
	if clean {
		return undo.DisconnectOk
	}
	log.Debug("ApplyBlockUndo: success. block: %s", blk.GetHash())
	return undo.DisconnectOk
}

func undoCoinSpend(tx *tx.Tx, txundo *undo.TxUndo, cm *utxo.CoinsMap) undo.DisconnectResult {
	clean := true
	ins := tx.GetIns()
	for k := len(ins) - 1; k >= 0; k-- {
		out := ins[k].PreviousOutPoint
		if cm.FetchCoin(out) != nil {
			// Overwriting transaction output.
			clean = false
		}

		coin := txundo.GetUndoCoins()[k]
		if coin.GetHeight() == 0 {
			// Missing undo metadata (height and coinbase). Older versions included
			// this information only in undo records for the last spend of a
			// transactions' outputs. This implies that it must be present for some
			// other output of the same tx.
			alternate := cm.AccessCoin(out)
			if alternate.IsSpent() {
				// Adding output for transaction without known metadata
				return undo.DisconnectFailed
			}
			// The potential_overwrite parameter to AddCoin is only allowed to be false
			// if we know for sure that the coin did not already exist in the cache. As
			// we have queried for that above using HaveCoin, we don't need to guess.
			// When fClean is false, a coin already existed and it is an overwrite.
			txOut := coin.GetTxOut()
			coin = utxo.NewFreshCoin(&txOut, alternate.GetHeight(), alternate.IsCoinBase())
		}
		// The potential_overwrite parameter to AddCoin is only allowed to be false
		// if we know for sure that the coin did not already exist in the cache. As
		// we have queried for that above using HaveCoin, we don't need to guess.
		// When fClean is false, a coin already existed and it is an overwrite.
		cm.AddCoin(out, coin, !clean)
	}
	if clean {
		return undo.DisconnectOk
	}
	return undo.DisconnectFailed
}
