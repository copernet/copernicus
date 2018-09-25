package lundo

import (
	"fmt"
	"sync/atomic"

	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/pow"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/util"
)

var latchToFalse int32

// IsInitialBlockDownload Check whether we are doing an initial block download
// (synchronizing from disk or network)
func IsInitialBlockDownload() bool {
	gChainActive := chain.GetInstance()
	// latchToFalse: pre-test latch before taking the lock.
	if atomic.LoadInt32(&latchToFalse) != 0 {
		return false
	}

	persist.CsMain.Lock()
	defer persist.CsMain.Unlock()

	if atomic.LoadInt32(&latchToFalse) != 0 {
		return false
	}
	if persist.Reindex {
		return true
	}
	if gChainActive.Tip() == nil {
		return true
	}
	minWorkSum := pow.HashToBig(&model.ActiveNetParams.MinimumChainWork)
	if gChainActive.Tip().ChainWork.Cmp(minWorkSum) < 0 {
		return true
	}

	if int64(gChainActive.Tip().GetBlockTime()) < util.GetTime()-persist.DefaultMaxTipAge {
		return true
	}
	atomic.AddInt32(&latchToFalse, 1)

	return false
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
				log.Error("DisconnectBlock(): transaction and undo data inconsistent")
				return undo.DisconnectFailed
			}

			for k := insLen - 1; k > 0; k-- {
				outpoint := ins[k].PreviousOutPoint
				undoCoin := txundo.GetUndoCoins()[k]
				res := CoinSpend(undoCoin, cm, outpoint)
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

//CoinSpend undo coin of spend
func CoinSpend(coin *utxo.Coin, cm *utxo.CoinsMap, out *outpoint.OutPoint) undo.DisconnectResult {
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
	cm.AddCoin(out, coin, coin.IsCoinBase())
	if clean {
		return undo.DisconnectOk
	}
	return undo.DisconnectUnclean
}
