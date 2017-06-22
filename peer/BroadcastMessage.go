package peer

import "copernicus/msg"

type BroadcastMessage struct {
	Message      *msg.Message
	ExcludePeers []*ServerPeer
}
