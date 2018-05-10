package tx

import (
	"github.com/btcboost/copernicus/model/tx"


)

func CheckRegularTransaction(tx *tx.Tx, state *ValidationState, allowLargeOpReturn bool) bool {
	return true
}

func CheckBlockCoinBaseTransaction(tx *tx.Tx, state *ValidationState, allowLargeOpReturn bool) bool {
	return true
}

func CheckBlockRegularTransaction(tx *tx.Tx, state *ValidationState, allowLargeOpReturn bool) bool {
	return true
}

func SubmitTransaction(txs []*tx.Tx) bool {
	return true
}

func UndoTransaction(txs []*txundo.TxUndo) bool {
	return true
}
