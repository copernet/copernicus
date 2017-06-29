package p2p

import (
	"crypto"
)

type UpdatePeerHeightsMessage struct {
	newHash    *crypto.Hash
	newHeight  int32
	originPeer *ServerPeer
}
