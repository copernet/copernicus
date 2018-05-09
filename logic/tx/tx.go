package tx

import "github.com/btcboost/copernicus/model/tx"

type Tx struct {
	tx tx.Tx
}

func (tx *Tx) CheckRegularTransaction(state *ValidationState, allowLargeOpReturn bool) bool {

}