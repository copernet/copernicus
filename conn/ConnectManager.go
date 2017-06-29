package conn

import (
	"github.com/astaxie/beego/logs"
	"github.com/pkg/errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxRetryDuration      = time.Minute * 5
	defaultRetryDuration  = time.Second * 5
	defaultTargetOutbound = uint32(8)
	maxFailedAttempts     = 25
)

var log = logs.NewLogger()

type ConnectManager struct {
	connRequestCount uint64
	start            int32
	stop             int32
	listener         *ConnectListener
	waitGroup        sync.WaitGroup
	failedAttempts   uint64
	requests         chan interface{}
	quit             chan struct{}
}

type handleConnected struct {
	connectRequest *ConnectRequest
	conn           net.Conn
}

type handleDisconnected struct {
	id    uint64
	retry bool
}

type handleFailed struct {
	connectRequest *ConnectRequest
	err            error
}

func (connectManager *ConnectManager) handleFailedConnect(connectRequest *ConnectRequest) {
	if atomic.LoadInt32(&connectManager.stop) != 0 {
		return
	}
	if connectRequest.Permanent {
		connectRequest.retryCount++
		duration := time.Duration(connectRequest.retryCount) * connectManager.listener.RetryDuration
		if duration > maxRetryDuration {
			duration = maxRetryDuration
		}
		log.Debug("Retrying connection to %v in %v", connectRequest, duration)
		time.AfterFunc(duration, func() {
			connectManager.Connect(connectRequest)
		})

	} else if connectManager.listener.GetNewAddress != nil {
		connectManager.failedAttempts++
		if connectManager.failedAttempts >= maxFailedAttempts {
			log.Debug("max failed connection attemps reached :[%d]--retyeing connection in:%v",
				maxFailedAttempts, connectManager.listener.RetryDuration)
			time.AfterFunc(connectManager.listener.RetryDuration, func() {
				connectManager.NewConnectRequest()
			})
		} else {
			go connectManager.NewConnectRequest()
		}

	}
}

func (connectManager *ConnectManager) Connect(connectRequest *ConnectRequest) {
	if atomic.LoadInt32(&connectManager.stop) != 0 {
		return
	}
	if atomic.LoadUint64(&connectRequest.id) == 0 {
		atomic.StoreUint64(&connectRequest.id, atomic.AddUint64(&connectManager.connRequestCount, 1))

	}
	log.Debug("attempting to conn to %v ", connectRequest)
	conn, err := connectManager.listener.Dial(connectRequest.Address)
	if err != nil {
		connectManager.requests <- handleFailed{connectRequest, err}
	} else {
		connectManager.requests <- handleConnected{connectRequest, conn}
	}

}

func (connectManager *ConnectManager) Disconnect(id uint64) {
	if atomic.LoadInt32(&connectManager.stop) != 0 {
		return
	}
	connectManager.requests <- handleDisconnected{id, true}
}

func (connectManager *ConnectManager) Remove(id uint64) {
	if atomic.LoadInt32(&connectManager.stop) != 0 {
		return
	}
	connectManager.requests <- handleDisconnected{id, false}
}

func (connectManager *ConnectManager) listenHandler(listener net.Listener) {
	log.Info("server listening on %s", listener.Addr())
	for atomic.LoadInt32(&connectManager.stop) == 0 {
		conn, err := listener.Accept()
		if err != nil {
			if atomic.LoadInt32(&connectManager.stop) == 0 {
				log.Error("Can't accept connection :%v", err)
			}
			continue
		}
		go connectManager.listener.OnAccept(conn)
	}
	connectManager.waitGroup.Done()
	log.Trace("Listener handler done for %s", listener.Addr())
}

