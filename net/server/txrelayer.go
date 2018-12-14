package server

import (
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/util"
	"sync"
	"time"
)

type txCacheItem struct {
	txn       *tx.Tx
	cacheTime time.Time
}

type TxRelayer struct {
	lck sync.RWMutex

	cache map[util.Hash]*txCacheItem

	lastCompactTime time.Time
}

func (txr *TxRelayer) Cache(txID *util.Hash, txn *tx.Tx) {
	txr.lck.Lock()
	defer txr.lck.Unlock()

	if txn == nil {
		txe := lmempool.FindTxInMempool(mempool.GetInstance(), *txID)
		if txe == nil {
			return
		}
		txn = txe.Tx
	}

	txr.cache[*txID] = &txCacheItem{txn: txn, cacheTime: time.Now()}

	txr.limitCache()
}

func (txr *TxRelayer) TxToRelay(txID *util.Hash) *tx.Tx {
	txr.lck.RLock()
	defer txr.lck.RUnlock()

	if v, exists := txr.cache[*txID]; exists {
		return v.txn
	}

	if txe := mempool.GetInstance().FindTx(*txID); txe != nil {
		return txe.Tx
	}

	return nil
}

func (txr *TxRelayer) limitCache() {
	if time.Now().Before(txr.lastCompactTime.Add(time.Minute * 15)) {
		return
	}

	_15minsAgo := time.Now().Add(time.Minute * -15)
	for k, v := range txr.cache {
		if v.cacheTime.Before(_15minsAgo) {
			delete(txr.cache, k)
		}
	}

	txr.lastCompactTime = time.Now()
}

func NewTxRelayer() *TxRelayer {
	return &TxRelayer{
		cache:           make(map[util.Hash]*txCacheItem),
		lastCompactTime: time.Now(),
	}
}
