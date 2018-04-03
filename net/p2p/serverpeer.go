package p2p

import (
	"sync"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/net/conn"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/container"
	"github.com/btcboost/copernicus/net/network"
	"github.com/btcboost/copernicus/net/protocol"

	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/utils"
)

type ServerPeer struct {
	*Peer
	feeFilter       int64
	connectRequest  *conn.ConnectRequest
	peerManager     *PeerManager
	persistent      bool
	continueHash    *utils.Hash
	relayLock       sync.Mutex
	disableRelayTx  bool
	setAddress      bool
	requestQueue    []*msg.InventoryVector
	requestedTxns   map[utils.Hash]struct{}
	requestedBlocks map[utils.Hash]struct{}
	//filter          *bloom.Filter
	knownAddress   map[string]struct{}
	banScore       container.DynamicBanScore
	quit           chan struct{}
	txProcessed    chan struct{}
	blockProcessed chan struct{}
}

func NewServerPeer(peerManager *PeerManager, isPersistent bool) *ServerPeer {
	serverPeer := ServerPeer{
		peerManager:     peerManager,
		persistent:      isPersistent,
		requestedTxns:   make(map[utils.Hash]struct{}),
		requestedBlocks: make(map[utils.Hash]struct{}),
		//	filter:          bloom.LoadFilter(nil),
		knownAddress:   make(map[string]struct{}),
		quit:           make(chan struct{}),
		txProcessed:    make(chan struct{}, 1),
		blockProcessed: make(chan struct{}, 1),
	}
	return &serverPeer
}

func (serverPeer *ServerPeer) newestBlock() (utils.Hash, int32, error) {
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
		logs.Error("can't send address message to %s :%v", serverPeer.Peer, err)
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
			logs.Warn("misbehaving p2p %s :%s --ban score is %d it was not increased this time",
				serverPeer, reason, score)
		}
		return
	}
	score := serverPeer.banScore.Increase(persistent, transient)
	if score > warnThreshold {
		logs.Warn("misbehaving p2p %s :%s -- ban scote increased to %d",
			serverPeer, reason, score)
		if score > conf.AppConf.BanThreshold {
			logs.Warn("misbehaving p2p %s --banning and isconnecting ", serverPeer)
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
			hatTimestamp := serverPeer.ProtocolVersion >= protocol.PeerAddressTimeVersion
			if netAddressManager.NeedMoreAddresses() && hatTimestamp {
				serverPeer.SendMessage(msg.NewGetAddressMessage(), nil)
			}
			netAddressManager.MarkGood(serverPeer.GetNetAddress())
		}
	}
	serverPeer.peerManager.AddPeer(serverPeer)

}

func (serverPeer *ServerPeer) OnRead(p *Peer, bytesRead int, message msg.Message, err error) {

}
func (serverPeer *ServerPeer) OnWrite(p *Peer, bytesWritten int, message msg.Message, err error) {

}

func (serverPeer *ServerPeer) OnGetAddr(p *Peer, msg *msg.GetAddressMessage) {

}
func (serverPeer *ServerPeer) OnAddr(p *Peer, msg *msg.AddressMessage) {

}

func (serverPeer *ServerPeer) OnPing(p *Peer, msg *msg.PingMessage) {

}
func (serverPeer *ServerPeer) OnPong(p *Peer, msg *msg.PongMessage) {

}

func (serverPeer *ServerPeer) OnAlert(p *Peer, msg *msg.AlertMessage) {

}

func (serverPeer *ServerPeer) OnMemPool(p *Peer, msg *msg.MempoolMessage) {
	if serverPeer.peerManager.servicesFlag&protocol.SFNodeBloomFilter != protocol.SFNodeBloomFilter {
		logs.Debug("p2p %v sent mempool request with bloom filtering disable --disconnecting", serverPeer)
		serverPeer.Disconnect()
		return
	}
	serverPeer.addBanScore(0, 33, "mempool")
	//txMempool :=serverPeer.peerManager.txMemPool
	//txDescs :=txMempool.TxDescs()
	//inventoryMessage :=msg.NewInventoryMessageSizeHint(uint(len(txDescs)))
	//for _,teDesc := range txDescs{
	//	if !serverPeer.filter.IsLoaded()|| serverPeer.filter.MatchTxAndUpdate(teDesc.Tx){
	//
	//			inventory:=msg.NewInventoryVecror(msg.INVENTORY_TYPE_TX,txDesc.Tx.Hash())
	//
	//
	//	}
	//}

}
func (serverPeer *ServerPeer) OnTx(p *Peer, msg *msg.TxMessage) {

}
func (serverPeer *ServerPeer) OnBlock(p *Peer, msg *msg.BlockMessage, buf []byte) {

}

func (serverPeer *ServerPeer) OnInv(p *Peer, msg *msg.InventoryMessage) {

}

func (serverPeer *ServerPeer) OnHeaders(p *Peer, msg *msg.HeadersMessage) {

}
func (serverPeer *ServerPeer) OnNotFound(p *Peer, msg *msg.NotFoundMessage) {

}

func (serverPeer *ServerPeer) OnGetData(p *Peer, msg *msg.GetDataMessage) {

}

func (serverPeer *ServerPeer) OnGetBlocks(p *Peer, msg *msg.GetBlocksMessage) {

}
func (serverPeer *ServerPeer) OnGetHeaders(p *Peer, msg *msg.GetHeadersMessage) {

}

func (serverPeer *ServerPeer) OnFilterAdd(p *Peer, msg *msg.FilterAddMessage) {

}
func (serverPeer *ServerPeer) OnFilterClear(p *Peer, msg *msg.FilterClearMessage) {

}
func (serverPeer *ServerPeer) OnFilterLoad(p *Peer, msg *msg.FilterLoadMessage) {

}
func (serverPeer *ServerPeer) OnMerkleBlock(p *Peer, msg *msg.MerkleBlockMessage) {}

func (serverPeer *ServerPeer) OnVerAck(p *Peer, msg *msg.VersionACKMessage) {

}
func (serverPeer *ServerPeer) OnReject(p *Peer, msg msg.RejectMessage) {

}
func (serverPeer *ServerPeer) OnSendHeaders(p *Peer, msg *msg.SendHeadersMessage) {

}
