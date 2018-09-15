package service

import (
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
)

func handleRejectedTx(transaction *tx.Tx, err error, nodeID int64) (lostTx []util.Hash) {
	pool := mempool.GetInstance()
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
				lostTx = append(lostTx, preOut.Hash)
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

	return
}

func ProcessTransaction(transaction *tx.Tx,
	nodeID int64) ([]*tx.Tx, []util.Hash, error) {

	err := ltx.CheckRegularTransaction(transaction)
	if err != nil {
		lostTx := handleRejectedTx(transaction, err, nodeID)
		return nil, lostTx, err
	}
	acceptTx := make([]*tx.Tx, 0)
	err = lmempool.AcceptTxToMemPool(transaction)
	if err == nil {
		lostTx := make([]util.Hash, 0)
		lmempool.CheckMempool()
		acceptTx = append(acceptTx, transaction)
		acc := lmempool.ProcessOrphan(transaction)
		if len(acc) > 0 {
			temAccept := make([]*tx.Tx, len(acc)+1)
			temAccept[0] = transaction
			copy(temAccept[1:], acc[:])
			return temAccept, lostTx, nil
		}
		return acceptTx, nil, nil
	}
	lostTx := handleRejectedTx(transaction, err, nodeID)
	return acceptTx, lostTx, err
}
