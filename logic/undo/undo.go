package undo

import (
	"fmt"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/undo"
	"github.com/btcboost/copernicus/model/utxo"
	"time"
)

// GuessVerificationProgress Guess how far we are in the verification process at the given block index
func GuessVerificationProgress(data *chainparams.ChainTxData, index *blockindex.BlockIndex) float64 {
	if index == nil {
		return float64(0)
	}

	now := time.Now()

	var txTotal float64
	// todo confirm time precise
	if int64(index.ChainTxCount) <= data.TxCount {
		txTotal = float64(data.TxCount) + (now.Sub(data.Time).Seconds())*data.TxRate
	} else {
		txTotal = float64(index.ChainTxCount) + float64(now.Second()-int(index.GetBlockTime()))*data.TxRate
	}
	return float64(index.ChainTxCount) / txTotal
}

// IsInitialBlockDownload Check whether we are doing an initial block download
// (synchronizing from disk or network)
func IsInitialBlockDownload() bool {
	return true
	//// Once this function has returned false, it must remain false.
	//gLatchToFalse.Store(false)
	//// Optimization: pre-test latch before taking the lock.
	//if gLatchToFalse.Load().(bool) {
	//	return false
	//}
	//
	//// todo !!! add cs_main sync.lock in here
	//if gLatchToFalse.Load().(bool) {
	//	return false
	//}
	//if GImporting.Load().(bool) || GfReindex {
	//	return true
	//}
	//if GChainState.ChainActive.Tip() == nil {
	//	return true
	//}
	//if GChainState.ChainActive.Tip().ChainWork.Cmp(&msg.ActiveNetParams.MinimumChainWork) < 0 {
	//	return true
	//}
	//if int64(GChainState.ChainActive.Tip().GetBlockTime()) < util.GetMockTime()-GMaxTipAge {
	//	return true
	//}
	//gLatchToFalse.Store(true)
	//
	//return false
}

func ApplyBlockUndo(blockUndo *undo.BlockUndo, blk *block.Block,
	cm *utxo.CoinsMap) undo.DisconnectResult {
	clean := true
	txUndos := blockUndo.GetTxundo()
	if len(txUndos)+1 != len(blk.Txs) {
		fmt.Println("DisconnectBlock(): block and undo data inconsistent")
		return undo.DisconnectFailed
	}
	// Undo transactions in reverse order.
	for i := len(blk.Txs) - 1; i > 0; i-- {
		tx := blk.Txs[i]
		txid := tx.GetHash()

		// Check that all outputs are available and match the outputs in the
		// block itself exactly.
		for j := 0; j < tx.GetOutsCount(); j++ {
			if tx.GetTxOut(j).IsSpendable() {
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
				log.Error("DisconnectBlock(): transaction and undo data inconsistent")
				return undo.DisconnectFailed
			}

			for k := insLen - 1; k > 0; k-- {
				outpoint := ins[k].PreviousOutPoint
				undoCoin := txundo.GetUndoCoins()[k]
				res := UndoCoinSpend(undoCoin, cm, outpoint)
				if res == undo.DisconnectFailed {
					return undo.DisconnectFailed
				}
				clean = clean && (res != undo.DisconnectUnclean)
			}
		}
	}

	if clean {
		return undo.DisconnectOk
	}
	return undo.DisconnectUnclean
}

func UndoCoinSpend(coin *utxo.Coin, cm *utxo.CoinsMap, out *outpoint.OutPoint) undo.DisconnectResult {
	clean := true
	if cm.FetchCoin(out) != nil {
		// Overwriting transaction output.
		clean = false
	}
	// delete this logic from core-abc
	//if coin.GetHeight() == 0 {
	//	// Missing undo metadata (height and coinbase). Older versions included
	//	// this information only in undo records for the last spend of a
	//	// transactions' outputs. This implies that it must be present for some
	//	// other output of the same tx.
	//	alternate := utxo.AccessByTxid(cache, &out.Hash)
	//	if alternate.IsSpent() {
	//		// Adding output for transaction without known metadata
	//		return DisconnectFailed
	//	}
	//
	//	// This is somewhat ugly, but hopefully utility is limited. This is only
	//	// useful when working from legacy on disck data. In any case, putting
	//	// the correct information in there doesn't hurt.
	//	coin = utxo.NewCoin(coin.GetTxOut(), alternate.GetHeight(), alternate.IsCoinBase())
	//}
	cm.AddCoin(out, coin)
	if clean {
		return undo.DisconnectOk
	}
	return undo.DisconnectUnclean
}
