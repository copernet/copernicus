package peer

import (
	"github.com/btccom/copernicus/protocol"
	"github.com/btccom/copernicus/network"
	"github.com/btccom/copernicus/utils"
	"github.com/btccom/copernicus/msg"
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
