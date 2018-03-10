package p2p

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/container"
	log2 "github.com/btcboost/copernicus/logger"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/net/network"
	"github.com/btcboost/copernicus/net/protocol"
	"github.com/btcboost/copernicus/utils"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

const (
	MaxInventoryTickleSize = 1000
	StallResponseTimeout   = 30 * time.Second
	StallTickInterval      = 15 * time.Second
	IdleTimeout            = 5 * time.Minute
	TrickleTimeout         = 10 * time.Second
	PingInterval           = 2 * time.Minute
	NegotiateTimeOut       = 30 * time.Second
	OutPutBufferSize       = 50
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Peer struct {
	Config           *PeerConfig
	ID               int32
	bytesReceived    uint64
	bytesSent        uint64
	lastReceive      int64
	lastSent         int64
	lastReceiveTime  time.Time
	lastSendTime     time.Time
	connected        int32
	disconnect       int32
	Inbound          bool
	BlockStatusMutex sync.RWMutex
	conn             net.Conn
	AddressString    string
	lastDeclareBlock *utils.Hash
	PeerStatusMutex  sync.RWMutex
	PeerAddress      *network.PeerAddress
	ServiceFlag      protocol.ServiceFlag

	UserAgent    string
	PingNonce    uint64
	PingTime     time.Time
	PingMicros   int64
	VersionKnown bool
	SentVerAck   bool
	GotVerAck    bool

	versionSent    bool
	verAckReceived bool

	knownInventory *container.LRUCache

	StallControl chan StallControlMessage

	ProtocolVersion      uint32
	LastBlock            int32
	ConnectedTime        time.Time
	TimeOffset           int64
	StartingHeight       int32
	SendHeadersPreferred bool
	OutputQueue          chan msg.OutMessage
	SendQueue            chan msg.OutMessage

	GetBlocksLock  sync.Mutex
	GetBlocksBegin *utils.Hash
	GetBlocksStop  *utils.Hash

	GetHeadersLock  sync.Mutex
	GetHeadersBegin *utils.Hash
	GetHeadersStop  *utils.Hash

	quit          chan struct{}
	inQuit        chan struct{}
	queueQuit     chan struct{}
	outQuit       chan struct{}
	sendDoneQueue chan struct{}
	outputInvChan chan *msg.InventoryVector
}

var log = logs.NewLogger()
var (
	sentNoces = container.NewLRUCache(50)
	nodeCount int32
)

func (p *Peer) String() string {
	directionString := "Inbound"
	if !p.Inbound {
		directionString = "outbound"
	}
	return fmt.Sprintf("%s (%s)", p.AddressString, directionString)
}
func (p *Peer) UpdateBlockHeight(newHeight int32) {
	p.BlockStatusMutex.Lock()
	log.Trace("Updating last block height of p2p %v from %v to %v", p.AddressString, p.LastBlock, newHeight)
	p.LastBlock = newHeight
	p.BlockStatusMutex.Unlock()
}

func (p *Peer) UpdateDeclareBlock(blackHash *utils.Hash) {
	log.Trace("Updating last block:%v form p2p %v", blackHash, p.AddressString)
	p.BlockStatusMutex.Lock()
	p.lastDeclareBlock = blackHash
	p.BlockStatusMutex.Unlock()

}
func (p *Peer) GetPeerID() int32 {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.ID

}
func (p *Peer) GetNetAddress() *network.PeerAddress {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.PeerAddress
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
func (p *Peer) GetLastDeclareBlock() *utils.Hash {

	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	return p.lastDeclareBlock
}
func (p *Peer) LastSent() uint64 {
	return atomic.LoadUint64(&p.bytesSent)
}
func (p *Peer) LastReceived() uint64 {
	return atomic.LoadUint64(&p.bytesReceived)
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
	remoteAddress := p.PeerAddress
	if p.Config.Proxy != "" {
		proxyAddress, _, err := net.SplitHostPort(p.Config.Proxy)
		if err != nil || p.PeerAddress.IP.String() == proxyAddress {
			remoteAddress = &network.PeerAddress{
				Timestamp: time.Now(),
				IP:        net.IP([]byte{0, 0, 0, 0}),
			}
		}
	}
	localAddress := p.PeerAddress
	if p.Config.BestAddress != nil {
		localAddress = p.Config.BestAddress(p.PeerAddress)
	}
	nonce, err := utils.RandomUint64()
	if err != nil {
		return nil, err
	}
	sentNoces.Add(nonce, nonce)
	message := msg.GetNewVersionMessage(localAddress, remoteAddress, nonce, blockNumber)
	message.AddUserAgent(p.Config.UserAgent, p.Config.UserAgentVersion)
	message.LocalAddress.ServicesFlag = protocol.SFNodeNetworkAsFullNode
	message.ServiceFlag = p.Config.ServicesFlag
	message.ProtocolVersion = p.ProtocolVersion
	message.DisableRelayTx = p.Config.DisableRelayTx
	return message, nil
}

func (p *Peer) HandleRemoteVersionMessage(versionMessage *msg.VersionMessage) error {
	if sentNoces.Exists(versionMessage.Nonce) {
		return errors.New("disconnecting p2p connected to self")
	}
	if versionMessage.ProtocolVersion < protocol.MultipleAddressVersion {

		str := fmt.Sprintf("protocol version must be %d or greater", protocol.MultipleAddressVersion)
		rejectMessage := msg.NewRejectMessage(msg.CommandVersion, msg.RejectObsolete, str)
		err := p.WriteMessage(rejectMessage)
		return err

	}
	p.BlockStatusMutex.Lock()
	p.LastBlock = versionMessage.LastBlock
	p.StartingHeight = versionMessage.LastBlock
	p.TimeOffset = versionMessage.Timestamp.Unix() - time.Now().Unix()
	p.BlockStatusMutex.Unlock()

	p.PeerStatusMutex.Lock()
	p.ProtocolVersion = container.MinUint32(p.ProtocolVersion, versionMessage.ProtocolVersion)
	p.VersionKnown = true
	log.Debug("Negotiated protocol version %d for p2p %s", p.ProtocolVersion, p)
	p.ID = atomic.AddInt32(&nodeCount, 1)
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
		return fmt.Sprintf("Sending %v %s to %s", message.Command(), summary, p.String())

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
	n, err := msg.WriteMessage(p.conn, message, p.ProtocolVersion, p.Config.ChainParams.BitcoinNet)
	atomic.AddUint64(&p.bytesSent, uint64(n))
	if p.Config.Listener.OnWrite != nil {
		p.Config.Listener.OnWrite(p, n, message, err)
	}
	return err

}
func (p *Peer) SendMessage(message msg.Message, doneChan chan<- struct{}) {
	if !p.Connected() {
		if doneChan != nil {
			go func() {
				doneChan <- struct{}{}
			}()
		}
		return
	}
	p.OutputQueue <- msg.OutMessage{Message: message, Done: doneChan}
}
func (p *Peer) SendAddrMessage(addresses []*network.PeerAddress) ([]*network.PeerAddress, error) {

	if len(addresses) == 0 {
		return nil, nil
	}
	length := len(addresses)
	addressMessage := msg.AddressMessage{AddressList: make([]*network.PeerAddress, 0, length)}
	if len(addressMessage.AddressList) > msg.MaxAddressesCount {
		for i := range addressMessage.AddressList {
			j := rand.Intn(i + 1)
			addressMessage.AddressList[i], addressMessage.AddressList[j] = addressMessage.AddressList[j], addressMessage.AddressList[i]

		}
		addressMessage.AddressList = addressMessage.AddressList[:msg.MaxAddressesCount]
	}
	p.SendMessage(&addressMessage, nil)
	return addressMessage.AddressList, nil

}
func (p *Peer) Connected() bool {
	return atomic.LoadInt32(&p.connected) != 0 && atomic.LoadInt32(&p.disconnect) == 0

}

func (p *Peer) SendGetBlocks(locator []*utils.Hash, stopHash *utils.Hash) error {
	var beginHash *utils.Hash
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

func (p *Peer) SendGetHeadersMessage(locator []*utils.Hash, stopHash *utils.Hash) error {
	var beginHash *utils.Hash
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

func (p *Peer) SendRejectMessage(command string, code msg.RejectCode, reason string, hash *utils.Hash, wait bool) {
	if p.VersionKnown && p.ProtocolVersion < protocol.RejectVersion {
		return
	}
	var zeroHash utils.Hash
	rejectMessage := msg.NewRejectMessage(command, code, reason)
	if command == msg.CommandTx || command == msg.CommandBlock {
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
	if p.ServiceFlag&protocol.SFNodeBloomFilter != protocol.SFNodeBloomFilter {
		if p.ProtocolVersion >= protocol.Bip0111Version {
			log.Debug("%s sent an unsupported %s request --disconnecting", p, command)
			p.Stop()

		} else {
			log.Debug("Ignoring %s request from %s -- bloom support is disabled", command, p)

		}
		return false
	}
	return true
}
func (p *Peer) HandlePingMessage(pingMessage *msg.PingMessage) {
	if p.ProtocolVersion > protocol.Bip0031Version {
		pongMessage := msg.InitPondMessage(pingMessage.Nonce)
		p.SendMessage(pongMessage, nil)
	}
}
func (p *Peer) HandlePongMessage(pongMessage *msg.PongMessage) {
	p.PeerStatusMutex.Lock()
	defer p.PeerStatusMutex.Unlock()
	if p.ProtocolVersion > protocol.Bip0031Version && p.PingNonce != 0 && pongMessage.Nonce == p.PingNonce {
		p.PingMicros = (time.Since(p.PingTime).Nanoseconds()) / 1000
		p.PingNonce = 0
	}

}

func (p *Peer) ReadMessage() (msg.Message, []byte, error) {
	n, message, buf, err := msg.ReadMessage(p.conn, p.ProtocolVersion, p.Config.ChainParams.BitcoinNet)
	atomic.AddUint64(&p.bytesReceived, uint64(n))
	if p.Config.Listener.OnRead != nil {
		p.Config.Listener.OnRead(p, n, message, err)
	}
	if err != nil {
		return nil, nil, err
	}
	log.Debug("%v", log2.InitLogClosure(func() string {
		summary := msg.MessageSummary(message)
		if len(summary) > 0 {
			summary = fmt.Sprintf("(%s)", summary)
		}
		return fmt.Sprintf("Received %v %v from %s",
			message.Command(), summary, p.String())

	}))
	log.Trace("%v", log2.InitLogClosure(func() string {
		return spew.Sdump(message)
	}))
	log.Trace("%v", log2.InitLogClosure(func() string {

		return spew.Sdump(buf)
	}))
	return message, buf, nil

}
func (p *Peer) IsAllowedReadError(err error) bool {
	if p.Config.ChainParams.BitcoinNet != utils.TestNet {
		return false
	}
	host, _, err := net.SplitHostPort(p.AddressString)
	if err != nil {
		return false
	}
	if host != "127.0.0.1" && host != "localhost" {
		return false
	}
	return true

}
func (p *Peer) shouldHandleReadError(err error) bool {
	if atomic.LoadInt32(&p.disconnect) != 0 {
		return false
	}
	if err == io.EOF {
		return false
	}
	if opErr, ok := err.(*net.OpError); ok && !opErr.Temporary() {
		return false
	}
	return true

}
func (p *Peer) maybeAddDeadline(pendingResponses map[string]time.Time, command string) {
	deadLine := time.Now().Add(StallResponseTimeout)
	switch command {
	case msg.CommandVersion:
		pendingResponses[msg.CommandVersionAck] = deadLine
	case msg.CommandMempool:
		pendingResponses[msg.CommandInv] = deadLine
	case msg.CommandGetBlocks:
		pendingResponses[msg.CommandInv] = deadLine
	case msg.CommandGetData:
		pendingResponses[msg.CommandBlock] = deadLine
		pendingResponses[msg.CommandTx] = deadLine
		pendingResponses[msg.CommandNotFound] = deadLine
	case msg.CommandGetHeaders:
		deadLine = time.Now().Add(StallResponseTimeout * 3)
		pendingResponses[msg.CommandHeaders] = deadLine

	}

}
func (p *Peer) stallHandler() {
	var handlerActive bool
	var handlerStartTime time.Time
	var deadlineOffset time.Duration
	pendingResponses := make(map[string]time.Time)
	stallTicker := time.NewTicker(StallTickInterval)
	defer stallTicker.Stop()
	var ioStopped bool
out:
	for {
		select {
		case stall := <-p.StallControl:
			switch stall.Command {
			case SccSendMessage:
				p.maybeAddDeadline(pendingResponses, stall.Message.Command())
			case SccReceiveMessage:
				switch messageCommand := stall.Message.Command(); messageCommand {
				case msg.CommandBlock:
					fallthrough
				case msg.CommandTx:
					fallthrough
				case msg.CommandNotFound:
					delete(pendingResponses, msg.CommandBlock)
					delete(pendingResponses, msg.CommandTx)
					delete(pendingResponses, msg.CommandNotFound)
				default:
					delete(pendingResponses, messageCommand)
				}
			case SccHandlerStart:
				if handlerActive {
					log.Warn("Received handler start control command while a handler is already active")
					continue
				}
				handlerActive = true
				handlerStartTime = time.Now()
			case SccHandlerDone:
				if !handlerActive {
					log.Warn("Received handler done control command when a handler is not already active")
					continue
				}

				duration := time.Since(handlerStartTime)
				deadlineOffset += duration
				handlerActive = false
			default:
				log.Warn("unsupported message command %v", stall.Command)

			}
		case <-stallTicker.C:
			now := time.Now()
			offset := deadlineOffset
			if handlerActive {
				offset += now.Sub(handlerStartTime)

			}
			for command, deadline := range pendingResponses {
				if now.Before(deadline.Add(offset)) {
					continue
				}
				log.Debug("p2p %s appears to be stalled or misbehaving,%s timeout -- disconnecting",
					p, command)
				p.Stop()
				break
			}
			deadlineOffset = 0
		case <-p.inQuit:
			if ioStopped {
				break out
			}
			ioStopped = true
		case <-p.outQuit:
			if ioStopped {
				break out
			}
			ioStopped = true

		}
	}

cleanup:
	for {
		select {
		case <-p.StallControl:
		default:
			break cleanup
		}
	}
	log.Trace("Peer stall handler done for %s", p)

}
func (p *Peer) inHandler() {
	idleTimer := time.AfterFunc(IdleTimeout, func() {
		log.Warn("Peer %s no answer for %s --disconnected", p, IdleTimeout)
		p.Stop()
	})
out:
	for atomic.LoadInt32(&p.disconnect) == 0 {
		readMessage, buf, err := p.ReadMessage()
		fmt.Println("readeMessage from other peer:")
		fmt.Println(buf)
		idleTimer.Stop()
		if err != nil {
			if p.IsAllowedReadError(err) {
				log.Error("Allowed test error from %s :%v", p, err)
				idleTimer.Reset(IdleTimeout)
				continue
			}
			if p.shouldHandleReadError(err) {
				errMessage := fmt.Sprintf("Can't read message from %s: %v", p.String(), err)
				log.Error(errMessage)
				p.SendRejectMessage("malformed", msg.RejectMalformed, errMessage, nil, true)

			}
			break out
		}
		atomic.StoreInt64(&p.lastReceive, time.Now().Unix())
		p.StallControl <- StallControlMessage{SccReceiveMessage, readMessage}
		p.StallControl <- StallControlMessage{SccHandlerStart, readMessage}
		//todo add other message
		switch message := readMessage.(type) {
		case *msg.VersionMessage:
			p.SendRejectMessage(message.Command(), msg.RejectDuplicate, "duplicate version message", nil, true)
			break out
		case *msg.PingMessage:
			p.HandlePingMessage(message)
		case *msg.PongMessage:
			p.HandlePongMessage(message)

		default:
			log.Debug("Received unhandled message of type %v from %v", readMessage.Command(), p)

		}

		p.StallControl <- StallControlMessage{SccHandlerDone, readMessage}
		idleTimer.Reset(IdleTimeout)

	}
	idleTimer.Stop()
	close(p.inQuit)

	log.Trace("Peer input handler done for %s", p)

}

func (p *Peer) queueHandler() {
	pendingMessages := list.New()
	invSendQueue := list.New()
	trickleTicker := time.NewTicker(TrickleTimeout)
	defer trickleTicker.Stop()
	waiting := false
	queuePacket := func(outMsg msg.OutMessage, list *list.List, waiting bool) bool {
		if !waiting {
			p.SendQueue <- outMsg
		} else {
			list.PushBack(outMsg)
		}
		return true
	}
out:
	for {
		select {
		case message := <-p.OutputQueue:
			waiting = queuePacket(message, pendingMessages, waiting)
		case <-p.sendDoneQueue:
			next := pendingMessages.Front()
			if next == nil {
				waiting = false
				continue
			}
			val := pendingMessages.Remove(next)
			p.SendQueue <- val.(msg.OutMessage)
		case iv := <-p.outputInvChan:
			if p.VersionKnown {
				invSendQueue.PushBack(iv)
			}
		case <-trickleTicker.C:
			if atomic.LoadInt32(&p.disconnect) != 0 || invSendQueue.Len() == 0 {
				continue
			}
			inventoryMessage := msg.NewInventoryMessageSizeHint(uint(invSendQueue.Len()))
			for e := invSendQueue.Front(); e != nil; e = invSendQueue.Front() {
				iv := invSendQueue.Remove(e).(*msg.InventoryVector)
				if p.knownInventory.Exists(iv) {
					continue
				}
				inventoryMessage.AddInventoryVector(iv)
				if len(inventoryMessage.InventoryList) >= MaxInventoryTickleSize {
					waiting = queuePacket(
						msg.OutMessage{Message: inventoryMessage},
						pendingMessages, waiting,
					)
					inventoryMessage = msg.NewInventoryMessageSizeHint(uint(invSendQueue.Len()))
				}
				p.knownInventory.Add(iv, iv)

			}
			if len(inventoryMessage.InventoryList) > 0 {
				waiting = queuePacket(
					msg.OutMessage{Message: inventoryMessage},
					pendingMessages, waiting)

			}
		case <-p.quit:
			break out
		}

	}
	for e := pendingMessages.Front(); e != nil; e = pendingMessages.Front() {
		val := pendingMessages.Remove(e)
		message := val.(msg.OutMessage)
		if message.Done != nil {
			message.Done <- struct{}{}
		}
	}
cleanup:
	for {
		select {
		case message := <-p.OutputQueue:
			if message.Done != nil {
				message.Done <- struct{}{}
			}
		case <-p.outputInvChan:
		default:
			break cleanup
		}
	}
	close(p.queueQuit)
	log.Trace("Peer queue handler done for %s", p)
}

func (p *Peer) outHandler() {
	pingTicker := time.NewTicker(PingInterval)
out:
	for {
		select {
		case message := <-p.SendQueue:
			switch m := message.Message.(type) {
			case *msg.PingMessage:
				if p.ProtocolVersion > protocol.Bip0031Version {
					p.PeerStatusMutex.Lock()
					p.PingNonce = m.Nonce
					p.PingTime = time.Now()
					p.PeerStatusMutex.Unlock()
				}
			}
			p.StallControl <- StallControlMessage{SccSendMessage, message.Message}
			err := p.WriteMessage(message.Message)
			if err != nil {
				p.Stop()
				if p.shouldHandleReadError(err) {
					log.Error("failed to send message to %s :%v", p, err)
				}
				if message.Done != nil {
					message.Done <- struct{}{}
				}
				continue
			}
			atomic.StoreInt64(&p.lastSent, time.Now().Unix())
			if message.Done != nil {
				message.Done <- struct{}{}
			}
			p.sendDoneQueue <- struct{}{}

		case <-pingTicker.C:
			nonce, err := utils.RandomUint64()
			if err != nil {
				log.Error("Not sending ping to %s :%v", p, err)
				continue
			}
			p.SendMessage(msg.InitPingMessage(nonce), nil)
		case <-p.quit:
			break out

		}

	}
	<-p.queueQuit
cleanup:
	for {
		select {
		case message := <-p.SendQueue:
			if message.Done != nil {
				message.Done <- struct{}{}
			}
		default:
			break cleanup
		}
	}
	close(p.outQuit)
	log.Trace("p2p output handler done for %s", p)

}
func (p *Peer) QueueInventory(inventoryVector *msg.InventoryVector) {
	if p.knownInventory.Exists(inventoryVector) {
		return
	}
	if !p.Connected() {
		return
	}
	p.outputInvChan <- inventoryVector

}

func (p *Peer) Connect(conn net.Conn) {
	if !atomic.CompareAndSwapInt32(&p.connected, 0, 1) {
		return
	}
	p.conn = conn
	p.ConnectedTime = time.Now()
	if p.Inbound {
		p.AddressString = p.conn.RemoteAddr().String()

		peerAddress, err := network.NewPeerAddressWithNetAddr(p.conn.RemoteAddr(), p.ServiceFlag)
		if err != nil {
			log.Error("Cannot create remote net AddressString :%v", err)
			p.Stop()
			return
		}
		p.PeerAddress = peerAddress
	}
	go func() {
		err := p.start()
		if err != nil {
			log.Warn("Can note start peer %v , err :%v", p, err)
			p.Stop()
		}

	}()

}
func (p *Peer) Disconnect() {
	if atomic.AddInt32(&p.disconnect, 1) != 1 {
		return
	}
	log.Trace("disconnecting %s", p)
	if atomic.LoadInt32(&p.connected) != 0 {
		p.conn.Close()
	}
	close(p.quit)
}
func (p *Peer) writeLocalVersionMessage() error {
	localVersion, err := p.LocalVersionMsg()
	if err != nil {
		return err
	}
	err = p.WriteMessage(localVersion)
	if err != nil {
		return err
	}
	p.PeerStatusMutex.Lock()
	p.versionSent = true
	p.PeerStatusMutex.Unlock()
	return nil

}
func (p *Peer) readRemoteVersionMessage() error {
	message, _, err := p.ReadMessage()
	if err != nil {
		return err
	}
	remoteVersionMessage, ok := message.(*msg.VersionMessage)
	if !ok {
		errStr := "A version message must precede all  others"
		log.Error(errStr)
		rejectMessage := msg.NewRejectMessage(message.Command(), msg.RejectMalformed, errStr)
		err := p.WriteMessage(rejectMessage)
		if err != nil {
			return err
		}
	}
	err = p.HandleRemoteVersionMessage(remoteVersionMessage)
	if err != nil {
		return err
	}
	if p.Config.Listener.OnVersion != nil {
		p.Config.Listener.OnVersion(p, remoteVersionMessage)
	}
	return nil
}

func (p *Peer) negotiateInboundProtocol() error {

	err := p.readRemoteVersionMessage()
	if err != nil {
		return err
	}
	err = p.writeLocalVersionMessage()
	return err
}
func (p *Peer) negotiateOutboundProtocol() error {
	err := p.writeLocalVersionMessage()
	if err != nil {
		return err
	}
	err = p.readRemoteVersionMessage()
	return err

}
func (p *Peer) start() error {
	log.Trace("start p2p %s ", p)
	negotiateErr := make(chan error)
	go func() {
		if p.Inbound {
			negotiateErr <- p.negotiateInboundProtocol()
		} else {
			negotiateErr <- p.negotiateOutboundProtocol()
		}
	}()
	select {
	case err := <-negotiateErr:
		if err != nil {
			return err
		}
	case <-time.After(NegotiateTimeOut):
		return errors.New("protocol negotiation timeout ")
	}
	log.Debug("Connected to %s", p.AddressString)
	go p.stallHandler()
	go p.inHandler()
	go p.queueHandler()
	go p.outHandler()
	p.SendMessage(&msg.VersionACKMessage{}, nil)
	return nil
}

func (p *Peer) WaitForDisconnect() {
	<-p.quit
}

func (p *Peer) Stop() {
	if atomic.AddInt32(&p.disconnect, 1) != 1 {
		return
	}
	log.Trace("Disconnecting %s", p)
	if atomic.LoadInt32(&p.connected) != 0 {
		p.conn.Close()
	}
	close(p.quit)
}

func newPeer(peerConfig *PeerConfig, inbound bool) *Peer {
	protocolVersion := protocol.MaxProtocolVersion
	if peerConfig.ProtocolVersion != 0 {
		protocolVersion = peerConfig.ProtocolVersion
	}
	if peerConfig.ChainParams == nil {
		peerConfig.ChainParams = &msg.TestNet3Params
	}
	perr := Peer{
		Inbound:         inbound,
		knownInventory:  container.NewLRUCache(protocol.MaxKnownInventory),
		StallControl:    make(chan StallControlMessage, 1),
		OutputQueue:     make(chan msg.OutMessage, OutPutBufferSize),
		SendQueue:       make(chan msg.OutMessage, 1),
		outputInvChan:   make(chan *msg.InventoryVector, OutPutBufferSize),
		inQuit:          make(chan struct{}),
		queueQuit:       make(chan struct{}),
		outQuit:         make(chan struct{}),
		quit:            make(chan struct{}),
		Config:          peerConfig,
		ServiceFlag:     peerConfig.ServicesFlag,
		ProtocolVersion: protocolVersion,
	}
	return &perr
}
func NewInboundPeer(peerConfig *PeerConfig) *Peer {
	return newPeer(peerConfig, true)
}
func NewOutboundPeer(peerConfig *PeerConfig, addressString string) (*Peer, error) {
	p := newPeer(peerConfig, false)
	p.AddressString = addressString
	host, portStr, err := net.SplitHostPort(addressString)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}
	if peerConfig.HostToAddressFunc != nil {
		peerAddress, err := peerConfig.HostToAddressFunc(host, uint16(port), peerConfig.ServicesFlag)
		if err != nil {
			return nil, err
		}
		p.PeerAddress = peerAddress

	} else {
		p.PeerAddress = network.NewPeerAddressIPPort(peerConfig.ServicesFlag, net.ParseIP(host), uint16(port))
	}
	return p, nil
}
