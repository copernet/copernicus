package tx

import (
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
)

//UpdateCoins update coins about tx
func UpdateCoins(tx *tx.Tx, coinMap *utxo.CoinsMap, txundo *undo.TxUndo, height int32) {
	if !tx.IsCoinBase() {
		undoCoins := make([]*utxo.Coin, len(tx.GetIns()), len(tx.GetIns()))
		for idx, txin := range tx.GetIns() {
			coin := coinMap.GetCoin(txin.PreviousOutPoint)
			if coin == nil {
				panic("no coin find to spend!")
			}
			undoCoins[idx] = coin.DeepCopy()
			coinMap.SpendCoin(txin.PreviousOutPoint)
		}
		txundo.SetUndoCoins(undoCoins)
	}
	txAddCoins(tx, coinMap, height)
}

func txAddCoins(tx *tx.Tx, coinMap *utxo.CoinsMap, height int32) {
	isCoinbase := tx.IsCoinBase()
	txid := tx.GetHash()
	for idx, out := range tx.GetOuts() {
		op := outpoint.NewOutPoint(txid, uint32(idx))
		coin := utxo.NewCoin(out, height, isCoinbase)
		coinMap.AddCoin(op, coin)
	}
}
