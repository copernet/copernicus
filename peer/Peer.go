package peer

import (
	"net"
	"fmt"
	"github.com/astaxie/beego/logs"
	"sync"
	"copernicus/crypto"
	"copernicus/protocol"
	"time"
	"sync/atomic"
	"copernicus/msg"
	"copernicus/utils"
	"math/rand"
	"copernicus/algorithm"
	"github.com/pkg/errors"
	log2 "copernicus/log"
	"bytes"
	"github.com/davecgh/go-spew/spew"
)

type Peer struct {
	Config           *PeerConfig
	Id               int32
	lastReceive      uint64
	lastSent         uint64
	lastReceiveTime  time.Time
	lastSendTime     time.Time
	connected        int32
	disconnect       int32
	Inbound          bool
	BlockStatusMutex sync.RWMutex
	conn             net.Conn
	address          string
	lastDeclareBlock *crypto.Hash
	PeerStatusMutex  sync.RWMutex
	Address          *msg.PeerAddress
	ServiceFlag      protocol.ServiceFlag

	UserAgent    string
	PingNonce    uint64
	PingTime     time.Time
	PingMicros   int64
	VersionKnown bool
	SentVerAck   bool
	GotVerAck    bool

	ProtocolVersion      uint32
	LastBlock            int32
	ConnectedTime        time.Time
	TimeOffset           int64
	StartingHeight       int32
	SendHeadersPreferred bool
	OutputQueue          chan msg.OutMessage
	SendQueue            chan msg.OutMessage

	GetBlocksLock  sync.Mutex
	GetBlocksBegin *crypto.Hash
	GetBlocksStop  *crypto.Hash

	GetHeadersLock  sync.Mutex
	GetHeadersBegin *crypto.Hash
	GetHeadersStop  *crypto.Hash

	quit chan struct{}
}

var (
	log       = logs.NewLogger()
	sentNoces = algorithm.NewLRUCache(50)
	nodeCount int32
)

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
func (p *Peer) GetNetAddress() *msg.PeerAddress {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.Address
}
func (p *Peer) GetServiceFlag() protocol.ServiceFlag {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.ServiceFlag

}
func (p *Peer) GetUserAgent() string {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.UserAgent
}
func (p *Peer) GetLastDeclareBlock() *crypto.Hash {

	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.lastDeclareBlock
}
func (p *Peer) LastSent() uint64 {
	return atomic.LoadUint64(&p.lastSent)
}
func (p *Peer) LastReceived() uint64 {
	return atomic.LoadUint64(&p.lastReceive)
}

func (p *Peer) LocalVersionMsg() (*msg.VersionMessage, error) {
	var blockNumber int32
	if p.Config.NewBlock != nil {
		_, blockNumber, err := p.Config.NewBlock()
		if err != nil {
			return nil, err
		}
		log.Info("block number:%v", blockNumber)

	}
	remoteAddress := p.Address
	if p.Config.Proxy != "" {
		proxyAddress, _, err := net.SplitHostPort(p.Config.Proxy)
		if err != nil || p.Address.IP.String() == proxyAddress {
			remoteAddress = &msg.PeerAddress{
				Timestamp: time.Now(),
				IP:        net.IP([]byte{0, 0, 0, 0}),
			}
		}
	}
	localAddress := p.Address
	if p.Config.BestAddress != nil {
		localAddress = p.Config.BestAddress(p.Address)
	}
	nonce, err := utils.RandomUint64()
	if err != nil {
		return nil, err
	}
	sentNoces.Add(nonce, nonce)
	msg := msg.GetNewVersionMessage(localAddress, remoteAddress, nonce, blockNumber)
	msg.AddUserAgent(p.Config.UserAgent, p.Config.UserAgentVersion)
	msg.LocalAddress.ServicesFlag = protocol.SF_NODE_NETWORK_AS_FULL_NODE
	msg.ServiceFlag = p.Config.ServicesFlag
	msg.ProtocolVersion = p.ProtocolVersion
	msg.DisableRelayTx = p.Config.DisableRelayTx
	return msg, nil
}

