package peer

import "copernicus/protocol"

type PeerManager struct {
	bytesReceived uint64
	bytesSend     uint64
	started       int32
	shutdown      int32
	shutdownSched int32

	chainParams *protocol.BitcoinParams

}
