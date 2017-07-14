package p2p

import "github.com/btcboost/copernicus/msg"

type BroadcastMessage struct {
	Message      *msg.Message
	ExcludePeers []*ServerPeer
}
