package peer

import (
	"copernicus/connect"
	"copernicus/crypto"
	"sync"
	"copernicus/msg"
	"github.com/btcsuite/btcutil/bloom"
	"copernicus/algorithm"
	"copernicus/network"
	"copernicus/conf"
	"copernicus/protocol"
)

type ServerPeer struct {
	feeFilter       int64
	*Peer
	connectRequest  *connect.ConnectRequest
	peerManager     *PeerManager
	persistent      bool
	continueHash    *crypto.Hash
	relayLock       sync.Mutex
	disableRelayTx  bool
	setAddress      bool
	requestQueue    [] *msg.InventoryVector
	requestedTxns   map[crypto.Hash]struct{}
	requestedBlocks map[crypto.Hash]struct{}
	filter          *bloom.Filter
	knownAddress    map[string]struct{}
	banScore        algorithm.DynamicBanScore
	quit            chan struct{}
	txProcessed     chan struct{}
	blockProcessed  chan struct{}
}

func NewServerPeer(peerManager *PeerManager, isPersistent bool) (*ServerPeer) {
	serverPeer := ServerPeer{
		peerManager:     peerManager,
		persistent:      isPersistent,
		requestedTxns:   make(map[crypto.Hash]struct{}),
		requestedBlocks: make(map[crypto.Hash]struct{}),
		filter:          bloom.LoadFilter(nil),
		knownAddress:    make(map[string]struct{}),
		quit:            make(chan struct{}),
		txProcessed:     make(chan struct{}, 1),
		blockProcessed:  make(chan struct{}, 1),
		
	}
	return &serverPeer
}

func (serverPeer *ServerPeer) newestBlock() (crypto.Hash, int32, error) {
	return serverPeer.peerManager.BlockManager.Chain.BestBlockHash()
}

func (serverPeer *ServerPeer) addKnownAddress(addresses []*network.PeerAddress) {
	for _, peerAddress := range addresses {
		serverPeer.knownAddress[peerAddress.NetAddressKey()] = struct{}{}
	}
}

func (serverPeer *ServerPeer) addressKnown(peerAddress *network.PeerAddress) bool {
	_, exists := serverPeer.knownAddress[peerAddress.NetAddressKey()]
	return exists
}

func (serverPeer *ServerPeer) RelayTxDisabled() bool {
	serverPeer.relayLock.Lock()
	defer serverPeer.relayLock.Unlock()
	return serverPeer.disableRelayTx
}
func (serverPeer *ServerPeer) SetDisableRelayTx(disable bool) {
	serverPeer.relayLock.Lock()
	defer serverPeer.relayLock.Unlock()
	serverPeer.disableRelayTx = disable
}

func (serverPeer *ServerPeer) pushAddressMessage(peerAddresses []*network.PeerAddress) {
	addresses := make([]*network.PeerAddress, 0, len(peerAddresses))
	for _, address := range addresses {
		if !serverPeer.addressKnown(address) {
			addresses = append(addresses, address)
		}
		
	}
	knownes, err := serverPeer.SendAddrMessage(addresses)
	if err != nil {
		log.Error("can't send address message to %s :%v", serverPeer.Peer, err)
		serverPeer.Disconnect()
		return
	}
	serverPeer.addKnownAddress(knownes)
	
}

func (serverPeer *ServerPeer) addBanScore(persistent, transient uint32, reason string) {
	if conf.AppConf.DisableBanning {
		return
	}
	warnThreshold := conf.AppConf.BanThreshold >> 1
	if transient == 0 && persistent == 0 {
		score := serverPeer.banScore.Int()
		if score > warnThreshold {
			log.Warn("misbehaving peer %s :%s --ban score is %d it was not increased this time",
				serverPeer, reason, score)
		}
		return
	}
	score := serverPeer.banScore.Increase(persistent, transient)
	if score > warnThreshold {
		log.Warn("misbehaving peer %s :%s -- ban scote increased to %d",
			serverPeer, reason, score)
		if score > conf.AppConf.BanThreshold {
			log.Warn("misbehaving peer %s --banning and isconnecting ", serverPeer)
			serverPeer.peerManager.BanPeer(serverPeer)
			serverPeer.Disconnect()
		}
	}
	
}
func (serverPeer *ServerPeer) OnVersion(p *Peer, versionMessage *msg.VersionMessage) {
	serverPeer.peerManager.timeSource.AddTimeSample(serverPeer.AddressString, versionMessage.Timestamp)
	serverPeer.peerManager.BlockManager.NewPeer(serverPeer)
	serverPeer.SetDisableRelayTx(versionMessage.DisableRelayTx)
	if conf.AppConf.SimNet {
		netAddressManager := serverPeer.peerManager.netAddressManager
		if !serverPeer.Inbound {
			if !conf.AppConf.DisableListen {
				localAddress := netAddressManager.GetBestLocalAddress(serverPeer.GetNetAddress())
				if localAddress.IsRoutable() {
					addresses := []*network.PeerAddress{localAddress}
					serverPeer.pushAddressMessage(addresses)
					
				}
			}
			hatTimestamp := serverPeer.ProtocolVersion >= protocol.PEER_ADDRESS_TIME_VERSION
			if netAddressManager.NeedMoreAddresses() && hatTimestamp {
				serverPeer.SendMessage(msg.NewGetAddressMessage(), nil)
			}
			netAddressManager.MarkGood(serverPeer.GetNetAddress())
		}
	}
	serverPeer.peerManager.AddPeer(serverPeer)
	
}

func (serverPeer *ServerPeer) OnMemPool(p *Peer, msg *msg.MempoolMessage) {
	if serverPeer.peerManager.servicesFlag&protocol.SF_NODE_BLOOM_FILTER != protocol.SF_NODE_BLOOM_FILTER {
		log.Debug("peer %v sent mempool request with bloom filtering disable --disconnecting", serverPeer)
		serverPeer.Disconnect()
		return
	}
	serverPeer.addBanScore(0,33,"mempool")
	
	
}
