package peer

import "copernicus/msg"

type MessageListener struct {
	OnRead           func(p *Peer, bytesRead int, message msg.Message, err error)
	OnWrite          func(p *Peer, bytesWritten int, message msg.Message, err error)
	OnVersionMessage func(p *Peer, versionMessage *msg.VersionMessage)
	//OnGetAddr func(p *Peer, msg *)
}
