// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package service

import (
	"container/list"
	"context"
	//"fmt"
	"sync"

	"github.com/btcboost/copernicus/log"
	"github.com/btcboost/copernicus/model"
	//"github.com/btcboost/copernicus/model/block"
	//"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/chainparams"
	"github.com/btcboost/copernicus/net/wire"
	"github.com/btcboost/copernicus/peer"
	"github.com/btcboost/copernicus/util"
	"github.com/btcsuite/btcd/connmgr"
	//"github.com/btcboost/copernicus/internal/btcjson"
)

type MsgHandle struct {
	mtx           sync.Mutex
	recvFromNet   <-chan *peer.PeerMessage
	txAndBlockPro chan peer.PeerMessage
	chainparam    *chainparams.BitcoinParams
	//connect manager
	connManager connmgr.ConnManager

	// These fields should only be accessed from the blockHandler thread
	rejectedTxns    map[util.Hash]struct{}
	requestedTxns   map[util.Hash]struct{}
	requestedBlocks map[util.Hash]struct{}
	syncPeer        *peer.Peer
	peerStates      map[*peer.Peer]*peerSyncState

	// The following fields are used for headers-first mode.
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
	nextCheckpoint   *model.Checkpoint
}

// NewMsgHandle create a msgHandle for these message from peer And RPC.
// Then begins the core block handler which processes block and inv messages.
func NewMsgHandle(ctx context.Context, cmdCh <-chan *peer.PeerMessage) *MsgHandle {
	msg := &MsgHandle{mtx: sync.Mutex{}, recvFromNet: cmdCh}
	ctxChild, _ := context.WithCancel(ctx)

	go msg.startProcess(ctxChild)
	return msg
}

