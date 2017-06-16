package network

import "copernicus/msg"

type LocalAddress struct {
	PeerAddress *msg.PeerAddress
	score       AddressPriority
}
