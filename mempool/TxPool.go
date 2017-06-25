package mempool

import (
	"sync"
	"copernicus/model"
	"copernicus/utils"
)

type TxPool struct {
	lastUpdate    uint64
	lock          sync.RWMutex
	mempoolConfig MempoolConfig
	pool          map[utils.Hash]*TxDesc
	orghans       map[utils.Hash]*model.Transaction
	orphansByPrev map[utils.Hash]map[utils.Hash]*model.Transaction
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
