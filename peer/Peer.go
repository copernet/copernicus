package peer

import (
	"net"
	"fmt"
	"github.com/astaxie/beego/logs"
	"sync"
	"copernicus/crypto"
	"copernicus/network"
	"copernicus/protocol"
	"time"
	"sync/atomic"
	"copernicus/message"
	"copernicus/store"
)

type Peer struct {
	Config               *PeerConfig
	Id                   int32
	lastReceive          uint64
	lastSent             uint64
	lastReceiveTime      time.Time
	lastSendTime         time.Time
	connected            int32
	disconnect           int32
	Inbound              bool
	BlockStatusMutex     sync.RWMutex
	conn                 net.Conn
	address              string
	lastDeclareBlock     *crypto.Hash
	PeerStatusMutex      sync.RWMutex
	Address              *network.NetAddress
	ServiceFlag          protocol.ServiceFlag
	UserAgent            string
	PingNonce            uint64
	PingTime             time.Time
	PingMicros           int64
	VersionKnown         bool
	SentVerAck           bool
	GotVerAck            bool
	ProtocolVersion      uint32
	LastBlock            int32
	ConnectedTime        time.Time
	TimeOffset           int64
	StartingHeight       int32
	SendHeadersPreferred bool
}

var log = logs.NewLogger()

func (p *Peer) ToString() string {
	directionString := "Inbound"
	if !p.Inbound {
		directionString = "outbound"
	}
	return fmt.Sprintf("%s (%s)", p.address, directionString)
}
func (p *Peer) UpdateBlockHeight(newHeight int32) {
	p.BlockStatusMutex.Lock()
	log.Trace("Updating last block heighy of peer %v from %v to %v", p.address, p.LastBlock, newHeight)
	p.LastBlock = newHeight
	p.BlockStatusMutex.Unlock()
}

func (p *Peer) UpdateDeclareBlock(blackHash *crypto.Hash) {
	log.Trace("Updating last block:%v form peer %v", blackHash, p.address)
	p.BlockStatusMutex.Lock()
	p.lastDeclareBlock = blackHash
	p.BlockStatusMutex.Unlock()

}
func (p *Peer) GetPeerID() int32 {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.Id

}
func (p *Peer) GetNetAddress() *network.NetAddress {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.Address
}
func (p*Peer) GetServiceFlag() protocol.ServiceFlag {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.ServiceFlag

}
func (p*Peer) GetUserAgent() string {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.UserAgent
}
func (p *Peer) GetLastDeclareBlock() *crypto.Hash {

	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.lastDeclareBlock
}
func (p*Peer) LastSent() uint64 {
	return atomic.LoadUint64(&p.lastSent)
}
func (p*Peer) LastReceived() uint64 {
	return atomic.LoadUint64(&p.lastReceive)
}

func (p *Peer) LocalVersionMsg() (*message.VersionMessage, error) {
	var blockNumber int32
	if p.Config.NewBlock != nil {
		_, blockNumber, err := p.Config.NewBlock()
		if err != nil {
			return nil, err
		}

	}
	peerAddress := p.Address
	if p.Config.Proxy != "" {
		proxyAddress, _, err := net.SplitHostPort(p.Config.Proxy)
		if err != nil || p.Address.IP.String() == proxyAddress {
			peerAddress = &network.NetAddress{
				Timestamp: time.Now(),
				IP:        net.IP([]byte{0, 0, 0, 0}),
			}
		}
	}
	localAddress := p.Address
	if p.Config.BestAddress != nil {
		localAddress = p.Config.BestAddress(p.Address)
	}
	nonce, err := store.RandomUint64()
	if err != nil {
		return nil, err
	}

}
