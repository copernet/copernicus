package peer

import (
	"net"
	"btcboost/protocol"
)


type PeerConfig struct {
	NewestBlock protocol.HashFunc

}
type Peer struct {
	bytesReceived uint64
	bytesSent     uint64
	lastRecv      int64
	lastSend      int64
	connected     int32
	disconnect    int32

	conn          net.Conn

	address       string
}

