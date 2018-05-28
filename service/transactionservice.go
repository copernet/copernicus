package service

import (
	"github.com/btcboost/copernicus/errcode"
	"github.com/btcboost/copernicus/log"
	lmempool "github.com/btcboost/copernicus/logic/mempool"
	ltx "github.com/btcboost/copernicus/logic/tx"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/util"
)

func ProcessTransaction(transaction *tx.Tx, nodeID int64) ([]*tx.Tx, []util.Hash, error) {
	pool := mempool.GetInstance()
	if _, ok := pool.RecentRejects[transaction.GetHash()]; ok {
		return nil, nil, errcode.New(errcode.RejectTx)
	}

	err := ltx.CheckRegularTransaction(transaction)
	if err != nil {
		return nil, nil, err
	}

	utxoTip := utxo.GetUtxoCacheInstance()
	acceptTx := make([]*tx.Tx, 0)
	missTx := make([]util.Hash, 0)
	err = lmempool.AcceptTxToMemPool(transaction)
	if err == nil {
		lmempool.CheckMempool(utxoTip)
		acceptTx = append(acceptTx, transaction)
		acc := lmempool.ProcessOrphan(transaction)
		if len(acc) > 0 {
			temAccept := make([]*tx.Tx, len(acc)+1)
			temAccept[0] = transaction
			copy(temAccept[1:], acc[:])
			return temAccept, missTx, nil
		}
		return acceptTx, missTx, nil
	}

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
			log.Print("service", "debug", "Orphan transaction "+
				"overflow, removed %d tx", evicted)
		}
	}

	pool.RecentRejects[transaction.GetHash()] = struct{}{}
	return acceptTx, missTx, err
}
