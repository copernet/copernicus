package server

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/peer"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/service"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
)

var (
	pctx    = context.TODO()
	ctxTest context.Context
	clfunc  context.CancelFunc
)

func init() {
	ctxTest, clfunc = context.WithCancel(pctx)
	defer clfunc()
}

func TestSetMsgHandle(t *testing.T) {
	SetMsgHandle(ctxTest, s.MsgChan, s)
}

func TestValueFromAmount(t *testing.T) {
	amounts := valueFromAmount(1000)
	assert.Equal(t, amounts, float64(1e-05))

	amounts = valueFromAmount(100000)
	assert.Equal(t, amounts, float64(0.001))

	amounts = valueFromAmount(-1000)
	assert.Equal(t, amounts, float64(-1e-05))

	amounts = valueFromAmount(0)
	assert.Equal(t, amounts, float64(0))
}

func TestGetNetworkInfo(t *testing.T) {
	ret, err := GetNetworkInfo()
	if err != nil {
		t.Error(err.Error())
	}

	assert.Equal(t, ret.Version, 1000000)
	assert.Equal(t, ret.ProtocolVersion, uint32(70013))
	assert.Equal(t, ret.LocalRelay, true)
	assert.Equal(t, ret.NetworkActive, true)
}

func TestProcessForRPC(t *testing.T) {
	getConnCountReq := &service.GetConnectionCountRequest{}
	getConnCountRsp, err := ProcessForRPC(getConnCountReq)
	assert.Nil(t, err)
	assert.Equal(t, &service.GetConnectionCountResponse{Count: 0}, getConnCountRsp)

	msgPingReq := &wire.MsgPing{}
	msgPingRsp, err := ProcessForRPC(msgPingReq)
	assert.Nil(t, err)
	assert.Nil(t, msgPingRsp)

	getPeersInfoReq := &service.GetPeersInfoRequest{}
	getPeersInfoRsp, err := ProcessForRPC(getPeersInfoReq)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(getPeersInfoRsp.([]RPCServerPeer)))

	getNetworkInfoCmdReq := &btcjson.GetNetworkInfoCmd{}
	getNetworkInfoCmdRsp, err := ProcessForRPC(getNetworkInfoCmdReq)
	assert.Nil(t, err)
	netWorkInfo, err := GetNetworkInfo()
	assert.Nil(t, err)
	assert.Equal(t, netWorkInfo, getNetworkInfoCmdRsp)

	// unused message
	getNetTotalsReq := &service.GetNetTotalsRequest{}
	_, err = ProcessForRPC(getNetTotalsReq)
	assert.Nil(t, err)
	setBanCmdReq := &btcjson.SetBanCmd{}
	_, err = ProcessForRPC(setBanCmdReq)
	assert.Nil(t, err)
	listBannedReq := &service.ListBannedRequest{}
	_, err = ProcessForRPC(listBannedReq)
	assert.Nil(t, err)
	clearBannedReq := &service.ClearBannedRequest{}
	_, err = ProcessForRPC(clearBannedReq)
	assert.Nil(t, err)
	invVectReq := &wire.InvVect{}
	_, err = ProcessForRPC(invVectReq)
	assert.Nil(t, err)

	//unknown
	_, err = ProcessForRPC(struct{}{})
	assert.NotNil(t, err)
}

