package rpc

import (
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/net/msg"
)

// rpcConnManager provides a connection manager for use with the RPC server and
// implements the rpcserverConnManager interface.
type rpcConnManager struct {
	//server *server
}

// Ensure rpcConnManager implements the rpcserverConnManager interface.
var _ ServerConnManager = &rpcConnManager{}

// Connect adds the provided address as a new outbound peer.  The permanent flag
// indicates whether or not to make the peer persistent and reconnect if the
// connection is lost.  Attempting to connect to an already existing peer will
// return an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) Connect(addr string, permanent bool) error {
	replyChan := make(chan error)
	/*	cm.server.query <- connectNodeMsg{
		addr:      addr,
		permanent: permanent,
		reply:     replyChan,
	}*/ // Todo
	return <-replyChan
}

// RemoveByID removes the peer associated with the provided id from the list of
// persistent peers.  Attempting to remove an id that does not exist will return
// an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) RemoveByID(id int32) error {
	replyChan := make(chan error)
	/*	cm.server.query <- removeNodeMsg{
		cmp:   func(sp *serverPeer) bool { return sp.ID() == id },
		reply: replyChan,
	}*/ // Todo
	return <-replyChan
}

// RemoveByAddr removes the peer associated with the provided address from the
// list of persistent peers.  Attempting to remove an address that does not
// exist will return an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) RemoveByAddr(addr string) error {
	replyChan := make(chan error)
	//cm.server.query <- removeNodeMsg{
	//	cmp:   func(sp *serverPeer) bool { return sp.Addr() == addr },
	//	reply: replyChan,
	//}             // Todo
	return <-replyChan
}

// DisconnectByID disconnects the peer associated with the provided id.  This
// applies to both inbound and outbound peers.  Attempting to remove an id that
// does not exist will return an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) DisconnectByID(id int32) error {
	replyChan := make(chan error)
	//cm.server.query <- disconnectNodeMsg{
	//	cmp:   func(sp *serverPeer) bool { return sp.ID() == id },
	//	reply: replyChan,
	//}            // Todo
	return <-replyChan
}

// DisconnectByAddr disconnects the peer associated with the provided address.
// This applies to both inbound and outbound peers.  Attempting to remove an
// address that does not exist will return an error.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) DisconnectByAddr(addr string) error {
	replyChan := make(chan error)
	//cm.server.query <- disconnectNodeMsg{
	//	cmp:   func(sp *serverPeer) bool { return sp.Addr() == addr },
	//	reply: replyChan,
	//}            // Todo
	return <-replyChan
}

// ConnectedCount returns the number of currently connected peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) ConnectedCount() int32 {
	//return cm.server.ConnectedCount()        Todo
	return 1
}

// NetTotals returns the sum of all bytes received and sent across the network
// for all peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) NetTotals() (uint64, uint64) {
	//return cm.server.NetTotals()   Todo
	return 0, 0
}

// ConnectedPeers returns an array consisting of all connected peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
/*func (cm *rpcConnManager) ConnectedPeers() []rpcserverPeer {
	replyChan := make(chan []*serverPeer)
	cm.server.query <- getPeersMsg{reply: replyChan}
	serverPeers := <-replyChan

	// Convert to RPC server peers.
	peers := make([]rpcserverPeer, 0, len(serverPeers))
	for _, sp := range serverPeers {
		peers = append(peers, (*rpcPeer)(sp))
	}
	return peers
}*/ // Todo

// PersistentPeers returns an array consisting of all the added persistent
// peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
/*func (cm *rpcConnManager) PersistentPeers() []rpcserverPeer {
	replyChan := make(chan []*serverPeer)
	cm.server.query <- getAddedNodesMsg{reply: replyChan}
	serverPeers := <-replyChan

	// Convert to generic peers.
	peers := make([]rpcserverPeer, 0, len(serverPeers))
	for _, sp := range serverPeers {
		peers = append(peers, (*rpcPeer)(sp))
	}
	return peers
}*/ // Todo

// BroadcastMessage sends the provided message to all currently connected peers.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) BroadcastMessage(msg msg.Message) {
	//cm.server.BroadcastMessage(msg)       // Todo
}

// AddRebroadcastInventory adds the provided inventory to the list of
// inventories to be rebroadcast at random intervals until they show up in a
// block.
//
// This function is safe for concurrent access and is part of the
// rpcserverConnManager interface implementation.
func (cm *rpcConnManager) AddRebroadcastInventory(iv *msg.InventoryVector, data interface{}) {
	//cm.server.AddRebroadcastInventory(iv, data)   // Todo
}

// RelayTransactions generates and relays inventory vectors for all of the
// passed transactions to all connected peers.
func (cm *rpcConnManager) RelayTransactions(txns []*mempool.TxEntry) { // todo btcd: *mempool.TxDesc
	//cm.server.relayTransactions(txns)       // Todo
}
