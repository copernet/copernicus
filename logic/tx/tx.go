package tx

import (
	"github.com/btcboost/copernicus/model/tx"


)

func CheckCoinBaseTransaction(tx *tx.Tx, state *ValidationState, allowLargeOpReturn bool) bool {
	return true
}

func CheckRegularTransaction(tx *tx.Tx, state *ValidationState, allowLargeOpReturn bool) bool {
	return true
}

