package peer

import (
	"copernicus/protocol"
	"copernicus/storage"
	"copernicus/conf"
	"copernicus/network"
	"copernicus/manager"
	"copernicus/connect"
	"sync"
	"copernicus/blockchain"
)

const (
	DEFAULT_SERVICES = protocol.SF_NODE_NETWORK_AS_FULL_NODE | protocol.SF_NODE_BLOOM_FILTER
)

type PeerManager struct {
	bytesReceived uint64
	bytesSend     uint64
	started       int32
	shutdown      int32
	shutdownSched int32
	
	chainParams          *protocol.BitcoinParams
	netAddressManager    *network.NetAddressManager
	connectManager       *connect.ConnectManager
	BlockManager         *manager.BlockManager
	modifyRebroadcastInv chan interface{}
	newPeers             chan *ServerPeer
	banPeers             chan *ServerPeer
	donePeers            chan *ServerPeer
	query                chan interface{}
	relayInventory       chan RelayMessage
	broadcast            chan BroadcastMessage
	peerHeightsUpdate    chan UpdatePeerHeightsMessage
	waitGroup            sync.WaitGroup
	quit                 chan struct{}
	
	nat          network.NATInterface
	storage      storage.Storage
	timeSource   blockchain.IMedianTimeSource
	servicesFlag protocol.ServiceFlag
	
	//txIndex   *indexers.TxIndex
	//addrIndex *indexers.AddrIndex
}

func NewPeerManager(listenAddrs [] string, storage storage.Storage, bitcoinParam *protocol.BitcoinParams) (*PeerManager, error) {
	services := DEFAULT_SERVICES
	if conf.AppConf.NoPeerBloomFilters {
		services &^= protocol.SF_NODE_BLOOM_FILTER
	}
	netAddressManager := network.NewNetAddressManager(conf.AppConf.DataDir, conf.AppLookup)
	//var listeners []net.Listener
	var natListener network.NATInterface
	peerManager := PeerManager{
		chainParams:          bitcoinParam,
		netAddressManager:    netAddressManager,
		newPeers:             make(chan *ServerPeer, conf.AppConf.MaxPeers),
		donePeers:            make(chan *ServerPeer, conf.AppConf.MaxPeers),
		banPeers:             make(chan *ServerPeer, conf.AppConf.MaxPeers),
		query:                make(chan interface{}),
		relayInventory:       make(chan RelayMessage, conf.AppConf.MaxPeers),
		broadcast:            make(chan BroadcastMessage, conf.AppConf.MaxPeers),
		quit:                 make(chan struct{}),
		modifyRebroadcastInv: make(chan interface{}),
		peerHeightsUpdate:    make(chan UpdatePeerHeightsMessage),
		nat:                  natListener,
		storage:              storage,
		timeSource:           blockchain.NewMedianTime(),
		servicesFlag:         protocol.ServiceFlag(services),
		
		
	}
	//connectListener :=connect.ConnectListener{
	//	listeners:listeners,
	//	OnAccept:peerManager.in
	//
	//}
	return &peerManager, nil
	
}

func (peerManage *PeerManager) BanPeer(serverPeer *ServerPeer) {
	peerManage.banPeers <- serverPeer
}

//func (s *PeerManager) inboundPeerConnected(conn net.Conn) {
//	sp := NewServerPeer(s, false)
//	sp.Peer = peer.NewInboundPeer(InitPe(sp))
//	sp.AssociateConnection(conn)
//	go s.peerDoneHandler(sp)
//}
