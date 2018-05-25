package rpc

import (
	"fmt"
	"math"
	"time"

	"github.com/btcboost/copernicus/net/wire"
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
	request := service.GetConnectionCountRequest{}
	response, err := s.Handler.ProcessForRpc(request)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}
	count, ok := response.(*service.GetConnectionCountResponse)
	if !ok {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Server handle error",
		}
	}

	return count.Count, nil
}

func handlePing(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	nonce := util.GetRand(math.MaxInt64)
	pingCmd := wire.NewMsgPing(nonce)
	_, err := s.Handler.ProcessForRpc(pingCmd)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}

	return nil, nil
}

func handleGetPeerInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	getPeerInfoCmd := &service.GetPeersInfoRequest{}
	ret, _ := s.Handler.ProcessForRpc(getPeerInfoCmd) // todo Alert: match with return type
	peers := ret.([]RpcServerPeer)
	//syncPeerID := s.cfg.SyncMgr.SyncPeerID()
	infos := make([]*btcjson.GetPeerInfoResult, 0, len(peers))
	for _, item := range peers {
		statsSnap := item.ToPeer().StatsSnapshot()
		info := &btcjson.GetPeerInfoResult{
			ID:              statsSnap.ID,
			Addr:            statsSnap.Addr,
			AddrLocal:       item.ToPeer().LocalAddr().String(),
			Services:        fmt.Sprintf("%08d", uint64(statsSnap.Services)),
			RelayTxes:       !item.IsTxRelayDisabled(),
			LastSend:        statsSnap.LastSend.Unix(),
			LastRecv:        statsSnap.LastRecv.Unix(),
			BytesSent:       statsSnap.BytesSent,
			BytesRecv:       statsSnap.BytesRecv,
			ConnTime:        statsSnap.ConnTime.Unix(),
			TimeOffset:      statsSnap.TimeOffset,
			PingTime:        float64(statsSnap.LastPingMicros),
			MinPing:         statsSnap.MingPing,
			Version:         statsSnap.Version,
			SubVer:          statsSnap.UserAgent,
			Inbound:         statsSnap.Inbound,
			AddNode:         statsSnap.AddNode,
			StartingHeight:  statsSnap.StartingHeight,
			BanScore:        int32(item.BanScore()), // TODO
			SyncedHeaders:   statsSnap.SyncedHeaders,
			SyncedBlocks:    statsSnap.SyncedBlocks,
			Inflight:        []int{}, // TODO
			WhiteListed:     statsSnap.WhiteListed,
			CashMagic:       statsSnap.UsesCashMagic,
			BytesSendPerMsg: statsSnap.MapSendBytesPerMsgCmd,
			BytesRecvPerMsg: statsSnap.MapRecvBytesPerMsgCmd,
		}
		if item.ToPeer().LastPingNonce() != 0 {
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
	c := cmd.(*btcjson.DisconnectNodeCmd)

	_, err := s.Handler.ProcessForRpc(c)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: err.Error(),
		}
	}

	return nil, nil
}

func handleGetAddedNodeInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetAddedNodeInfoCmd)

	ret, err := s.Handler.ProcessForRpc(c)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}

	return ret, nil
}

func handleGetNetTotals(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	ret, err := s.Handler.ProcessForRpc(&service.GetNetTotalsRequest{})
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: err.Error(),
		}
	}
	return ret, nil
}

func handleGetnetWorkInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetNetworkInfoCmd)

	ret, err := s.Handler.ProcessForRpc(c)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}

	return ret, nil
}

func handleSetBan(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SetBanCmd)

	_, err := s.Handler.ProcessForRpc(c)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}

	return nil, nil
}

func handleListBanned(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	ret, err := s.Handler.ProcessForRpc(&service.ListBannedRequest{})
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}
	return ret, nil
}

func handleClearBanned(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	_, err := s.Handler.ProcessForRpc(&service.ClearBannedRequest{})
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}

	return nil, nil
}

func handleSetNetWorkActive(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SetNetWorkActiveCmd)

	ret, err := s.Handler.ProcessForRpc(c)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}

	return ret, nil
}

func registerNetRPCCommands() {
	for name, handler := range netHandlers {
		appendCommand(name, handler)
	}
}
