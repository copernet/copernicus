package peer

import msg2 "copernicus/msg"

type MessageListener struct {
	OnGetAddr func()
	OnRead    func(p *Peer, bytesRead int, msg msg2.Message, err error)
	OnWrite   func(p *Peer, bytesWritten int, msg msg2.Message, err error)
}
