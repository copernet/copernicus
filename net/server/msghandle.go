package server

// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

import (
	"context"
	"errors"
	"fmt"
	"github.com/copernet/copernicus/errcode"
	"strconv"
	"strings"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/peer"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/service"
	"github.com/copernet/copernicus/util"
)

type MsgHandle struct {
	recvMsgFromPeers <-chan *peer.PeerMessage
	*Server
}

var msgHandle *MsgHandle

// SetMsgHandle create a msgHandle for these message from peer And RPC.
// Then begins the core block handler which processes block and inv messages.
func SetMsgHandle(ctx context.Context, msgChan <-chan *peer.PeerMessage, server *Server) {
	msg := &MsgHandle{msgChan, server}
	go msg.startProcess(ctx)
	msgHandle = msg
}

func (mh *MsgHandle) startProcess(ctx context.Context) {

out:
	for {
		select {
		case msg := <-mh.recvMsgFromPeers:
			peerFrom := msg.Peerp
			switch data := msg.Msg.(type) {
			case *wire.MsgVersion:
				peerFrom.PushRejectMsg(data.Command(), errcode.RejectDuplicate, "duplicate version message",
					nil, false)
				peerFrom.Disconnect()
				msg.Done <- struct{}{}
			case *wire.MsgVerAck:
				if peerFrom.VerAckReceived() {
					log.Info("Already received 'verack' from peer %v -- "+
						"disconnecting", peerFrom)
					peerFrom.Disconnect()
				}
				peerFrom.SetAckReceived(true)
				if peerFrom.Cfg.Listeners.OnVerAck != nil {
					peerFrom.Cfg.Listeners.OnVerAck(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgGetAddr:
				if peerFrom.Cfg.Listeners.OnGetAddr != nil {
					peerFrom.Cfg.Listeners.OnGetAddr(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgAddr:
				if peerFrom.Cfg.Listeners.OnAddr != nil {
					peerFrom.Cfg.Listeners.OnAddr(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgPing:
				peerFrom.HandlePingMsg(data)
				if peerFrom.Cfg.Listeners.OnPing != nil {
					peerFrom.Cfg.Listeners.OnPing(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgPong:
				peerFrom.HandlePongMsg(data)
				if peerFrom.Cfg.Listeners.OnPong != nil {
					peerFrom.Cfg.Listeners.OnPong(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgAlert:
				if peerFrom.Cfg.Listeners.OnAlert != nil {
					peerFrom.Cfg.Listeners.OnAlert(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgMemPool:
				if peerFrom.Cfg.Listeners.OnTransferMsgToBusinessPro != nil {
					peerFrom.Cfg.Listeners.OnTransferMsgToBusinessPro(msg, msg.Done)
				} else if peerFrom.Cfg.Listeners.OnMemPool != nil {
					peerFrom.Cfg.Listeners.OnMemPool(peerFrom, data)
					msg.Done <- struct{}{}
				}

			case *wire.MsgTx:
				if peerFrom.Cfg.Listeners.OnTx != nil {
					peerFrom.Cfg.Listeners.OnTx(peerFrom, data, msg.Done)
				}

			case *wire.MsgBlock:
				if peerFrom.Cfg.Listeners.OnBlock != nil {
					peerFrom.Cfg.Listeners.OnBlock(peerFrom, data, msg.Buf, msg.Done)
				}

			case *wire.MsgInv:
				if peerFrom.Cfg.Listeners.OnInv != nil {
					peerFrom.Cfg.Listeners.OnInv(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgHeaders:
				if peerFrom.Cfg.Listeners.OnHeaders != nil {
					peerFrom.Cfg.Listeners.OnHeaders(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgNotFound:
				if peerFrom.Cfg.Listeners.OnNotFound != nil {
					peerFrom.Cfg.Listeners.OnNotFound(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgGetData:
				if peerFrom.Cfg.Listeners.OnTransferMsgToBusinessPro != nil {
					peerFrom.Cfg.Listeners.OnTransferMsgToBusinessPro(msg, msg.Done)
				} else if peerFrom.Cfg.Listeners.OnGetData != nil {
					peerFrom.Cfg.Listeners.OnGetData(peerFrom, data)
					msg.Done <- struct{}{}
				}

			case *wire.MsgGetBlocks:
				if peerFrom.Cfg.Listeners.OnTransferMsgToBusinessPro != nil {
					peerFrom.Cfg.Listeners.OnTransferMsgToBusinessPro(msg, msg.Done)
				} else if peerFrom.Cfg.Listeners.OnGetBlocks != nil {
					peerFrom.Cfg.Listeners.OnGetBlocks(peerFrom, data)
					msg.Done <- struct{}{}
				}

			case *wire.MsgGetHeaders:
				if peerFrom.Cfg.Listeners.OnGetHeaders != nil {
					peerFrom.Cfg.Listeners.OnGetHeaders(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgFeeFilter:
				if peerFrom.Cfg.Listeners.OnFeeFilter != nil {
					peerFrom.Cfg.Listeners.OnFeeFilter(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgFilterAdd:
				if peerFrom.Cfg.Listeners.OnFilterAdd != nil {
					peerFrom.Cfg.Listeners.OnFilterAdd(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgFilterClear:
				if peerFrom.Cfg.Listeners.OnFilterClear != nil {
					peerFrom.Cfg.Listeners.OnFilterClear(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgFilterLoad:
				if peerFrom.Cfg.Listeners.OnFilterLoad != nil {
					peerFrom.Cfg.Listeners.OnFilterLoad(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgMerkleBlock:
				if peerFrom.Cfg.Listeners.OnMerkleBlock != nil {
					peerFrom.Cfg.Listeners.OnMerkleBlock(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgReject:
				if peerFrom.Cfg.Listeners.OnReject != nil {
					peerFrom.Cfg.Listeners.OnReject(peerFrom, data)
				}
				msg.Done <- struct{}{}
			case *wire.MsgSendHeaders:
				if peerFrom.Cfg.Listeners.OnSendHeaders != nil {
					peerFrom.Cfg.Listeners.OnSendHeaders(peerFrom, data)
				}
				msg.Done <- struct{}{}
			default:
				log.Debug("Received unhandled message of type %v "+
					"from %v", data, data.Command())
			}
		case <-ctx.Done():
			log.Info("msgHandle service exit. function : startProcess")
			break out
		}
	}

}

// ProcessForRPC are RPC process things
func ProcessForRPC(message interface{}) (rsp interface{}, err error) {
	switch m := message.(type) {

	case *service.GetConnectionCountRequest:
		rsp := &service.GetConnectionCountResponse{
			Count: int(msgHandle.ConnectedCount()),
		}
		return rsp, nil

	case *wire.MsgPing:
		msgHandle.BroadcastMessage(m)
		return nil, nil

	case *service.GetPeersInfoRequest:
		return NewRPCConnManager(msgHandle.Server).ConnectedPeers(), nil

	case *btcjson.AddNodeCmd:
		cmd := message.(*btcjson.AddNodeCmd)
		var err error
		switch cmd.SubCmd {
		case "add":
			err = NewRPCConnManager(msgHandle.Server).Connect(cmd.Addr, true)
		case "remove":
			err = NewRPCConnManager(msgHandle.Server).RemoveByAddr(cmd.Addr)
		case "onetry":
			err = NewRPCConnManager(msgHandle.Server).Connect(cmd.Addr, false)
		default:
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidParameter,
				Message: "invalid subcommand for addnode",
			}
		}

		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidParameter,
				Message: err.Error(),
			}
		}
		return nil, nil

	case *btcjson.DisconnectNodeCmd:
		return

		//case *btcjson.GetAddedNodeInfoCmd:
		//	return msgHandle.connManager.PersistentPeers(), nil

	case *service.GetNetTotalsRequest:
		return

	case *btcjson.GetNetworkInfoCmd:
		return GetNetworkInfo()

	case *btcjson.SetBanCmd:
		return

	case *service.ListBannedRequest:
		return

	case *service.ClearBannedRequest:
		return

		//case *tx.Tx:
		//	msgHandle.recvChannel <- m
		//	ret := <-msgHandle.resultChannel
		//	switch r := ret.(type) {
		//	case error:
		//		return nil, r
		//	case []*tx.Tx:
		//		return r, nil
		//	}
		//
		//case *block.Block:
		//	msgHandle.recvChannel <- m
		//	ret := <-msgHandle.resultChannel
		//	switch r := ret.(type) {
		//	case error:
		//		return nil, r
		//	case BlockState:
		//		return r, nil
		//	}
	}

	return nil, errors.New("unknown rpc request")
}

func GetNetworkInfo() (*btcjson.GetNetworkInfoResult, error) {
	verNum := 0
	vers := strings.Split(conf.Cfg.Version, ".")
	for _, ver := range vers {
		subVer, err := strconv.Atoi(ver)
		if err == nil {
			verNum = verNum*1000 + subVer
		}
	}

	localAddrInfo := msgHandle.addrManager.GetAllLocalAddress()
	rpcLocalAddrList := make([]btcjson.LocalAddressesResult, 0, len(localAddrInfo))
	for _, localAddr := range localAddrInfo {
		rpcLocalAddr := btcjson.LocalAddressesResult{
			Address: localAddr.Na.IP.String(),
			Port:    localAddr.Na.Port,
			Score:   localAddr.Score,
		}
		rpcLocalAddrList = append(rpcLocalAddrList, rpcLocalAddr)
	}

	chainInfo := &btcjson.GetNetworkInfoResult{
		Version:          verNum,
		SubVersion:       "/Copernicus:" + conf.Cfg.Version + "/",
		ProtocolVersion:  wire.ProtocolVersion,
		LocalServices:    "0", // TODO:
		LocalRelay:       !conf.Cfg.P2PNet.BlocksOnly,
		TimeOffset:       util.GetTimeOffset(),
		Connections:      msgHandle.ConnectedCount(),
		NetworkActive:    true, // NOT support RPC 'setnetworkactive'
		Networks:         getNetworksInfo(),
		RelayFee:         valueFromAmount(mempool.GetInstance().GetMinFeeRate().SataoshisPerK),
		ExcessUtxoCharge: 0,
		LocalAddresses:   rpcLocalAddrList,
		Warnings:         "", // TODO: network warnings
	}
	return chainInfo, nil
}

func getNetworksInfo() []btcjson.NetworksResult {
	networkInfos := make([]btcjson.NetworksResult, 0)
	ipv4NetWork := btcjson.NetworksResult{
		Name:      "ipv4",
		Limited:   false,
		Reachable: true,
		//Proxy                     string `json:"proxy"`
		//ProxyRandomizeCredentials bool   `json:"proxy_randomize_credentials"`
	}
	ipv6NetWork := btcjson.NetworksResult{
		Name:      "ipv6",
		Limited:   false,
		Reachable: true,
		//Proxy                     string `json:"proxy"`
		//ProxyRandomizeCredentials bool   `json:"proxy_randomize_credentials"`
	}
	onionNetWork := btcjson.NetworksResult{
		Name:      "ipv4",
		Limited:   conf.Cfg.P2PNet.NoOnion,
		Reachable: !conf.Cfg.P2PNet.NoOnion,
		//Proxy                     string `json:"proxy"`
		//ProxyRandomizeCredentials bool   `json:"proxy_randomize_credentials"`
	}
	networkInfos = append(networkInfos, ipv4NetWork, ipv6NetWork, onionNetWork)
	return networkInfos
}

func valueFromAmount(sizeLimit int64) float64 {
	var nAbs int64
	var strFormat string
	if sizeLimit < 0 {
		nAbs = -sizeLimit
		strFormat = "-%d.%08d"
	} else {
		nAbs = sizeLimit
		strFormat = "%d.%08d"
	}
	quotient := nAbs / util.COIN
	remainder := nAbs % util.COIN
	strValue := fmt.Sprintf(strFormat, quotient, remainder)

	result, err := strconv.ParseFloat(strValue, 64)
	if err != nil {
		return 0
	}
	return result
}
