package connect

import (
	"net"
	"time"
)

type ConnectListener struct {
	listeners       []net.Listener
	OnAccept        func(conn net.Conn)
	TargetOutbound  uint32
	RetryDuration   time.Duration
	OnConnection    func(request *ConnectRequest, conn net.Conn)
	OnDisconnection func(request *ConnectRequest)
	GetNewAddress   func() (net.Addr, error)
	Dial            func(addr net.Addr) (net.Conn, error)
}
