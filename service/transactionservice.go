package service

import (
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/tx"
)

func ProcessTransaction(transaction *tx.Tx) error {
	err := ltx.CheckRegularTransaction(transaction)
	if err != nil {
		return err
	}
	//err := mempool.AccpetTxToMemPool(transaction)
	//if err != nil {
	//	return err
	//}
	return nil
}

//
//func ProcessTransaction(tx *tx.Tx, nodeID int64) ([]*tx.Tx, error) {
//	if _, ok := mempool.Gpool.RecentRejects[tx.GetHash()]; ok {
//		return nil, errcode.New(errcode.RejectTx)
//	}
//
//	var err error
//	utxoTip := utxo.GetUtxoCacheInstance()
//	acceptTx := make([]*tx.Tx, 0)
//	err = mempool.AccpetTxToMemPool(tx)
//	if err == nil {
//		//bestHeight := utxoTip.GetBestBlock()
//		mempool.Gpool.Check(utxoTip, 0)
//		acceptTx = append(acceptTx, tx)
//		acc := ProcessOrphan(tx)
//		if len(acc) > 0 {
//			temAccept := make([]*tx.Tx, len(acc)+1)
//			temAccept[0] = tx
//			copy(temAccept[1:], acc[:])
//			return temAccept, nil
//		}
//		return acceptTx, nil
//	}
//
//	if errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut) {
//		fRejectedParents := false
//		for _, preOut := range tx.GetAllPreviousOut() {
//			if _, ok := mempool.Gpool.RecentRejects[preOut.Hash]; ok {
//				fRejectedParents = true
//				break
//			}
//		}
//		if !fRejectedParents {
//			for _, preOut := range tx.GetAllPreviousOut() {
//				//todo... require its parent transaction for all connect net node.
//				_ = preOut
//			}
//			mempool.Gpool.AddOrphanTx(tx, nodeID)
//		}
//		evicted := mempool.Gpool.LimitOrphanTx()
//		if evicted > 0 {
//			//todo add log
//			log.Debug("")
//		}
//	} else {
//		mempool.Gpool.RecentRejects[tx.GetHash()] = struct{}{}
//	}
//
//	return nil, err
//}
