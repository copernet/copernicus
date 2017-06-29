package p2p

import "github.com/btccom/copernicus/msg"

type BroadcastMessage struct {
	Message      *msg.Message
	ExcludePeers []*ServerPeer
}