func (connectManager *ConnectManager) NewConnectRequest() {
	//if atomic.LoadInt32(&connectManager.stop) != 0 {
	//	return
	//}
	log.Debug("NewConnectRequest ")
	if connectManager.listener.GetNewAddress == nil {
		return
	}
	connectRequest := &ConnectRequest{}
	atomic.StoreUint64(&connectRequest.id, atomic.AddUint64(&connectManager.connRequestCount, 1))
	address, err := connectManager.listener.GetNewAddress()
	log.Debug("NewConnectRequest :%s", address)
	if err != nil {
		connectManager.requests <- handleFailed{connectRequest, err}
		return
	}
	connectRequest.Address = address
	connectManager.Connect(connectRequest)

}

func (connectManager *ConnectManager) connectHandler() {
	conns := make(map[uint64]*ConnectRequest, connectManager.listener.TargetOutbound)
out:
	for {
		select {
		case request := <-connectManager.requests:
			switch msg := request.(type) {
			case handleConnected:
				connectRequest := msg.connectRequest
				connectRequest.updateState(ConnectEstablished)
				connectRequest.Conn = msg.conn
				conns[connectRequest.id] = connectRequest
				log.Debug("connected to %v", connectRequest)
				connectRequest.retryCount = 0
				connectManager.failedAttempts = 0
				if connectManager.listener.OnConnection != nil {
					go connectManager.listener.OnConnection(connectRequest, msg.conn)
				}
			case handleDisconnected:
				if connectRequest, ok := conns[msg.id]; ok {
					connectRequest.updateState(ConnectDisconnected)
					if connectRequest.Conn != nil {
						connectRequest.Conn.Close()
					}
					log.Debug("Disconnected from %v", connectRequest)
					delete(conns, msg.id)
					if connectManager.listener.OnDisconnection != nil {
						go connectManager.listener.OnDisconnection(connectRequest)
					}
					if uint32(len(conns)) < connectManager.listener.TargetOutbound && msg.retry {
						connectManager.handleFailedConnect(connectRequest)
					}
				} else {
					log.Error("unknown connectiong :%d", msg.id)
				}

			case handleFailed:
				connectRequest := msg.connectRequest
				connectRequest.updateState(ConnectFailed)
				log.Debug("failed to conn to %v :%v", connectRequest, msg.err)
				connectManager.handleFailedConnect(connectRequest)

			}
		case <-connectManager.quit:
			break out

		}
	}

}

func (connectManager *ConnectManager) Start() {
	if atomic.AddInt32(&connectManager.start, 1) != 1 {
		log.Trace("Connection manager return")
		return
	}
	log.Trace("Connection manager started")
	connectManager.waitGroup.Add(1)
	go connectManager.connectHandler()
	log.Trace("Connection manager connectHandler")
	if connectManager.listener.OnAccept != nil {
		for _, listener := range connectManager.listener.Listeners {
			connectManager.waitGroup.Add(1)
			go connectManager.listenHandler(listener)
		}
	}
	for i := atomic.LoadUint64(&connectManager.connRequestCount); i < uint64(connectManager.listener.TargetOutbound); i++ {
		log.Trace("Connection manager NewConnectRequest")
		go connectManager.NewConnectRequest()
	}

}

func (connectManager *ConnectManager) Wait() {
	connectManager.waitGroup.Wait()
}

func (connectManager *ConnectManager) Stop() {
	if atomic.AddInt32(&connectManager.stop, 1) != 1 {
		log.Warn("connection manager already stopped")
		return
	}
	for _, listener := range connectManager.listener.Listeners {
		_ = listener.Close()
	}
	close(connectManager.quit)
	log.Trace("connection manager stopped")
}

func NewConnectManager(listener *ConnectListener) (*ConnectManager, error) {
	if listener.Dial == nil {
		return nil, errors.New("dial can't be nil")
	}
	if listener.RetryDuration <= 0 {
		listener.RetryDuration = defaultRetryDuration

	}
	if listener.TargetOutbound == 0 {
		listener.TargetOutbound = defaultTargetOutbound

	}
	connectManager := ConnectManager{
		listener: listener,
		requests: make(chan interface{}),
		quit:     make(chan struct{}),
	}
	return &connectManager, nil
}
