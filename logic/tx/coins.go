package tx

import (
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/model/undo"
	"github.com/btcboost/copernicus/model/outpoint"
)

func UpdateCoins(tx *tx.Tx, coinMap *utxo.CoinsMap, txundo *undo.TxUndo, height uint32){
	if !tx.IsCoinBase(){
		undoCoins := make([]*utxo.Coin, 0, len(tx.GetIns()))
		for idx, txin := range tx.GetIns(){
			coin := coinMap.FetchCoin(txin.PreviousOutPoint)
			if coin == nil{
				panic("not coin find to spend!")
			}
			undoCoins[idx] = coin.DeepCopy()
			coinMap.SpendCoin(txin.PreviousOutPoint)
		}
		AddCoins(coinMap, tx, height)
	}
}

func AddCoins(coinMap *utxo.CoinsMap, tx *tx.Tx, height uint32){
	isCoinbase := tx.IsCoinBase()
	txid := tx.Hash
	for idx, out := range tx.GetOuts(){
		op := outpoint.NewOutPoint(txid, uint32(idx))
		coin := utxo.NewCoin(out, height,isCoinbase)
		coinMap.AddCoin(op, coin)
	}

}

