package peer

import "time"

type PeerState struct {
	inboundPeers    map[int32]*ServerPeer
	outboundPeers   map[int32]*ServerPeer
	persistentPeers map[int32]*ServerPeer
	banned          map[string]time.Time
	outboundGroups  map[string]int
}

func (peerState *PeerState) Count() int {
	return len(peerState.inboundPeers) + len(peerState.outboundPeers) + len(peerState.persistentPeers)
}
func (peerState *PeerState) forAllOutboundPeers(closure func(serverPeer *ServerPeer)) {
	for _, peer := range peerState.outboundPeers {
		closure(peer)
	}
	for _, peer := range peerState.persistentPeers {
		closure(peer)
	}
}

func (peerState *PeerState) forAllPeers(closure func(sererPeer *ServerPeer)) {
	for _, peer := range peerState.inboundPeers {
		closure(peer)
	}
	peerState.forAllOutboundPeers(closure)
}
