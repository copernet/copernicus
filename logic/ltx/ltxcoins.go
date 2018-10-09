package ltx

import (
	"fmt"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/undo"
	"github.com/copernet/copernicus/model/utxo"
)

//UpdateTxCoins update coins about tx
func UpdateTxCoins(tx *tx.Tx, coinMap *utxo.CoinsMap, txundo *undo.TxUndo, height int32) {
	txHash := tx.GetHash()
	if !tx.IsCoinBase() {
		undoCoins := make([]*utxo.Coin, len(tx.GetIns()))
		for idx, txin := range tx.GetIns() {
			coin := coinMap.AccessCoin(txin.PreviousOutPoint)
			undoCoins[idx] = coin.DeepCopy()
			coinMap.SpendCoin(txin.PreviousOutPoint)
		}
		if txHash.String() == "7e621eeb02874ab039a8566fd36f4591e65eca65313875221842c53de6907d6c" {
			fmt.Println("after spend coin*************************")
			utxo.DisplayCoinMap(coinMap)
		}
		txundo.SetUndoCoins(undoCoins)
	}
	txAddCoins(tx, coinMap, height)
	if txHash.String() == "7e621eeb02874ab039a8566fd36f4591e65eca65313875221842c53de6907d6c" {
		fmt.Println("after add coin*************************")
		utxo.DisplayCoinMap(coinMap)
	}
}

func txAddCoins(tx *tx.Tx, coinMap *utxo.CoinsMap, height int32) {
	isCoinbase := tx.IsCoinBase()
	txid := tx.GetHash()
	for idx, out := range tx.GetOuts() {
		op := outpoint.NewOutPoint(txid, uint32(idx))
		coin := utxo.NewCoin(out, height, isCoinbase)
		coinMap.AddCoin(op, coin, isCoinbase)
	}
}
