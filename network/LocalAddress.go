package network

type LocalAddress struct {
	PeerAddress *PeerAddress
	score       AddressPriority
}
