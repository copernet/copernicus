package server

import (
	"net"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/peer"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/util"
)

// RPCServerPeer represents a peer for use with the RPC server.
//
// The interface contract requires that all of these methods are safe for
// concurrent access.
type RPCServerPeer interface {
	// ToPeer returns the underlying peer instance.
	ToPeer() *peer.Peer

	// IsTxRelayDisabled returns whether or not the peer has disabled
	// transaction relay.
	IsTxRelayDisabled() bool

	// BanScore returns the current integer value that represents how close
	// the peer is to being banned.
	BanScore() uint32

	// FeeFilter returns the requested current minimum fee rate for which
	// transactions should be announced.
	FeeFilter() int64
}

// rpcPeer provides a peer for use with the RPC server and implements the
// RPCServerPeer interface.
type rpcPeer serverPeer

var _ RPCServerPeer = (*rpcPeer)(nil)

// ToPeer returns the underlying peer instance.
//
// This function is safe for concurrent access and is part of the rpcserverPeer
// interface implementation.
func (p *rpcPeer) ToPeer() *peer.Peer {
	if p == nil {
		return nil
	}
	return (*serverPeer)(p).Peer
}

// IsTxRelayDisabled returns whether or not the peer has disabled transaction
// relay.
//
// This function is safe for concurrent access and is part of the rpcserverPeer
// interface implementation.
func (p *rpcPeer) IsTxRelayDisabled() bool {
	return (*serverPeer)(p).disableRelayTx
}

// BanScore returns the current integer value that represents how close the peer
// is to being banned.
//
// This function is safe for concurrent access and is part of the rpcserverPeer
// interface implementation.
func (p *rpcPeer) BanScore() uint32 {
	return (*serverPeer)(p).banScore.Int()
}

// FeeFilter returns the requested current minimum fee rate for which
// transactions should be announced.
//
// This function is safe for concurrent access and is part of the rpcserverPeer
// interface implementation.
func (p *rpcPeer) FeeFilter() int64 {
	return atomic.LoadInt64(&(*serverPeer)(p).feeFilter)
}

// RPCConnManager provides a connection manager for use with the RPC server and
// implements the rpcserverConnManager interface.
type RPCConnManager struct {
	server *Server
}

func NewRPCConnManager(s *Server) *RPCConnManager {
	return &RPCConnManager{server: s}
}

// Connect adds the provided address as a new outbound peer.  The permanent flag
// indicates whether or not to make the peer persistent and reconnect if the
// connection is lost.  Attempting to connect to an already existing peer will
// return an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) Connect(addr string, permanent bool) error {
	replyChan := make(chan error)
	cm.server.query <- connectNodeMsg{
		addr:      addr,
		permanent: permanent,
		reply:     replyChan,
	}
	return <-replyChan
}

// RemoveByID removes the peer associated with the provided id from the list of
// persistent peers.  Attempting to remove an id that does not exist will return
// an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) RemoveByID(id int32) error {
	replyChan := make(chan error)
	cm.server.query <- removeNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.ID() == id },
		reply: replyChan,
	}
	return <-replyChan
}

// RemoveByAddr removes the peer associated with the provided address from the
// list of persistent peers.  Attempting to remove an address that does not
// exist will return an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) RemoveByAddr(addr string) error {
	replyChan := make(chan error)
	cm.server.query <- removeNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.Addr() == addr },
		reply: replyChan,
	}
	return <-replyChan
}

// DisconnectByID disconnects the peer associated with the provided id.  This
// applies to both inbound and outbound peers.  Attempting to remove an id that
// does not exist will return an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) DisconnectByID(id int32) error {
	replyChan := make(chan error)
	cm.server.query <- disconnectNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.ID() == id },
		reply: replyChan,
	}
	return <-replyChan
}

// DisconnectByAddr disconnects the peer associated with the provided address.
// This applies to both inbound and outbound peers.  Attempting to remove an
// address that does not exist will return an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) DisconnectByAddr(addr string) error {
	replyChan := make(chan error)
	cm.server.query <- disconnectNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.Addr() == addr },
		reply: replyChan,
	}
	return <-replyChan
}

// ConnectedCount returns the number of currently connected peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) ConnectedCount() int32 {
	return cm.server.ConnectedCount()
}

