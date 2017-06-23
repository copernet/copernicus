package mempool

import "sync"

type TxPool struct {
lastUpdate uint64
	lock sync.Mutex
	
}
