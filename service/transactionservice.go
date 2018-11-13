package service

import (
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
)

func HandleRejectedTx(txn *tx.Tx, err error, nodeID int64, recentRejects map[util.Hash]struct{}) (missTxs []util.Hash, rejectTxs []util.Hash) {
	missingInputs := errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut)
	isNormalOrphan := missingInputs && !txn.AnyInputTxIn(recentRejects)

	if isNormalOrphan {
		mempool.GetInstance().AddOrphanTx(txn, nodeID)
		missTxs = txn.PrevoutHashs()
		return
	}

	rejectTxs = append(rejectTxs, txn.GetHash())
	return
}

func ProcessTransaction(txn *tx.Tx, recentRejects map[util.Hash]struct{}, nodeID int64) ([]*tx.Tx, []util.Hash, []util.Hash, error) {
	err := lmempool.AcceptTxToMemPool(txn)
	if err == nil {
		lmempool.CheckMempool(chain.GetInstance().Height())
		acceptedOrphans, rejectTxs := lmempool.TryAcceptOrphansTxs(txn, chain.GetInstance().Height(), true)
		pool := mempool.GetInstance()
		if !pool.HaveTransaction(txn) {
			log.Error("the tx not exist mempool")
			return nil, nil, nil, err
		}

		acceptedTxs := append([]*tx.Tx{txn}, acceptedOrphans...)
		return acceptedTxs, nil, rejectTxs, nil
	}

	missTxs, rejectTxs := HandleRejectedTx(txn, err, nodeID, recentRejects)
	return nil, missTxs, rejectTxs, err
}
