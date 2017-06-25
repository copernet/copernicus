package mempool

import (
	"sync"
	"copernicus/crypto"
	"copernicus/model"
)

type TxPool struct {
	lastUpdate    uint64
	lock          sync.RWMutex
	mempoolConfig MempoolConfig
	pool          map[crypto.Hash]*TxDesc
	orghans       map[crypto.Hash]*model.Transaction
	orphansByPrev map[crypto.Hash]map[crypto.Hash]*model.Transaction
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