// NetTotals returns the sum of all bytes received and sent across the network
// for all peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) NetTotals() (uint64, uint64) {
	return cm.server.NetTotals()
}

// ConnectedPeers returns an array consisting of all connected peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) ConnectedPeers() []RPCServerPeer {
	replyChan := make(chan []*serverPeer)
	cm.server.query <- getPeersMsg{reply: replyChan}
	serverPeers := <-replyChan

	// Convert to RPC server peers.
	peers := make([]RPCServerPeer, 0, len(serverPeers))
	for _, sp := range serverPeers {
		peers = append(peers, (*rpcPeer)(sp))
	}
	return peers
}

// PersistentPeers returns an array consisting of all the added persistent
// peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) PersistentPeers() []RPCServerPeer {
	replyChan := make(chan []*serverPeer)
	cm.server.query <- getAddedNodesMsg{reply: replyChan}
	serverPeers := <-replyChan

	// Convert to generic peers.
	peers := make([]RPCServerPeer, 0, len(serverPeers))
	for _, sp := range serverPeers {
		peers = append(peers, (*rpcPeer)(sp))
	}
	return peers
}

// BroadcastMessage sends the provided message to all currently connected peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) BroadcastMessage(msg wire.Message) {
	cm.server.BroadcastMessage(msg)
}

// AddRebroadcastInventory adds the provided inventory to the list of
// inventories to be rebroadcast at random intervals until they show up in a
// block.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *RPCConnManager) AddRebroadcastInventory(iv *wire.InvVect, data interface{}) {
	cm.server.AddRebroadcastInventory(iv, data)
}

func (cm *RPCConnManager) SetBan(c *btcjson.SetBanCmd) *btcjson.RPCError {
	if strings.Contains(c.SubNet, "/") {
		_, _, err := net.ParseCIDR(c.SubNet)
		if err != nil {
			return btcjson.NewRPCError(btcjson.RPCClientInvalidIPOrSubnet, "Error: Invalid IP/Subnet")
		}
	} else {
		ip := net.ParseIP(c.SubNet)
		if ip == nil {
			return btcjson.NewRPCError(btcjson.RPCClientInvalidIPOrSubnet, "Error: Invalid IP/Subnet")
		}
	}

	if c.Command == "add" {
		now := util.GetTimeSec()
		var endTime int64
		if c.BanTime != nil {
			if c.Absolute != nil && *c.Absolute {
				endTime = *c.BanTime
			} else {
				endTime = now + *c.BanTime
			}
		} else {
			endTime = now + conf.Cfg.P2PNet.BanDuration
		}
		hasBanned := cm.server.BanAddr(c.SubNet, now, endTime, BanReasonManuallyAdded)
		if hasBanned {
			return btcjson.NewRPCError(btcjson.RPCClientNodeAlreadyAdded, "Error: IP/Subnet already banned")
		}

	} else if c.Command == "remove" {
		hasBanned := cm.server.UnbanAddr(c.SubNet)
		if !hasBanned {
			return btcjson.NewRPCError(btcjson.RPCClientInvalidIPOrSubnet,
				"Error: Unban failed. Requested address/subnet was not previously banned.")
		}
	}

	return nil
}

func (cm *RPCConnManager) ListBanned() (*btcjson.ListBannedResult, *btcjson.RPCError) {
	bannedInfoList := cm.server.GetBannedInfo()
	retList := make([]btcjson.BannedInfo, 0, len(bannedInfoList))
	for _, info := range bannedInfoList {
		address := info.Address
		if !strings.Contains(address, "/") {
			address += "/32"
		}
		bannedInfo := btcjson.BannedInfo{
			Address:     address,
			BannedUntil: info.BanUntil,
			BanCreated:  info.CreateTime,
			BanReason:   banReasonToString(info.Reason),
		}
		retList = append(retList, bannedInfo)
	}
	result := btcjson.ListBannedResult(retList)
	sort.Sort(result)
	return &result, nil
}

func (cm *RPCConnManager) ClearBanned() {
	cm.server.ClearBanned()
}

func banReasonToString(banReason int) string {
	switch banReason {
	case BanReasonNodeMisbehaving:
		return "node misbehaving"
	case BanReasonManuallyAdded:
		return "manually added"
	}
	return "unknown"
}
