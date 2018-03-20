package blockchain

import (
	"errors"

	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/utxo"
)

func TxCheckChain(tx *core.Tx) (int, error) {
	TxInsLen := len(tx.Ins)
	var PreTxsOutTotalMoney int64
	for i := 0; i < TxInsLen; i++ {
		PreTx := mempool.GetTxFromMemPool(tx.Ins[i].PreviousOutPoint.Hash)
		if PreTx != nil {
			if PreTx.ValState == core.TxOrphan {
				return core.TxOrphan, errors.New("previous TX is orphan")
			}
			PreTxsOutTotalMoney += PreTx.Outs[tx.Ins[i].PreviousOutPoint.Index].Value
		} else {
			PreTx = utxo.GetTxFromUTXO(tx.Ins[i].PreviousOutPoint.Hash)
			if PreTx == nil {
				return core.TxOrphan, errors.New("UTXO has no previous tx")
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
		return core.TxInvalid, errors.New("Ins' money < outs' money")
	}

	for k := 0; k < TxInsLen; k++ {
		ret, err := tx.Ins[k].Script.Eval()
		if ret != 0 {
			return ret, err
		}
	}
	return 0, nil
}
