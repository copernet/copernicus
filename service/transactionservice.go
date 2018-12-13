package service

import (
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
)

func ProcessTransaction(txn *tx.Tx, recentRejects map[util.Hash]struct{}, nodeID int64) ([]*tx.Tx, []util.Hash, []util.Hash, error) {
	return lmempool.AcceptNewTxToMempool(txn, chain.GetInstance().Height(), nodeID)
}