func (mh *MsgHandle) startProcess(ctx context.Context) {

out:
	for {
		select {
		case msg := <-mh.recvFromNet:
			peerFrom := msg.Peerp
			switch data := msg.Msg.(type) {
			case *wire.MsgVersion:
				peerFrom.PushRejectMsg(data.Command(), wire.RejectDuplicate, "duplicate version message",
					nil, false)
				break out
			case *wire.MsgVerAck:
				if peerFrom.VerAckReceived() {
					log.Info("Already received 'verack' from peer %v -- "+
						"disconnecting", peerFrom)
					break out
				}
				peerFrom.SetAckReceived(true)
				if peerFrom.Cfg.Listeners.OnVerAck != nil {
					peerFrom.Cfg.Listeners.OnVerAck(peerFrom, data)
				}
			case *wire.MsgGetAddr:
				if peerFrom.Cfg.Listeners.OnGetAddr != nil {
					peerFrom.Cfg.Listeners.OnGetAddr(peerFrom, data)
				}
			case *wire.MsgAddr:
				if peerFrom.Cfg.Listeners.OnAddr != nil {
					peerFrom.Cfg.Listeners.OnAddr(peerFrom, data)
				}
			case *wire.MsgPing:
				peerFrom.HandlePingMsg(data)
				if peerFrom.Cfg.Listeners.OnPing != nil {
					peerFrom.Cfg.Listeners.OnPing(peerFrom, data)
				}
			case *wire.MsgPong:
				peerFrom.HandlePongMsg(data)
				if peerFrom.Cfg.Listeners.OnPong != nil {
					peerFrom.Cfg.Listeners.OnPong(peerFrom, data)
				}
			case *wire.MsgAlert:
				if peerFrom.Cfg.Listeners.OnAlert != nil {
					peerFrom.Cfg.Listeners.OnAlert(peerFrom, data)
				}
			case *wire.MsgMemPool:
				if peerFrom.Cfg.Listeners.OnMemPool != nil {
					peerFrom.Cfg.Listeners.OnMemPool(peerFrom, data)
				}
			case *wire.MsgTx:
				if peerFrom.Cfg.Listeners.OnTx != nil {
					peerFrom.Cfg.Listeners.OnTx(peerFrom, data)
				}
			case *wire.MsgBlock:
				if peerFrom.Cfg.Listeners.OnBlock != nil {
					peerFrom.Cfg.Listeners.OnBlock(peerFrom, data, msg.Buf)
				}
			case *wire.MsgInv:
				if peerFrom.Cfg.Listeners.OnInv != nil {
					peerFrom.Cfg.Listeners.OnInv(peerFrom, data)
				}
			case *wire.MsgHeaders:
				if peerFrom.Cfg.Listeners.OnHeaders != nil {
					peerFrom.Cfg.Listeners.OnHeaders(peerFrom, data)
				}
			case *wire.MsgNotFound:
				if peerFrom.Cfg.Listeners.OnNotFound != nil {
					peerFrom.Cfg.Listeners.OnNotFound(peerFrom, data)
				}
			case *wire.MsgGetData:
				if peerFrom.Cfg.Listeners.OnGetData != nil {
					peerFrom.Cfg.Listeners.OnGetData(peerFrom, data)
				}
			case *wire.MsgGetBlocks:
				if peerFrom.Cfg.Listeners.OnGetBlocks != nil {
					peerFrom.Cfg.Listeners.OnGetBlocks(peerFrom, data)
				}
			case *wire.MsgGetHeaders:
				if peerFrom.Cfg.Listeners.OnGetHeaders != nil {
					peerFrom.Cfg.Listeners.OnGetHeaders(peerFrom, data)
				}
			case *wire.MsgFeeFilter:
				if peerFrom.Cfg.Listeners.OnFeeFilter != nil {
					peerFrom.Cfg.Listeners.OnFeeFilter(peerFrom, data)
				}
			case *wire.MsgFilterAdd:
				if peerFrom.Cfg.Listeners.OnFilterAdd != nil {
					peerFrom.Cfg.Listeners.OnFilterAdd(peerFrom, data)
				}
			case *wire.MsgFilterClear:
				if peerFrom.Cfg.Listeners.OnFilterClear != nil {
					peerFrom.Cfg.Listeners.OnFilterClear(peerFrom, data)
				}
			case *wire.MsgFilterLoad:
				if peerFrom.Cfg.Listeners.OnFilterLoad != nil {
					peerFrom.Cfg.Listeners.OnFilterLoad(peerFrom, data)
				}
			case *wire.MsgMerkleBlock:
				if peerFrom.Cfg.Listeners.OnMerkleBlock != nil {
					peerFrom.Cfg.Listeners.OnMerkleBlock(peerFrom, data)
				}
			case *wire.MsgReject:
				if peerFrom.Cfg.Listeners.OnReject != nil {
					peerFrom.Cfg.Listeners.OnReject(peerFrom, data)
				}
			case *wire.MsgSendHeaders:
				if peerFrom.Cfg.Listeners.OnSendHeaders != nil {
					peerFrom.Cfg.Listeners.OnSendHeaders(peerFrom, data)
				}
			default:
				log.Debug("Received unhandled message of type %v "+
					"from %v", data.Command())
			}
		case <-ctx.Done():
			log.Info("msgHandle service exit. function : startProcess")
			break out
		}
	}

}

/*
// Rpc process things
func (mh *MsgHandle) ProcessForRpc(message interface{}) (rsp interface{}, err error) {
	switch m := message.(type) {

	case *btcjson.AddNodeCmd:
		err = mh.NodeOpera(m)

	case *tx.Tx:
		mh.recvChannel <- m
		ret := <-mh.resultChannel
		switch r := ret.(type) {
		case error:
			return nil, r
		case []*tx.Tx:
			return r, nil
		}

	case *block.Block:
		mh.recvChannel <- m
		ret := <-mh.resultChannel
		switch r := ret.(type) {
		case error:
			return nil, r
		case BlockState:
			return r, nil
		}

	case GetConnectCount:
		return mh.connManager.ConnectedCount(), nil

	case *wire.MsgPing:
		return mh.connManager.BroadCast(), nil

	case GetAddedNodeInfoMsg:
		return mh.connManager.PersistentPeers(), nil
	}

	return nil, fmt.Errorf("Unknown command")
}
*/
