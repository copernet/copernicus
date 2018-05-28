package service

import (
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/mempool"
	lpool "github.com/btcboost/copernicus/logic/mempool"
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/util"
)

func ProcessTransaction(transaction *tx.Tx, nodeID int64) ([]*tx.Tx, []util.Hash, error) {
	err := ltx.CheckRegularTransaction(transaction)
	if err != nil {
		return nil, nil, err
	}

	utxoTip := utxo.GetUtxoCacheInstance()
	acceptTx := make([]*tx.Tx, 0)
	missTx := make([]util.Hash, 0)
	err = lpool.AcceptTxToMemPool(transaction)
	if err == nil {
		lpool.CheckMempool(utxoTip)
		acceptTx = append(acceptTx, transaction)
		acc := lpool.ProcessOrphan(transaction)
		if len(acc) > 0 {
			temAccept := make([]*tx.Tx, len(acc)+1)
			temAccept[0] = transaction
			copy(temAccept[1:], acc[:])
			return temAccept, missTx, nil
		}
		return acceptTx, missTx, nil
	}

	pool:= mempool.GetInstance()
	if errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut) {
		fRejectedParents := false
		for _, preOut := range transaction.GetAllPreviousOut() {
			if _, ok := pool.RecentRejects[preOut.Hash]; ok {
				fRejectedParents = true
				break
			}
		}
		if !fRejectedParents {
			for _, preOut := range transaction.GetAllPreviousOut() {
				missTx = append(missTx, preOut.Hash)
			}
			pool.AddOrphanTx(transaction, nodeID)
		}
		evicted := pool.LimitOrphanTx()
		if evicted > 0 {
			log.Print("service", "debug", "Orphan transaction " +
				"overflow, removed %d tx", evicted)
		}
	}

	pool.RecentRejects[transaction.GetHash()] = struct{}{}
	return acceptTx, missTx, err
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
