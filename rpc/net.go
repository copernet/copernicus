package rpc

import (
	"github.com/btcboost/copernicus/internal/btcjson"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/service"
)

var netHandlers = map[string]commandHandler{
	"getconnectioncount": handleGetConnectionCount,
	"ping":               handlePing,
	"getpeerinfo":        handleGetPeerInfo,
	"addnode":            handleAddNode,
	"disconnectnode":     handleDisconnectNode,
	"getaddednodeinfo":   handleGetAddedNodeInfo,
	"getnettotals":       handleGetNetTotals,
	"getnetworkinfo":     handleGetnetWorkinfo,
	"setban":             handleSetBan,
	"listbanned":         handleListBanned,
	"clearbanned":        handleClearBanned,
	"setnetworkactive":   handleSetnetWorkActive,
}

func handleGetConnectionCount(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*
		return s.cfg.ConnMgr.ConnectedCount(), nil
	*/
	return nil, nil
}

func handlePing(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	// Ask server to ping \o_
	/*	nonce, err := utils.RandomUint64()
		if err != nil {
			return nil, internalRPCError("Not sending ping - failed to "+
				"generate nonce: "+err.Error(), "")
		}
		s.cfg.ConnMgr.BroadcastMessage(msg.InitPingMessage(nonce))

		return nil, nil*/
	return nil, nil
}

func handleGetPeerInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*
		peers := s.cfg.ConnMgr.ConnectedPeers()
		syncPeerID := s.cfg.SyncMgr.SyncPeerID()
		infos := make([]*btcjson.GetPeerInfoResult, 0, len(peers))
		for _, p := range peers {
			statsSnap := p.ToPeer().StatsSnapshot()
			info := &btcjson.GetPeerInfoResult{
				ID:             statsSnap.ID,
				Addr:           statsSnap.Addr,
				AddrLocal:      p.ToPeer().LocalAddr().String(),
				Services:       fmt.Sprintf("%08d", uint64(statsSnap.Services)),
				RelayTxes:      !p.IsTxRelayDisabled(),
				LastSend:       statsSnap.LastSend.Unix(),
				LastRecv:       statsSnap.LastRecv.Unix(),
				BytesSent:      statsSnap.BytesSent,
				BytesRecv:      statsSnap.BytesRecv,
				ConnTime:       statsSnap.ConnTime.Unix(),
				PingTime:       float64(statsSnap.LastPingMicros),
				TimeOffset:     statsSnap.TimeOffset,
				Version:        statsSnap.Version,
				SubVer:         statsSnap.UserAgent,
				Inbound:        statsSnap.Inbound,
				StartingHeight: statsSnap.StartingHeight,
				CurrentHeight:  statsSnap.LastBlock,
				BanScore:       int32(p.BanScore()),
				FeeFilter:      p.FeeFilter(),
				SyncNode:       statsSnap.ID == syncPeerID,
			}
			if p.ToPeer().LastPingNonce() != 0 {
				wait := float64(time.Since(statsSnap.LastPingTime).Nanoseconds())
				// We actually want microseconds.
				info.PingWait = wait / 1000
			}
			infos = append(infos, info)
		}
		return infos, nil
	*/

	return nil, nil
}

func handleAddNode(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.AddNodeCmd)

	addr := normalizeAddress(c.Addr, consensus.ActiveNetParams.DefaultPort)
	nodeCmd := service.NodeOperateMsg{addr, 0}
	_, err := s.Handler.ProcessForRpc(nodeCmd)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: err.Error(),
		}
	}

	// no data returned unless an error.
	return nil, nil
}

func handleDisconnectNode(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleGetAddedNodeInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*
		c := cmd.(*btcjson.GetAddedNodeInfoCmd)

		// Retrieve a list of persistent (added) peers from the server and
		// filter the list of peers per the specified address (if any).
		peers := s.cfg.ConnMgr.PersistentPeers()
		if c.Node != nil {
			node := *c.Node
			found := false
			for i, peer := range peers {
				if peer.ToPeer().Addr() == node {
					peers = peers[i : i+1]
					found = true
				}
			}
			if !found {
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCClientNodeNotAdded,
					Message: "Node has not been added",
				}
			}
		}

		// Without the dns flag, the result is just a slice of the addresses as
		// strings.
		if !c.DNS {
			results := make([]string, 0, len(peers))
			for _, peer := range peers {
				results = append(results, peer.ToPeer().Addr())
			}
			return results, nil
		}

		// With the dns flag, the result is an array of JSON objects which
		// include the result of DNS lookups for each peer.
		results := make([]*btcjson.GetAddedNodeInfoResult, 0, len(peers))
		for _, rpcPeer := range peers {
			// Set the "address" of the peer which could be an ip address
			// or a domain name.
			peer := rpcPeer.ToPeer()
			var result btcjson.GetAddedNodeInfoResult
			result.AddedNode = peer.Addr()
			result.Connected = btcjson.Bool(peer.Connected())

			// Split the address into host and port portions so we can do
			// a DNS lookup against the host.  When no port is specified in
			// the address, just use the address as the host.
			host, _, err := net.SplitHostPort(peer.Addr())
			if err != nil {
				host = peer.Addr()
			}

			var ipList []string
			switch {
			case net.ParseIP(host) != nil, strings.HasSuffix(host, ".onion"):
				ipList = make([]string, 1)
				ipList[0] = host
			default:
				// Do a DNS lookup for the address.  If the lookup fails, just
				// use the host.
				ips, err := btcdLookup(host)
				if err != nil {
					ipList = make([]string, 1)
					ipList[0] = host
					break
				}
				ipList = make([]string, 0, len(ips))
				for _, ip := range ips {
					ipList = append(ipList, ip.String())
				}
			}

			// Add the addresses and connection info to the result.
			addrs := make([]btcjson.GetAddedNodeInfoResultAddr, 0, len(ipList))
			for _, ip := range ipList {
				var addr btcjson.GetAddedNodeInfoResultAddr
				addr.Address = ip
				addr.Connected = "false"
				if ip == host && peer.Connected() {
					addr.Connected = directionString(peer.Inbound())
				}
				addrs = append(addrs, addr)
			}
			result.Addresses = &addrs
			results = append(results, &result)
		}
		return results, nil
	*/

	return nil, nil
}

func handleGetNetTotals(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*
			totalBytesRecv, totalBytesSent := s.cfg.ConnMgr.NetTotals()
			reply := &btcjson.GetNetTotalsResult{
				TotalBytesRecv: totalBytesRecv,
				TotalBytesSent: totalBytesSent,
				TimeMillis:     time.Now().UTC().UnixNano() / int64(time.Millisecond),
			}
		}
	*/
	return nil, nil
}

func handleGetnetWorkinfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleSetBan(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleListBanned(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleClearBanned(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleSetnetWorkActive(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func registerNetRPCCommands() {
	for name, handler := range netHandlers {
		appendCommand(name, handler)
	}
}
