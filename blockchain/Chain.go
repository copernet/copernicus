package blockchain

import (
	"errors"

	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utxo"
)

func CheckChain(tx *model.Tx) (int, error) {
	TxInsLen := len(tx.Ins)
	var PreTxsOutTotalMoney int64
	for i := 0; i < TxInsLen; i++ {
		PreTx := mempool.GetTxFromMemPool(tx.Ins[i].PreviousOutPoint.Hash)
		if PreTx != nil {
			if PreTx.ValState == model.TX_ORPHAN {
				return model.TX_ORPHAN, errors.New("previous TX is orphan")
			}
			PreTxsOutTotalMoney += PreTx.Outs[tx.Ins[i].PreviousOutPoint.Index].Value
		} else {
			PreTx = utxo.GetTxFromUTXO(tx.Ins[i].PreviousOutPoint.Hash)
			if PreTx == nil {
				return model.TX_ORPHAN, errors.New("UTXO has no previous tx")
			}
			PreTxsOutTotalMoney += PreTx.Outs[tx.Ins[i].PreviousOutPoint.Index].Value
		}
	}
	TxOutsLen := len(tx.Outs)
	var TotalOutsMoney int64
	for j := 0; j < TxOutsLen; j++ {
		TotalOutsMoney += tx.Outs[j].Value
	}
	if PreTxsOutTotalMoney < TotalOutsMoney {
		return model.TX_INVALID, errors.New("Ins' money < outs' money")
	}

	for k := 0; k < TxInsLen; k++ {
		tx.Ins[k].Script.Eval()
	}
	return 0, nil
}
