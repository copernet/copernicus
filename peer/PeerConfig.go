package peer

import "copernicus/protocol"

type PeerConfig struct {
	NewBlock    protocol.HashFunc
	HostToAdd   protocol.HostToNetAddrFunc
	BestAddress protocol.NetAddressFunc

	Proxy            string
	UserAgent        string
	UserAgentVersion string

	// BIP 14
	UserAgentComments []string

	ServicesFlag    protocol.ServiceFlag
	ProtocolVersion uint32
	DisableRelayTx  bool
	Listener        MessageListener
}
