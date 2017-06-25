package peer

import (
	"copernicus/protocol"
	"copernicus/network"
	"copernicus/utils"
	"copernicus/msg"
)

type PeerConfig struct {
	NewBlock          utils.HashFunc
	HostToAddressFunc network.HostToNetAddrFunc
	BestAddress       network.PeerAddressFunc
	
	Proxy            string
	UserAgent        string
	UserAgentVersion string
	
	// BIP 14
	UserAgentComments []string
	
	ServicesFlag    protocol.ServiceFlag
	ProtocolVersion uint32
	DisableRelayTx  bool
	Listener        MessageListener
	ChainParams     *msg.BitcoinParams
}
