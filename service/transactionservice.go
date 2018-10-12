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

func handleRejectedTx(transaction *tx.Tx, err error,
	nodeID int64) (lostTx []util.Hash) {

	pool := mempool.GetInstance()
	if errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut) {
		fRejectedParents := false
		prevouts := transaction.GetAllPreviousOut()
		for _, preOut := range prevouts {
			if _, ok := pool.RecentRejects[preOut.Hash]; ok {
				fRejectedParents = true
				break
			}
		}
		if !fRejectedParents {
			for _, preOut := range prevouts {
				lostTx = append(lostTx, preOut.Hash)
			}
			pool.AddOrphanTx(transaction, nodeID)
		}
		evicted := pool.LimitOrphanTx()
		if evicted > 0 {
			log.Print("service", "debug", "Orphan transaction "+
				"overflow, removed %d tx", evicted)
		}
	} else {
		pool.RecentRejects[transaction.GetHash()] = struct{}{}
	}

	return
}

func ProcessTransaction(transaction *tx.Tx, nodeID int64) ([]*tx.Tx, []util.Hash, error) {

	err := ltx.CheckRegularTransaction(transaction)
	if err != nil {
		lostTx := handleRejectedTx(transaction, err, nodeID)
		return nil, lostTx, err
	}

	err = lmempool.AcceptTxToMemPool(transaction)
	if err == nil {
		lmempool.CheckMempool()
		// Firstly, place the first accepted @transaction for a peer,
		// Otherwise, it may discard other orphan transactions.
		return append([]*tx.Tx{transaction}, lmempool.ProcessOrphan(transaction)...), nil, nil
	}
	lostTx := handleRejectedTx(transaction, err, nodeID)
	return nil, lostTx, err
}
