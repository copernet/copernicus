package service

import (
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	lmempool "github.com/copernet/copernicus/logic/mempool"
	ltx "github.com/copernet/copernicus/logic/tx"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
)

func ProcessTransaction(transaction *tx.Tx, nodeID int64) ([]*tx.Tx, []util.Hash, error) {
	err := ltx.CheckRegularTransaction(transaction)
	if err != nil {
		return nil, nil, err
	}
	acceptTx := make([]*tx.Tx, 0)
	missTx := make([]util.Hash, 0)
	err = lmempool.AcceptTxToMemPool(transaction)
	if err == nil {
		lmempool.CheckMempool()
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