func (p *Peer) HandleRemoteVersionMessage(versionMessage *msg.VersionMessage) error {
	if sentNoces.Exists(versionMessage.Nonce) {
		return errors.New("disconnecting peer connected to self")
	}
	if versionMessage.ProtocolVersion < uint32(protocol.MULTIPLE_ADDRESS_VERSION) {

		str := fmt.Sprintf("protocol version must be %d or greater", protocol.MULTIPLE_ADDRESS_VERSION)
		rejectMessage := msg.NewRejectMessage(msg.COMMAND_VERSION, msg.REJECT_OBSOLETE, str)
		err := p.WriteMessage(rejectMessage)
		return err

	}
	p.BlockStatusMutex.Lock()
	p.LastBlock = versionMessage.LastBlock
	p.StartingHeight = versionMessage.LastBlock
	p.TimeOffset = versionMessage.Timestamp.Unix() - time.Now().Unix()
	p.BlockStatusMutex.Unlock()

	p.PeerStatusMutex.Lock()
	p.ProtocolVersion = algorithm.MinUint32(p.ProtocolVersion, uint32(versionMessage.ProtocolVersion))
	p.VersionKnown = true
	log.Debug("Negotiated protocol version %d for peer %s", p.ProtocolVersion, p)
	p.Id = atomic.AddInt32(&nodeCount, 1)
	p.ServiceFlag = versionMessage.ServiceFlag
	p.UserAgent = versionMessage.UserAgent
	p.PeerStatusMutex.Unlock()
	return nil
}

func (p *Peer) WriteMessage(message msg.Message) error {
	if atomic.LoadInt32(&p.disconnect) != 0 {
		return nil
	}
	//todo func()string
	log.Debug("%v", log2.InitLogClosure(func() string {
		summary := msg.MessageSummary(message)
		if len(summary) > 0 {
			summary = fmt.Sprintf("(%s)", summary)
		}
		return fmt.Sprintf("Sending %v %s to %s", message.Command(), summary, p)

	}))
	log.Debug("%v", log2.InitLogClosure(func() string {
		return spew.Sdump(message)
	}))
	log.Debug("%v", log2.InitLogClosure(func() string {
		var buf bytes.Buffer
		_, err := msg.WriteMessage(&buf, message, p.ProtocolVersion, p.Config.ChainParams.BitcoinNet)
		if err != nil {
			return err.Error()
		}
		//todo what is mean spew
		return spew.Sdump(buf.Bytes())

	}))
	n, err := msg.WriteMessage(p.conn, msg, p.ProtocolVersion, p.Config.ChainParams.BitcoinNet)
	atomic.AddUint64(&p.lastSent, uint64(n))
	if p.Config.Listener.OnWrite != nil {
		p.Config.Listener.OnWrite(p, n, message, err)
	}
	return err

}
func (p *Peer) SendMessage(msg msg.Message, doneChan chan<- struct{}) {
	if !p.Connected() {
		if doneChan != nil {
			go func() {
				doneChan <- struct{}{}
			}()
		}
		return
	}
	p.OutputQueue <- msg.OutMessage{Message: msg, Done: doneChan}
}
func (p *Peer) SendAddrMessage(addresses []*msg.PeerAddress) ([]*msg.PeerAddress, error) {

	if len(addresses) == 0 {
		return nil, nil
	}
	length := len(addresses)
	addressMessage := msg.PeerAddressMessage{AddressList: make([]*msg.PeerAddress, 0, length)}
	if len(addressMessage.AddressList) > msg.MAX_ADDRESSES_COUNT {
		for i := range addressMessage.AddressList {
			j := rand.Intn(i + 1)
			addressMessage.AddressList[i], addressMessage.AddressList[j] = addressMessage.AddressList[j], addressMessage.AddressList[i]

		}
		addressMessage.AddressList = addressMessage.AddressList[:msg.MAX_ADDRESSES_COUNT]
	}
	p.SendMessage(addressMessage, nil)
	return addressMessage.AddressList, nil

}
func (p *Peer) Connected() bool {
	return atomic.LoadInt32(&p.connected) != 0 && atomic.LoadInt32(&p.disconnect) == 0

}

