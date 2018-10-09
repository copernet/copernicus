package lundo

import (
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
)

func ApplyBlockUndo(blockUndo *undo.BlockUndo, blk *block.Block,
	cm *utxo.CoinsMap) undo.DisconnectResult {
	clean := true
	txUndos := blockUndo.GetTxundo()
	if len(txUndos)+1 != len(blk.Txs) {
		log.Error("DisconnectBlock(): block(%d) and undo(%d) data inconsistent", len(txUndos)+1, len(blk.Txs))
		return undo.DisconnectFailed
	}
	// Undo transactions in reverse order.
	for i := len(blk.Txs) - 1; i > 0; i-- {
		tx := blk.Txs[i]
		txid := tx.GetHash()

		// Check that all outputs are available and match the outputs in the
		// block itself exactly.
		for j := 0; j < tx.GetOutsCount(); j++ {
			if !tx.GetTxOut(j).IsSpendable() {
				continue
			}
			out := outpoint.NewOutPoint(txid, uint32(j))
			coin := cm.SpendGlobalCoin(out)
			coinOut := coin.GetTxOut()
			if coin == nil || !tx.GetTxOut(j).IsEqual(&coinOut) {
				// transaction output mismatch
				clean = false
			}

			// Restore inputs
			if i < 1 {
				// Skip the coinbase
				break
			}

			txundo := txUndos[i-1]
			ins := tx.GetIns()
			insLen := len(ins)
			if len(txundo.GetUndoCoins()) != insLen {
				log.Error("DisconnectBlock(): transaction(%d) and undo data(%d) inconsistent", len(txundo.GetUndoCoins()), insLen)
				return undo.DisconnectFailed
			}

			for k := insLen - 1; k > 0; k-- {
				outPoint := ins[k].PreviousOutPoint
				undoCoin := txundo.GetUndoCoins()[k]
				res := CoinSpend(undoCoin, cm, outPoint)
				if res == undo.DisconnectFailed {
					log.Error("DisconnectBlock(): the undo coin(%+v) had spend, the outpoint is:%+v", undoCoin, outPoint)
					return undo.DisconnectFailed
				}
				clean = clean && (res != undo.DisconnectUnclean)
			}
		}
	}

	if clean {
		log.Debug("DisconnectBlock(): disconnect block success.")
		return undo.DisconnectOk
	}
	return undo.DisconnectUnclean
}

//CoinSpend undo coin of spend
func CoinSpend(coin *utxo.Coin, cm *utxo.CoinsMap, out *outpoint.OutPoint) undo.DisconnectResult {
	clean := true
	if cm.FetchCoin(out) != nil {
		// Overwriting transaction output.
		clean = false
	}
	cm.AddCoin(out, coin, coin.IsCoinBase())
	if clean {
		log.Debug("CoinSpend(): disconnect block success.")
		return undo.DisconnectOk
	}
	return undo.DisconnectUnclean
}
