package p2p

import (
	"errors"
	"fmt"
	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/conn"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/msg"
	"github.com/btcboost/copernicus/network"
	"github.com/btcboost/copernicus/protocol"
	"github.com/btcboost/copernicus/storage"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultServices         = protocol.SFNodeNetworkAsFullNode | protocol.SFNodeBloomFilter
	DefaultRequiredServices = protocol.SFNodeNetworkAsFullNode
)

type PeerManager struct {
	bytesReceived uint64
	bytesSend     uint64
	started       int32
	shutdown      int32
	shutdownSched int32

	chainParams          *msg.BitcoinParams
	netAddressManager    *network.NetAddressManager
	connectManager       *conn.ConnectManager
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

type getOutboundGroup struct {
	key   string
	reply chan int
}

func NewPeerManager(listenAddrs []string, storage storage.Storage, bitcoinParam *msg.BitcoinParams) (*PeerManager, error) {
	services := DefaultServices
	if conf.AppConf.NoPeerBloomFilters {
		services &^= protocol.SFNodeBloomFilter
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

	connectListener := conn.ConnectListener{
		Listeners:     listeners,
		Dial:          conf.AppDial,
		GetNewAddress: peerManager.newAddressFunc,
	}

	connectManager, err := conn.NewConnectManager(&connectListener)
	if err != nil {
		return nil, err
	}
	peerManager.connectManager = connectManager
	return &peerManager, nil

}
func (peerManager *PeerManager) OutboundGroupCount(key string) int {
	replyChan := make(chan int)
	peerManager.query <- getOutboundGroup{key: key, reply: replyChan}
	return <-replyChan
}

func (peerManager *PeerManager) newAddressFunc() (net.Addr, error) {
	for tries := 0; tries < 100; tries++ {
		address := peerManager.netAddressManager.GetAddress()

		log.Debug(" newAddressFunc ")
		if address == nil {
			log.Debug(" newAddressFunc address is nil")
			break
		}
		key := peerManager.netAddressManager.GetAddress().NetAddress.NetAddressKey()
		if peerManager.OutboundGroupCount(key) != 0 {
			log.Debug("peerManager OutboundGroupCount :%s", key)
			continue
		}

		//if tries < 30 && time.Since(address.LastAttempt) < 10*time.Minute {
		//	continue
		//}
		//port := fmt.Sprintf("%d", address.NetAddress.Port)
		//if tries < 50 && port != msg.ActiveNetParams.DefaultPort {
		//	continue
		//}
		addressString := peerManager.netAddressManager.GetAddress().NetAddress.NetAddressKey()
		log.Debug("get address :%s", addressString)
		return addrStringToNetAddr(addressString)

	}
	return nil, errors.New("no valid conn address")
}

func addrStringToNetAddr(addr string) (net.Addr, error) {
	host, strPort, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, err
	}

	// Skip if host is already an IP address.
	if ip := net.ParseIP(host); ip != nil {
		return &net.TCPAddr{
			IP:   ip,
			Port: port,
		}, nil
	}

	// Tor addresses cannot be resolved to an IP, so just return an onion
	// address instead.
	//if strings.HasSuffix(host, ".onion") {
	//	if cfg.NoOnion {
	//		return nil, errors.New("tor has been disabled")
	//	}
	//
	//	return &onionAddr{addr: addr}, nil
	//}
	//
	// Attempt to look up an IP address associated with the parsed host.
	ips, err := conf.AppLookup(host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for %s", host)
	}

	return &net.TCPAddr{
		IP:   ips[0],
		Port: port,
	}, nil
}

func (peerManager *PeerManager) BanPeer(serverPeer *ServerPeer) {
	peerManager.banPeers <- serverPeer
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

func (peerManager *PeerManager) Start() {
	if atomic.AddInt32(&peerManager.started, 1) != 1 {
		return
	}
	log.Trace("Satarting server")
	peerManager.waitGroup.Add(1)
	go peerManager.peerHandler()
	if peerManager.nat != nil {
		peerManager.waitGroup.Add(1)
		go peerManager.upnpUpdateThread()
	}

}

func (peerManager *PeerManager) peerHandler() {
	peerManager.netAddressManager.Start()
	peerManager.BlockManager.Start()
	log.Trace("Starting p2p handler")
	peerState := &PeerState{
		inboundPeers:    make(map[int32]*ServerPeer),
		persistentPeers: make(map[int32]*ServerPeer),
		outboundPeers:   make(map[int32]*ServerPeer),
		banned:          make(map[string]time.Time),
		outboundGroups:  make(map[string]int),
	}
	if !conf.AppConf.DisableDNSSeed {
		conn.SeedFromDNS(msg.ActiveNetParams, DefaultRequiredServices, conf.AppLookup, func(addresses []*network.PeerAddress) {
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
				log.Trace("Shutdown p2p %s", serverPeer)
				serverPeer.Disconnect()
			})
			break out
		}

	}
	peerManager.connectManager.Stop()
	peerManager.BlockManager.Stop()
	peerManager.netAddressManager.Stop()

}

func (peerManager *PeerManager) handleAddPeerMsg(peerState *PeerState, serverPeer *ServerPeer) bool {
	if serverPeer == nil {
		return false
	}

	// Ignore new peers if we're shutting down.
	if atomic.LoadInt32(&peerManager.shutdown) != 0 {
		log.Info("New p2p %s ignored - server is shutting down", serverPeer)
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
			log.Debug("Peer %s is banned for another %s - disconnecting",
				host, banEnd.String())
			serverPeer.Disconnect()
			return false
		}

		log.Info("Peer %s is no longer banned", host)
		delete(peerState.banned, host)
	}

	// TODO: Check for max peers from a single IP.

	// Limit max number of total peers.
	if peerState.Count() >= conf.AppConf.MaxPeers {
		log.Info("Max peers reached [%d] - disconnecting p2p %s",
			conf.AppConf.MaxPeers, serverPeer)
		serverPeer.Disconnect()
		// TODO: how to handle permanent peers here?
		// they should be rescheduled.
		return false
	}

	// Add the new p2p and start it.
	log.Debug("New p2p %s", serverPeer)
	if serverPeer.Inbound {
		peerState.inboundPeers[serverPeer.ID] = serverPeer
	} else {
		peerState.outboundGroups[serverPeer.PeerAddress.GroupKey()]++
		if serverPeer.persistent {
			peerState.persistentPeers[serverPeer.ID] = serverPeer
		} else {
			peerState.outboundPeers[serverPeer.ID] = serverPeer
		}
	}

	return true
}
func (peerManager *PeerManager) upnpUpdateThread() {

}

//func (s *PeerManager) inboundPeerConnected(conn net.Conn) {
//	sp := NewServerPeer(s, false)
//	sp.Peer = p2p.NewInboundPeer(InitPe(sp))
//	sp.AssociateConnection(conn)
//	go s.peerDoneHandler(sp)
//}
