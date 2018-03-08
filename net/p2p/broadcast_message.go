package p2p

import "github.com/btcboost/copernicus/net/msg"

type BroadcastMessage struct {
	Message      *msg.Message
	ExcludePeers []*ServerPeer
}
