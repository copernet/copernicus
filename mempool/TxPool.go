package mempool

import (
	"github.com/btcboost/copernicus/model"
	"github.com/btcboost/copernicus/utils"
	"sync"
)

type TxPool struct {
	lastUpdate    uint64
	lock          sync.RWMutex
	mempoolConfig TxPoolConfig
	pool          map[utils.Hash]*TxDesc
	orghans       map[utils.Hash]*model.Tx
	orphansByPrev map[utils.Hash]map[utils.Hash]*model.Tx
	pennyTotal    float64
	lastPennyUnix int64
}

func (txPool *TxPool) TxDescs() []*TxDesc {
	txPool.lock.RLock()
	defer txPool.lock.RUnlock()
	descs := make([]*TxDesc, len((txPool.pool)))
	i := 0
	for _, desc := range txPool.pool {
		descs[i] = desc
		i++
	}
	return descs
}
