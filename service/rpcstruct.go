package service

// GetConnectionCountRequest Returns the number of connections to other nodes
type GetConnectionCountRequest struct{}

type GetConnectionCountResponse struct {
	Count int
}

// *wire.MsgPing Requests that a ping be sent to all other nodes, to measure
// ping time.
// return error if encountering any error

// *GetPeersInfoRequest Returns data about each connected network node as a json array
// of object.
// return one object implementing []rpc.RpcServerPeer interface
type GetPeersInfoRequest struct{}

// *btcjson.AddNodeCmd Attempts add or remove a node from the addnode list."
// Or try a connection to a node once.
// return error if encountering any error

// *btcjson.DisconnectNodeCmd Immediately disconnects from the specified peer node.
// Strictly one out of 'address' and 'nodeid' can be provided to identify the node.
// return error if encountering any error

// *btcjson.GetAddedNodeInfoCmd
// please note the item is optional.
// return []btcjson.GetAddedNodeInfoResult

// return btcjson.GetNetTotalsResult
type GetNetTotalsRequest struct{}

// *btcjson.GetnetWorkInfo
// return btcjson.GetNetworkInfoResult

// *btcjson.SetBanCmd
// return error if encountering any error

// return btcjson.ListBannedResult and any error
type ListBannedRequest struct{}

// return boolean
type ClearBannedRequest struct{}

// *InvVect to broadcast a tx inv message
// return error if encountering any error
