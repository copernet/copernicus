package imempool

import (
	"github.com/btcboost/copernicus/model/mempool"
	core2 "github.com/btcboost/copernicus/model/tx"
	mempool2 "github.com/btcboost/copernicus/logic/mempool"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/util"
)

type imempool interface {
	HasSpentOut(point *outpoint.OutPoint) bool
	LimitMempoolSize()[]*outpoint.OutPoint
	RemoveUnFinalTx(*chain.Chain, *CoinsViewCache, int, int)
	RemoveTxSelf([]*tx.Tx)
	RemoveTxRecursive(*tx.Tx, mempool2.PoolRemovalReason)
	Check(*CoinsViewCache, int)
	GetCoin(point *outpoint.OutPoint)Coin
	GetRootTx()map[util.Hash]mempool.TxEntry
	GetAllTxEntry()map[util.Hash]*mempool.TxEntry
	FindTx(util.Hash)*core2.Tx
	Size()int
} 