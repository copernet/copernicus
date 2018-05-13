package tx

import (
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"

	"github.com/btcboost/copernicus/util"
)

func CheckRegularTransaction(tx *tx.Tx, allowLargeOpReturn bool) bool {
	return true
}

func CheckBlockCoinBaseTransaction(tx *tx.Tx, allowLargeOpReturn bool) bool {
	return true
}

func CheckBlockRegularTransaction(tx *tx.Tx, allowLargeOpReturn bool) bool {
	tempCoinsMap :=  utxo.NewEmptyCoinsMap()

	if !checkInputs(tx, tempCoinsMap, 1) {
		return false
	}

	return true
}

func SubmitTransaction(txs []*tx.Tx) bool {
	return true
}
/*
func UndoTransaction(txs []*txundo.TxUndo) bool {
	return true
}
*/

func checkInputs(tx *tx.Tx, tempCoinMap *utxo.CoinsMap, flag int) bool {
	ins := tx.GetIns()
	for _, in := range ins {
		coin := tempCoinMap.GetCoin(in.PreviousOutPoint)
		if coin == nil {
			return false
		}
		scriptPubKey := coin.GetTxOut().GetScriptPubKey()
		scriptSig := in.GetScriptSig()
		stack := util.NewStack()
		scriptSig.Eval(stack)
		scriptPubKey.Eval(stack)

	}
	return true
}