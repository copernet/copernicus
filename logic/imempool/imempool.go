package imempool

import (
	"github.com/btcboost/copernicus/model/mempool"
	core2 "github.com/btcboost/copernicus/model/tx"
	mempool2 "github.com/btcboost/copernicus/logic/mempool"
)

type imempool interface {
	HasSpentOut(*OutPoint) bool
	AcceptTx(*Tx) (bool, error)
	LimitMempoolSize()[]*OutPoint
	RemoveUnFinalTx(*Chain, *CoinsViewCache, int, int)
	RemoveTxSelf([]*Tx)
	RemoveTxRecursive(*Tx, mempool2.PoolRemovalReason)
	Check(*CoinsViewCache, int)
	GetCoin(*OutPoint)Coin
	GetRootTx()map[util.Hash]mempool.TxEntry
	GetAllTxEntry()map[util.Hash]*mempool.TxEntry
	FindTx(util.Hash)*core2.Tx
	Size()int
} 