func TestProcessForRPC_Connection(t *testing.T) {
	tests := []struct {
		name  string
		req   *btcjson.AddNodeCmd
		isErr bool
	}{
		{
			name: "test addnode add",
			req: &btcjson.AddNodeCmd{
				Addr:   "127.0.0.1:18834",
				SubCmd: "add",
			},
			isErr: false,
		},
		{
			name: "test addnode remove",
			req: &btcjson.AddNodeCmd{
				Addr:   "127.0.0.1:18834",
				SubCmd: "remove",
			},
			isErr: true,
		},
		{
			name: "test addnode onetry",
			req: &btcjson.AddNodeCmd{
				Addr:   "127.0.0.1:18834",
				SubCmd: "onetry",
			},
			isErr: false,
		},
		{
			name: "test addnode unknown",
			req: &btcjson.AddNodeCmd{
				Addr:   "127.0.0.1:18834",
				SubCmd: "unknown",
			},
			isErr: true,
		},
	}

	for _, test := range tests {
		t.Logf("testing %s\n", test.name)
		rsp, err := ProcessForRPC(test.req)
		assert.Nil(t, rsp)
		if test.isErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestProcessForRPC_DisConnection(t *testing.T) {
	tests := []struct {
		name  string
		req   *btcjson.DisconnectNodeCmd
		isErr bool
	}{
		{
			name: "test disconnect invalid address",
			req: &btcjson.DisconnectNodeCmd{
				Target: "test address",
			},
			isErr: true,
		},
		{
			name: "test disconnect valid address",
			req: &btcjson.DisconnectNodeCmd{
				Target: "127.0.0.1:18834",
			},
			isErr: false,
		},
		{
			name: "test disconnect node id",
			req: &btcjson.DisconnectNodeCmd{
				Target: "0",
			},
			isErr: false,
		},
	}

	for _, test := range tests {
		t.Logf("testing %s\n", test.name)
		rsp, err := ProcessForRPC(test.req)
		assert.Nil(t, rsp)
		if test.isErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestMsgHandle(t *testing.T) {
	execCount := make(map[string]int)
	peerCfg := &peer.Config{
		Listeners: peer.MessageListeners{
			OnWrite: func(p *peer.Peer, bytesWritten int, msg wire.Message, err error) {
				execCount["OnWrite"]++
			},
			OnRead: func(p *peer.Peer, bytesRead int, msg wire.Message, err error) {
				execCount["OnRead"]++
			},
			OnGetAddr: func(p *peer.Peer, msg *wire.MsgGetAddr) {
				execCount["OnGetAddr"]++
			},
			OnAddr: func(p *peer.Peer, msg *wire.MsgAddr) {
				execCount["OnAddr"]++
			},
			OnPing: func(p *peer.Peer, msg *wire.MsgPing) {
				execCount["OnPing"]++
			},
			OnPong: func(p *peer.Peer, msg *wire.MsgPong) {
				execCount["OnPong"]++
			},
			OnAlert: func(p *peer.Peer, msg *wire.MsgAlert) {
				execCount["OnAlert"]++
			},
			OnMemPool: func(p *peer.Peer, msg *wire.MsgMemPool) {
				execCount["OnMemPool"]++
			},
			OnTx: func(p *peer.Peer, msg *wire.MsgTx, done chan<- struct{}) {
				execCount["OnTx"]++
				done <- struct{}{}
			},
			OnBlock: func(p *peer.Peer, msg *wire.MsgBlock, buf []byte, done chan<- struct{}) {
				execCount["OnBlock"]++
				done <- struct{}{}
			},
			OnInv: func(p *peer.Peer, msg *wire.MsgInv) {
				execCount["OnInv"]++
			},
			OnHeaders: func(p *peer.Peer, msg *wire.MsgHeaders) {
				execCount["OnHeaders"]++
			},
			OnNotFound: func(p *peer.Peer, msg *wire.MsgNotFound) {
				execCount["OnNotFound"]++
			},
			OnGetData: func(p *peer.Peer, msg *wire.MsgGetData) {
				execCount["OnGetData"]++
			},
			OnGetBlocks: func(p *peer.Peer, msg *wire.MsgGetBlocks) {
				execCount["OnGetBlocks"]++
			},
			OnGetHeaders: func(p *peer.Peer, msg *wire.MsgGetHeaders) {
				execCount["OnGetHeaders"]++
			},
			OnFeeFilter: func(p *peer.Peer, msg *wire.MsgFeeFilter) {
				execCount["OnFeeFilter"]++
			},
			OnFilterAdd: func(p *peer.Peer, msg *wire.MsgFilterAdd) {
				execCount["OnFilterAdd"]++
			},
			OnFilterClear: func(p *peer.Peer, msg *wire.MsgFilterClear) {
				execCount["OnFilterClear"]++
			},
			OnFilterLoad: func(p *peer.Peer, msg *wire.MsgFilterLoad) {
				execCount["OnFilterLoad"]++
			},
			OnMerkleBlock: func(p *peer.Peer, msg *wire.MsgMerkleBlock) {
				execCount["OnMerkleBlock"]++
			},
			OnVersion: func(p *peer.Peer, msg *wire.MsgVersion) {
				execCount["OnVersion"]++
			},
			OnVerAck: func(p *peer.Peer, msg *wire.MsgVerAck) {
				execCount["OnVerAck"]++
			},
			OnReject: func(p *peer.Peer, msg *wire.MsgReject) {
				execCount["OnReject"]++
			},
			OnSendHeaders: func(p *peer.Peer, msg *wire.MsgSendHeaders) {
				execCount["OnSendHeaders"]++
			},
		},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &model.MainNetParams,
		Services:          wire.SFNodeBloom,
	}

	r, w := io.Pipe()
	inConn := &conn{raddr: "127.0.0.1:18334", Writer: w, Reader: r}
	inMsgChan := make(chan *peer.PeerMessage)
	SetMsgHandle(context.TODO(), inMsgChan, nil)
	inPeer := peer.NewInboundPeer(peerCfg)
	inPeer.AssociateConnection(inConn, inMsgChan, func(*peer.Peer) {})
	inPeer.SetAckReceived(true)

	ret := inPeer.WantsHeaders()
	assert.Equal(t, ret, false)

	ourNA := &wire.NetAddress{
		Services: s.services,
	}

	bhdr := block.BlockHeader{
		Version:       1,
		HashPrevBlock: util.Hash{},
		MerkleRoot:    util.Hash{},
		Time:          uint32(time.Now().Unix()),
		Bits:          1,
		Nonce:         1,
	}

	type unknownType struct {
		wire.MsgVerAck
	}

	tests := []struct {
		listener   string
		msg        wire.Message
		doCallBack bool
		sendDone   bool
	}{
		{
			"OnGetAddr",
			wire.NewMsgGetAddr(),
			true,
			true,
		},
		{
			"OnAddr",
			wire.NewMsgAddr(),
			true,
			true,
		},
		{
			"OnPing",
			wire.NewMsgPing(42),
			true,
			true,
		},
		{
			"OnPong",
			wire.NewMsgPong(42),
			true,
			true,
		},
		{
			"OnAlert",
			wire.NewMsgAlert([]byte("payload"), []byte("signature")),
			true,
			true,
		},
		{
			"OnMemPool",
			wire.NewMsgMemPool(),
			true,
			true,
		},
		{
			"OnTx",
			(*wire.MsgTx)(tx.NewTx(0, tx.TxVersion)),
			true,
			true,
		},
		{
			"OnBlock",
			(*wire.MsgBlock)(&block.Block{Header: bhdr}),
			true,
			true,
		},
		{
			"OnInv",
			wire.NewMsgInv(),
			true,
			true,
		},
		{
			"OnHeaders",
			wire.NewMsgHeaders(),
			true,
			true,
		},
		{
			"OnNotFound",
			wire.NewMsgNotFound(),
			true,
			true,
		},
		{
			"OnGetData",
			wire.NewMsgGetData(),
			true,
			true,
		},
		{
			"OnGetBlocks",
			wire.NewMsgGetBlocks(&util.Hash{}),
			true,
			true,
		},
		{
			"OnGetHeaders",
			wire.NewMsgGetHeaders(),
			true,
			true,
		},
		{
			"OnFeeFilter",
			wire.NewMsgFeeFilter(15000),
			true,
			true,
		},
		{
			"OnFilterAdd",
			wire.NewMsgFilterAdd([]byte{0x01}),
			true,
			true,
		},
		{
			"OnFilterClear",
			wire.NewMsgFilterClear(),
			true,
			true,
		},
		{
			"OnFilterLoad",
			wire.NewMsgFilterLoad([]byte{0x01}, 10, 0, wire.BloomUpdateNone),
			true,
			true,
		},
		{
			"OnMerkleBlock",
			wire.NewMsgMerkleBlock(&bhdr),
			true,
			true,
		},
		{
			"OnVersion",
			wire.NewMsgVersion(ourNA, inPeer.NA(), 1, 1),
			false, // not call the callback function in msghandle
			true,
		},
		{
			"OnVerAck",
			wire.NewMsgVerAck(),
			true,
			true,
		},
		{
			"OnReject",
			wire.NewMsgReject("block", errcode.RejectDuplicate, "dupe block"),
			true,
			true,
		},
		{
			"OnSendHeaders",
			wire.NewMsgSendHeaders(),
			true,
			true,
		},
		{
			"Unknown",
			&unknownType{},
			false,
			false,
		},
	}
	t.Logf("Running %d tests", len(tests))

	buf := make([]byte, 100)
	done := make(chan struct{}, 1)
	ctx := context.TODO()
	SetMsgHandle(ctx, s.MsgChan, s)

	for index, test := range tests {
		t.Logf("testing %d handler:%s", index, test.listener)
		peerMsg := peer.NewPeerMessage(inPeer, test.msg, buf, done)
		s.MsgChan <- peerMsg
		if test.sendDone {
			<-done
		}
		if test.doCallBack {
			assert.NotNil(t, execCount[test.listener])
			assert.Equal(t, 1, execCount[test.listener])
		}
	}
}
