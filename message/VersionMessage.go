package message

import (
	"copernicus/protocol"
	"time"
	"copernicus/network"
)

type VersionMessage struct {
	Message
	ProtocolVersion int32
	ServiceFlag     protocol.ServiceFlag
	Timestamp       time.Time
	RemoteAddress   network.NetAddress
	LocalAddress    network.NetAddress
	Nonce           uint64
	UserAgent       string
	LastBlock       int32
	DisableRelayTx  bool
}
