package p2p

import (
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/net/network"
	"github.com/btcboost/copernicus/net/protocol"
	"github.com/btcboost/copernicus/utils"
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
