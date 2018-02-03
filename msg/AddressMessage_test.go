package msg

import (
	"net"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/btcboost/copernicus/network"
	"github.com/btcboost/copernicus/protocol"
)

func TestAddressMessage_AddPeerAddress(t *testing.T) {
	pver := protocol.BitcoinProtocolVersion

	wantCmd := "addr"

	addressMessage := NewAddressMessage()

	if cmd := addressMessage.Command(); cmd != wantCmd {
		t.Errorf("NewMsgAddr: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	wantPayload := uint32(30729)
	maxPayload := addressMessage.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	// Ensure NetAddresses are added properly.
	tcpAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	peerAddress, err := network.NewPeerAddressWithNetAddr(tcpAddr, protocol.SFNodeNetworkAsFullNode)
	err = addressMessage.AddPeerAddress(peerAddress)
	if err != nil {
		t.Errorf("AddAddress: %v", err)
	}
	if addressMessage.AddressList[0] != peerAddress {
		t.Errorf("AddAddress: wrong address added - got %v, want %v",
			spew.Sprint(addressMessage.AddressList[0]), spew.Sprint(peerAddress))
	}

	addressMessage.ClearAddresses()
	if len(addressMessage.AddressList) != 0 {
		t.Errorf("ClearAddresses: address list is not empty - "+
			"got %v [%v], want %v", len(addressMessage.AddressList),
			spew.Sprint(addressMessage.AddressList[0]), 0)
	}

	for i := 0; i <= MaxAddressesCount+1; i++ {
		err = addressMessage.AddPeerAddress(peerAddress)
	}

	if err == nil {
		t.Errorf("AddAddress: expected error on too many addresses not received")
	}

	err = addressMessage.AddPeerAddresses(peerAddress)
	if err == nil {
		t.Errorf("AddAddresses: expected error on too many addresses not received")
	}

	wantPayload = uint32(26633)
	maxPayload = addressMessage.MaxPayloadLength(31401)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	wantPayload = uint32(35)
	maxPayload = addressMessage.MaxPayloadLength(208)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

}
