package server

import (
	"testing"

	"github.com/copernet/copernicus/net/wire"
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
}
