package peer

import (
	"copernicus/protocol"
	"copernicus/storage"
	"copernicus/conf"
	"copernicus/network"
	"copernicus/connect"
	"sync"
	"copernicus/blockchain"
	"copernicus/mempool"
	"sync/atomic"
	"time"
	"net"
	"copernicus/msg"
)

const (
	DEFAULT_SERVICES          = protocol.SF_NODE_NETWORK_AS_FULL_NODE | protocol.SF_NODE_BLOOM_FILTER
	DEFAULT_REQUIRED_SERVICES = protocol.SF_NODE_NETWORK_AS_FULL_NODE
)

type PeerManager struct {
	bytesReceived uint64
	bytesSend     uint64
	started       int32
	shutdown      int32
	shutdownSched int32
	
	chainParams          *msg.BitcoinParams
	netAddressManager    *network.NetAddressManager
	connectManager       *connect.ConnectManager
	BlockManager         *BlockManager
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
	
	txMemPool    *mempool.TxPool
	nat          network.NATInterface
	storage      storage.Storage
	timeSource   blockchain.IMedianTimeSource
	servicesFlag protocol.ServiceFlag
	
	//txIndex   *indexers.TxIndex
	//addrIndex *indexers.AddrIndex
}

func NewPeerManager(listenAddrs [] string, storage storage.Storage, bitcoinParam *msg.BitcoinParams) (*PeerManager, error) {
	services := DEFAULT_SERVICES
	if conf.AppConf.NoPeerBloomFilters {
		services &^= protocol.SF_NODE_BLOOM_FILTER
	}
	netAddressManager := network.NewNetAddressManager(conf.AppConf.DataDir, conf.AppLookup)
	var listeners []net.Listener
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
	
	connectListener := connect.ConnectListener{
		Listeners:listeners,
		Dial:conf.AppDial,
	
	
	}
	
	connectManager, err := connect.NewConnectManager(&connectListener)
	if err != nil {
		return nil, err
	}
	peerManager.connectManager = connectManager
	return &peerManager, nil
	
}

func (peerManage *PeerManager) BanPeer(serverPeer *ServerPeer) {
	peerManage.banPeers <- serverPeer
}

func (peerManager *PeerManager) AddPeer(serverPeer *ServerPeer) {
	peerManager.newPeers <- serverPeer
}

func (peerManager *PeerManager) Stop() error {
	if atomic.AddInt32(&peerManager.shutdown, 1) != 1 {
		log.Info("PeerManager is aleray in the process of shtting down")
		return nil
	}
	log.Info("PeerManager shtting down")
	close(peerManager.quit)
	return nil
}
func (peerManager *PeerManager) WaitForShutdown() {
	peerManager.waitGroup.Wait()
}

func (peerManger *PeerManager) Start() {
	if atomic.AddInt32(&peerManger.started, 1) != 1 {
		return
	}
	log.Trace("Satarting server")
	peerManger.waitGroup.Add(1)
	go peerManger.peerHandler()
	if peerManger.nat != nil {
		peerManger.waitGroup.Add(1)
		go peerManger.upnpUpdateThread()
	}
	
}

func (peerManager *PeerManager) peerHandler() {
	peerManager.netAddressManager.Start()
	peerManager.BlockManager.Start()
	log.Trace("Starting peer handler")
	peerState := &PeerState{
		inboundPeers:    make(map[int32]*ServerPeer),
		persistentPeers: make(map[int32]*ServerPeer),
		outboundPeers:   make(map[int32]*ServerPeer),
		banned:          make(map[string]time.Time),
		outboundGroups:  make(map[string]int),
	}
	if !conf.AppConf.DisableDNSSeed {
		connect.SeedFromDNS(msg.ActiveNetParams, DEFAULT_REQUIRED_SERVICES, conf.AppLookup, func(addresses []*network.PeerAddress) {
		log.Warn(addresses[0].IP.String())
				peerManager.netAddressManager.AddPeerAddresses(addresses, addresses[0])
		})
		
	}
	go peerManager.connectManager.Start()

out:
	for {
		select {
		case peer := <-peerManager.newPeers:
			peerManager.handleAddPeerMsg(peerState, peer)
		case <-peerManager.quit:
			peerState.forAllPeers(func(serverPeer *ServerPeer) {
				log.Trace("Shutdown peer %s", serverPeer)
				serverPeer.Disconnect()
			})
			break out
		}
		
	}
	peerManager.connectManager.Stop()
	peerManager.BlockManager.Stop()
	peerManager.netAddressManager.Stop()
	
}

func (s *PeerManager) handleAddPeerMsg(peerState *PeerState, serverPeer *ServerPeer) bool {
	if serverPeer == nil {
		return false
	}
	
	// Ignore new peers if we're shutting down.
	if atomic.LoadInt32(&s.shutdown) != 0 {
		log.Info("New peer %s ignored - server is shutting down", serverPeer)
		serverPeer.Disconnect()
		return false
	}
	
	// Disconnect banned peers.
	host, _, err := net.SplitHostPort(serverPeer.AddressString)
	if err != nil {
		log.Debug("can't split hostport %v", err)
		serverPeer.Disconnect()
		return false
	}
	if banEnd, ok := peerState.banned[host]; ok {
		if time.Now().Before(banEnd) {
			log.Debug("Peer %s is banned for another %v - disconnecting",
				host, banEnd.Sub(time.Now()))
			serverPeer.Disconnect()
			return false
		}
		
		log.Info("Peer %s is no longer banned", host)
		delete(peerState.banned, host)
	}
	
	// TODO: Check for max peers from a single IP.
	
	// Limit max number of total peers.
	if peerState.Count() >= conf.AppConf.MaxPeers {
		log.Info("Max peers reached [%d] - disconnecting peer %s",
			conf.AppConf.MaxPeers, serverPeer)
		serverPeer.Disconnect()
		// TODO: how to handle permanent peers here?
		// they should be rescheduled.
		return false
	}
	
	// Add the new peer and start it.
	log.Debug("New peer %s", serverPeer)
	if serverPeer.Inbound {
		peerState.inboundPeers[serverPeer.Id] = serverPeer
	} else {
		peerState.outboundGroups[serverPeer.PeerAddress.GroupKey()]++
		if serverPeer.persistent {
			peerState.persistentPeers[serverPeer.Id] = serverPeer
		} else {
			peerState.outboundPeers[serverPeer.Id] = serverPeer
		}
	}
	
	return true
}
func (peerManager *PeerManager) upnpUpdateThread() {

}

//func (s *PeerManager) inboundPeerConnected(conn net.Conn) {
//	sp := NewServerPeer(s, false)
//	sp.Peer = peer.NewInboundPeer(InitPe(sp))
//	sp.AssociateConnection(conn)
//	go s.peerDoneHandler(sp)
//}
