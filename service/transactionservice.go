package service

import (
	"github.com/btcboost/copernicus/logic/mempool"
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/tx"
)

func ProcessTransaction(transaction *tx.Tx) error {
	err := ltx.CheckRegularTransaction(transaction)
	if err != nil {
		return err
	}
	err := mempool.AccpetTxToMemPool(transaction)
	if err != nil {
		return err
	}
	return nil
}
