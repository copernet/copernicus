package service

import (
	"github.com/btcboost/copernicus/errcode"
	lmempool "github.com/btcboost/copernicus/logic/mempool"
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"
)

//func ProcessTransaction(transaction *tx.Tx, nodeID int64) ([]*tx.Tx, error) {
//	pool:= mempool.GetInstance()
//	if _, ok := pool.RecentRejects[transaction.GetHash()]; ok{
//		return nil, errcode.New(errcode.RejectTx)
//	}
//
//	err := ltx.CheckRegularTransaction(transaction)
//	if err != nil {
//		return nil, err
//	}
//
//	//err := mempool.AccpetTxToMemPool(transaction)
//	//if err != nil {
//	//	return err
//	//}
//	return nil, nil
//}

func ProcessTransaction(transaction *tx.Tx, nodeID int64) ([]*tx.Tx, error) {
	gMempool := mempool.GetInstance()
	if _, ok := gMempool.RecentRejects[transaction.GetHash()]; ok {
		return nil, errcode.New(errcode.RejectTx)
	}

	err := ltx.CheckRegularTransaction(transaction)
	if err != nil {
		return nil, err
	}
	utxoTip := utxo.GetUtxoCacheInstance()

	acceptTx := make([]*tx.Tx, 0)
	err = lmempool.AcceptTxToMemPool(transaction)
	if err == nil {
		//bestHeight := utxoTip.GetBestBlock()
		lmempool.CheckMempool(utxoTip)
		acceptTx = append(acceptTx, transaction)
		acc := lmempool.ProcessOrphan(transaction)
		if len(acc) > 0 {
			temAccept := make([]*tx.Tx, len(acc)+1)
			temAccept[0] = transaction
			copy(temAccept[1:], acc[:])
			return temAccept, nil
		}
		return acceptTx, nil
	}

	//if errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut) {
	//	fRejectedParents := false
	//	for _, preOut := range tx.GetAllPreviousOut() {
	//		if _, ok := mempool.Gpool.RecentRejects[preOut.Hash]; ok {
	//			fRejectedParents = true
	//			break
	//		}
	//	}
	//	if !fRejectedParents {
	//		for _, preOut := range tx.GetAllPreviousOut() {
	//			//todo... require its parent transaction for all connect net node.
	//			_ = preOut
	//		}
	//		gMempool.AddOrphanTx(tx, nodeID)
	//	}
	//	evicted := gMempool.LimitOrphanTx()
	//	if evicted > 0 {
	//		//todo add log
	//		log.Debug("")
	//	}
	//} else {
	//	gMempool.RecentRejects[transaction.GetHash()] = struct{}{}
	//}

	return nil, err
}