func (p *Peer) SendGetBlocks(locator []*crypto.Hash, stopHash *crypto.Hash) error {
	var beginHash *crypto.Hash
	if len(locator) > 0 {
		beginHash = locator[0]
	}
	p.GetBlocksLock.Lock()
	isDuplicate := p.GetBlocksStop != nil && p.GetBlocksBegin != nil && beginHash != nil && stopHash.IsEqual(p.GetBlocksStop) && beginHash.IsEqual(p.GetBlocksBegin)
	if isDuplicate {
		log.Warn("duplicate getblocks with  %v -> %v", beginHash, stopHash)
		return nil
	}
	p.GetBlocksLock.Unlock()

	getBlocksMessage := msg.NewGetBlocksMessage(stopHash)
	for _, hash := range locator {
		err := getBlocksMessage.AddBlockHash(hash)
		if err != nil {
			return err
		}
	}
	p.SendMessage(getBlocksMessage, nil)
	p.GetBlocksLock.Lock()

	p.GetBlocksBegin = beginHash
	p.GetBlocksStop = stopHash
	p.GetBlocksLock.Unlock()
	return nil

}

func (p *Peer) SendGetHeadersMessage(locator []*crypto.Hash, stopHash *crypto.Hash) error {
	var beginHash *crypto.Hash
	if len(locator) > 0 {
		beginHash = locator[0]
	}
	p.GetHeadersLock.Lock()
	isDuplicate := p.GetHeadersStop != nil && p.GetHeadersBegin != nil && beginHash != nil && stopHash.IsEqual(p.GetHeadersStop) && beginHash.IsEqual(p.GetHeadersBegin)
	p.GetHeadersLock.Unlock()
	if isDuplicate {
		log.Warn("duplicate  getheaders with begin hash %v", beginHash)
		return nil
	}

	message := msg.NewGetHeadersMessage()
	message.HashStop = stopHash
	for _, hash := range locator {
		err := message.AddBlockHash(hash)
		if err != nil {
			return err
		}
	}
	p.SendMessage(message, nil)
	p.GetHeadersLock.Lock()
	p.GetHeadersBegin = beginHash
	p.GetHeadersStop = stopHash
	p.GetHeadersLock.Unlock()
	return nil

}

func (p *Peer) SendRejectMessage(command string, code msg.RejectCode, reason string, hash *crypto.Hash, wait bool) {
	if p.VersionKnown && p.ProtocolVersion < protocol.REJECT_VERSION {
		return
	}
	var zeroHash crypto.Hash
	rejectMessage := msg.NewRejectMessage(command, code, reason)
	if command == msg.COMMAND_TX || command == msg.COMMAND_BLOCK {
		if hash == nil {
			log.Warn("sending a reject message for command type %v which should have specified a hash ", command)
			hash = &zeroHash
		}
		rejectMessage.Hash = hash
	}
	if !wait {
		p.SendMessage(rejectMessage, nil)
		return
	}
	doneChan := make(chan struct{}, 1)
	p.SendMessage(rejectMessage, doneChan)
	<-doneChan

}
func (p *Peer) IsValidBIP0111(command string) bool {
	if p.ServiceFlag&protocol.SF_NODE_BLOOM_FILTER != protocol.SF_NODE_BLOOM_FILTER {
		if p.ProtocolVersion >= protocol.BIP0111_VERSION {
			log.Debug("%s sent an unsupported %s request --disconnecting", p, command)
			p.Disconnect()

		} else {
			log.Debug("Ignoring %s request from %s -- bloom support is disabled", command, p)

		}
		return false
	}
	return true
}
func (p *Peer) HandlePingMessage(pingMessage *msg.PingMessage) {
	if p.ProtocolVersion > protocol.BIP0031_VERSION {
		pongMessage := msg.InitPondMessage(pingMessage.Nonce)
		p.SendMessage(pongMessage, nil)
	}
}
func (p *Peer) HandlePongMessage(pongMessage *msg.PongMessage) {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	if p.ProtocolVersion > protocol.BIP0031_VERSION && p.PingNonce != 0 && pongMessage.Nonce == p.PingNonce {
		p.PingMicros = (time.Now().Sub(p.PingTime).Nanoseconds()) / 1000
		p.PingNonce = 0
	}

}
func (p *Peer) Disconnect() {
	if atomic.AddInt32(&p.disconnect, 1) != 1 {
		return
	}
	log.Trace("Disconnecting %s", p)
	if atomic.LoadInt32(&p.connected) != 0 {
		p.conn.Close()
	}
	close(p.quit)
}

//
//func (p *Peer)ReadMessage()(msg.Message,[]byte,error){
//
//}
