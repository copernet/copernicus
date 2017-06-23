package peer

import "copernicus/msg"

type MessageListener struct {
	OnRead  func(p *Peer, bytesRead int, message msg.Message, err error)
	OnWrite func(p *Peer, bytesWritten int, message msg.Message, err error)
	
	OnVersion func(p *Peer, versionMessage *msg.VersionMessage)
	OnGetAddr func(p *Peer, msg *msg.GetAddressMessage)
	OnAddr    func(p *Peer, msg *msg.AddressMessage)
	
	OnPing func(p *Peer, msg *msg.PingMessage)
	OnPong func(p *Peer, msg *msg.PongMessage)
	
	OnAlert func(p *Peer, msg *msg.AlertMessage)
	
	OnMemPool func(p *Peer, msg *msg.MempoolMessage)
	OnTx      func(p *Peer, msg *msg.TxMessage)
	OnBlock   func(p *Peer, msg *msg.BlockMessage, buf []byte)
	
	OnInv func(p *Peer, msg *msg.InventoryMessage)
	
	OnHeaders  func(p *Peer, msg *msg.HeadersMessage)
	OnNotFound func(p *Peer, msg *msg.NotFoundMessage)
	
	OnGetData func(p *Peer, msg *msg.GetDataMessage)
	
	OnGetBlocks  func(p *Peer, msg *msg.GetBlocksMessage)
	OnGetHeaders func(p *Peer, msg *msg.GetHeadersMessage)
	
	OnFilterAdd   func(p *Peer, msg *msg.FilterAddMessage)
	OnFilterClear func(p *Peer, msg *msg.FilterClearMessage)
	OnFilterLoad  func(p *Peer, msg *msg.FilterLoadMessage)
	OnMerkleBlock func(p *Peer, msg *msg.MerkleBlockMessage)
	
	OnVerAck      func(p *Peer, msg *msg.VersionACKMessage)
	OnReject      func(p *Peer, msg msg.RejectMessage)
	OnSendHeaders func(p *Peer, msg *msg.SendHeadersMessage)
}
