package connect

import (
	"net"
	"sync"
	"sync/atomic"
	"fmt"
)

type ConnectRequest struct {
	id         uint64
	Address    net.Addr
	Permanent  bool
	Conn       net.Conn
	state      ConnectState
	lock       sync.RWMutex
	retryCount uint32
}

func (connectRequest *ConnectRequest) updateState(state ConnectState) {
	connectRequest.lock.Lock()
	defer connectRequest.lock.Unlock()
	connectRequest.state = state
}
func (connectRequest *ConnectRequest) ID() uint64 {
	return atomic.LoadUint64(&connectRequest.id)
}

func (connectRequest *ConnectRequest) State() ConnectState {
	connectRequest.lock.RLock()
	defer connectRequest.lock.RUnlock()
	return connectRequest.state
}
func (connectRequest *ConnectRequest) String() string {
	if connectRequest.Address.String() == "" {
		return fmt.Sprintf("reqid %d", atomic.LoadUint64(&connectRequest.id))
	}
	return fmt.Sprintf("%s (reqid %d)", connectRequest.Address, atomic.LoadUint64(&connectRequest.id))
}

