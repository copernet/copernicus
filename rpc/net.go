package rpc

import (
	"fmt"
	"math"
	"time"

	"github.com/btcboost/copernicus/net/wire"
	"github.com/btcboost/copernicus/peer"
	"github.com/btcboost/copernicus/rpc/btcjson"
	"github.com/btcboost/copernicus/service"
	"github.com/btcboost/copernicus/util"
)

var netHandlers = map[string]commandHandler{
	"getconnectioncount": handleGetConnectionCount,
	"ping":               handlePing,
	"getpeerinfo":        handleGetPeerInfo,
	"addnode":            handleAddNode,
	"disconnectnode":     handleDisconnectNode,
	"getaddednodeinfo":   handleGetAddedNodeInfo,
	"getnettotals":       handleGetNetTotals,
	"getnetworkinfo":     handleGetnetWorkInfo,
	"setban":             handleSetBan,
	"listbanned":         handleListBanned,
	"clearbanned":        handleClearBanned,
	"setnetworkactive":   handleSetNetWorkActive,
}

func handleGetConnectionCount(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	getNodeCountCmd := service.GetNodeCount{}
	num, err := s.Handler.ProcessForRpc(getNodeCountCmd)
	if err != nil {

	}
	return num, nil
}

func handlePing(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	nonce := util.GetRand(math.MaxInt64)
	pingCmd := wire.NewMsgPing(nonce)
	_, err := s.Handler.ProcessForRpc(pingCmd)
	if err != nil {

	}
	return nil, nil
}

func handleGetPeerInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	getPeerInfoCmd := &service.GetPeersInfoMsg{}
	ret, _ := s.Handler.ProcessForRpc(getPeerInfoCmd) // todo Alert: match with return type
	peers := ret.([]*peer.Peer)
	//syncPeerID := s.cfg.SyncMgr.SyncPeerID()
	infos := make([]*btcjson.GetPeerInfoResult, 0, len(peers))
	for _, item := range peers {
		statsSnap := item.StatsSnapshot()
		info := &btcjson.GetPeerInfoResult{
			ID:              statsSnap.ID,
			Addr:            statsSnap.Addr,
			AddrLocal:       item.LocalAddr().String(),
			Services:        fmt.Sprintf("%08d", uint64(statsSnap.Services)),
			RelayTxes:       !item.IsTxRelayDisabled(), // TODO
			LastSend:        statsSnap.LastSend.Unix(),
			LastRecv:        statsSnap.LastRecv.Unix(),
			BytesSent:       statsSnap.BytesSent,
			BytesRecv:       statsSnap.BytesRecv,
			ConnTime:        statsSnap.ConnTime.Unix(),
			TimeOffset:      statsSnap.TimeOffset,
			PingTime:        float64(statsSnap.LastPingMicros),
			MinPing:         000,
			Version:         statsSnap.Version,
			SubVer:          statsSnap.UserAgent,
			Inbound:         statsSnap.Inbound,
			AddNode:         item.AddNode, // TODO
			StartingHeight:  statsSnap.StartingHeight,
			BanScore:        int32(item.BanScore()), // TODO
			SyncedHeaders:   000,                    // TODO
			SyncedBlocks:    000,                    // TODO
			Inflight:        []int{},                // TODO
			WhiteListed:     false,                  // TODO
			CashMagic:       false,                  // TODO
			BytesSendPerMsg: map[string]uint64{},    // TODO
			BytesRecvPerMsg: map[string]uint64{},    // TODO
		}
		if item.LastPingNonce() != 0 {
			wait := float64(time.Since(statsSnap.LastPingTime).Nanoseconds())
			// We actually want microseconds.
			info.PingWait = wait / 1000
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func handleAddNode(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.AddNodeCmd)
	_, err := s.Handler.ProcessForRpc(c)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: err.Error(),
		}
	}

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

func handleGetnetWorkInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
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

func handleSetNetWorkActive(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func registerNetRPCCommands() {
	for name, handler := range netHandlers {
		appendCommand(name, handler)
	}
}
