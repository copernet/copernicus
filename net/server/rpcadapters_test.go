package server

import (
	"testing"

	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/peer"
	"github.com/copernet/copernicus/util"
)

func TestRPCConnManager(t *testing.T) {
	rcm := NewRPCConnManager(s)
	if rcm.ConnectedCount() != 0 {
		t.Errorf("ConnectedCount() should be 0")
	}
	r, w := rcm.NetTotals()
	if r != 0 && w != 0 {
		t.Errorf("bytes sent and recv should be 0")
	}
	if len(rcm.ConnectedPeers()) != 0 {
		t.Errorf("ConnectedPeers should be 0")
	}
	if len(rcm.PersistentPeers()) != 0 {
		t.Errorf("PersistentPeers should be 0")
	}
	s.wg.Add(1)
	go s.rebroadcastHandler()
	rcm.BroadcastMessage(wire.NewMsgNotFound())
	iv := wire.NewInvVect(wire.InvTypeTx, &util.HashZero)
	rcm.AddRebroadcastInventory(iv, "test")
	if err := rcm.RemoveByID(0); err == nil {
		t.Errorf("remove not exist id, should return error")
	}
	if err := rcm.RemoveByAddr("127.0.0.1:8888"); err == nil {
		t.Errorf("remove not exist addr, should return error")
	}
	if err := rcm.DisconnectByID(0); err == nil {
		t.Errorf("disconnect not exist id, should return error")
	}
	if err := rcm.DisconnectByAddr("127.0.0.1:8888"); err == nil {
		t.Errorf("disconnect not exist addr, should return error")
	}
	if err := rcm.Connect("127.0.0.1:18444", false); err != nil {
		t.Errorf("self connect, should return nil")
	}
}

func TestRpcPeer(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config, false)
	sp := newServerPeer(s, false)
	sp.Peer = in
	rpcPeer := (*rpcPeer)(sp)
	if rpcPeer.ToPeer() != in {
		t.Errorf("ToPeer() failed")
	}
	if rpcPeer.IsTxRelayDisabled() {
		t.Errorf("rpcPeer Tx Relay should be false")
	}
	if rpcPeer.BanScore() != 0 {
		t.Errorf("rpcPeer BanScore should be 0")
	}
	if rpcPeer.FeeFilter() != 0 {
		t.Errorf("rpcPeer FeeFilter should be 0")
	}
}
