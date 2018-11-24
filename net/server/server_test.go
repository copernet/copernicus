package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/net/connmgr"
	"github.com/copernet/copernicus/net/upnp"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/peer"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
	"github.com/stretchr/testify/assert"
)

var s *Server
var dir string

func appInitMain(args []string) {
	conf.Cfg = conf.InitConfig(args)
	if conf.Cfg == nil {
		fmt.Println("please run `./copernicus -h` for usage.")
		os.Exit(0)
	}

	if conf.Cfg.P2PNet.TestNet {
		model.SetTestNetParams()
	} else if conf.Cfg.P2PNet.RegTest {
		model.SetRegTestParams()
	}

	fmt.Println("Current data dir:\033[0;32m", conf.DataDir, "\033[0m")

	//init log
	logDir := filepath.Join(conf.DataDir, log.DefaultLogDirname)
	if !conf.FileExists(logDir) {
		err := os.MkdirAll(logDir, os.ModePerm)
		if err != nil {
			panic("logdir create failed: " + err.Error())
		}
	}

	logConf := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
	}{
		FileName: logDir + "/" + conf.Cfg.Log.FileName + ".log",
		Level:    log.GetLevel(conf.Cfg.Log.Level),
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}
	log.Init(string(configuration))

	// Init UTXO DB
	utxoDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/chainstate",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	utxoConfig := utxo.UtxoConfig{Do: utxoDbCfg}
	utxo.InitUtxoLruTip(&utxoConfig)

	chain.InitGlobalChain()

	// Init blocktree DB
	blkDbCfg := &db.DBOption{
		FilePath:  conf.Cfg.DataDir + "/blocks/index",
		CacheSize: (1 << 20) * 8,
		Wipe:      conf.Cfg.Reindex,
	}
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: blkDbCfg}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	persist.InitPersistGlobal()

	// Load blockindex DB
	lblockindex.LoadBlockIndexDB()

	// when reindexing, we reuse the genesis block already on the disk
	if !conf.Cfg.Reindex {
		lchain.InitGenesisChain()
	}
	mempool.InitMempool()
	crypto.InitSecp256()
}

type mockNat struct{}

func (nat *mockNat) GetExternalAddress() (addr net.IP, err error) {
	return net.ParseIP("127.0.0.1"), nil
}

func (nat *mockNat) AddPortMapping(protocol string, externalPort, internalPort int, description string, timeout int) (mappedExternalPort int, err error) {
	return 18444, nil
}

func (nat *mockNat) DeletePortMapping(protocol string, externalPort, internalPort int) (err error) {
	return nil
}

func makeTestServer() (*Server, string, chan struct{}, error) {
	dir, err := ioutil.TempDir("", "server")
	if err != nil {
		return nil, "", nil, err
	}
	appInitMain([]string{"--datadir", dir, "--regtest"})
	c := make(chan struct{})
	conf.Cfg.P2PNet.ListenAddrs = []string{"127.0.0.1:0"}
	conf.Cfg.P2PNet.DisableBanning = false
	s, err := NewServer(model.ActiveNetParams, nil, c)
	if err != nil {
		return nil, "", nil, err
	}
	s.timeSource = util.GetTimeSource()
	s.nat = &mockNat{}
	return s, dir, c, nil
}

func TestMain(m *testing.M) {
	var err error
	s, dir, _, err = makeTestServer()
	if err != nil {
		fmt.Printf("makeTestServer(): %v\n", err)
		os.Exit(1)
	}
	flag.Parse()
	s.Start()
	exitCode := m.Run()
	s.Stop()
	s.WaitForShutdown()
	os.RemoveAll(dir)
	os.Exit(exitCode)
}

func TestCountBytes(t *testing.T) {
	s.AddBytesReceived(10)
	s.AddBytesSent(20)
	if r, s := s.NetTotals(); r != 10 || s != 20 {
		t.Fatalf("server received and sent bytes failed")
	}
}

func TestAddPeer(t *testing.T) {

	sp := newServerPeer(s, false)
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp.Peer = in

	s.AddPeer(sp)
}

func TestBanPeer(t *testing.T) {

	sp := newServerPeer(s, false)
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp.Peer = in

	s.BanPeer(sp)
}

func TestAddRebroadcastInventory(t *testing.T) {
	s.wg.Add(1)
	go s.rebroadcastHandler()
	iv := wire.NewInvVect(wire.InvTypeTx, &util.HashZero)
	s.AddRebroadcastInventory(iv, 10)
}

func TestRemoveRebroadcastInventory(t *testing.T) {
	s.wg.Add(1)
	go s.rebroadcastHandler()
	iv := wire.NewInvVect(wire.InvTypeTx, &util.HashZero)
	s.RemoveRebroadcastInventory(iv)
}

func makeTx() (*tx.Tx, error) {
	tmpTx := tx.Tx{}
	rawTxString := "0100000002f6b52b137227b1992a6289e2c1b265953b559d782faba905c209ddf1c7a48fb8" +
		"000000006b48304502200a0c787cb9c132584e640b7686a8f6a78d9c4a41201a0c7a139d5383970b39c50" +
		"22100d6fdc88b87328cdd772ed4dd9f15fea84c85968fe84308bb4a207ba03889cd680121020b76df009c" +
		"b91ce792ae00461e15a9340652c30d1b816129fc61246b3441a9e2ffffffff9285678bbc575493ea20372" +
		"3507fa22e37d775e766eccadde8894fc561602f2c000000006a47304402201ba951afbdeda2cb70483aac" +
		"b144b1ddd9db6fdbe4b6ccaec27c005b4bc4048f0220549ac7d19ddb6c37852bfd23ca2ed4aef429f2432" +
		"e27b309bed1e9217ce68d03012102ee67fdeb2f4484a2342db30f851808942016ff8df57f7e7798a14710" +
		"fa761590ffffffff02b58d1300000000001976a914d8a6c89e4207a50d7f57c6a02fc09f38113ccb6b88a" +
		"c6c400a000000000017a914abaad350c1c83d2f33cc35e8c1c886cd1287bda98700000000"

	originBytes, err := hex.DecodeString(rawTxString)
	if err != nil {
		return nil, err
	}
	if err := tmpTx.Unserialize(bytes.NewReader(originBytes)); err != nil {
		return nil, err
	}
	return &tmpTx, nil
}

func TestAnnounceNewTransactions(t *testing.T) {
	tmpTx, err := makeTx()
	if err != nil {
		t.Fatalf("makeTx() failed: %v\n", err)
	}
	s.AnnounceNewTransactions([]*mempool.TxEntry{{Tx: tmpTx}})
}

func TestBroadcastMessage(t *testing.T) {
	chn := make(chan struct{})
	svr, err := NewServer(model.ActiveNetParams, nil, chn)
	assert.Nil(t, err)

	r, w := io.Pipe()
	inConn := &conn{raddr: "127.0.0.1:18334", Writer: w, Reader: r}
	sp := newServerPeer(svr, false)
	sp.isWhitelisted = isWhitelisted(inConn.RemoteAddr())
	sp.Peer = peer.NewInboundPeer(newPeerConfig(sp))
	sp.AssociateConnection(inConn, svr.MsgChan, func(peer *peer.Peer) {
		svr.syncManager.NewPeer(peer)
	})

	svr.Start()
	defer svr.Stop()

	svr.AddPeer(sp)
	msg := wire.NewMsgPing(10)
	svr.BroadcastMessage(msg)
	// can not check the channel in peer

	ps := peerState{
		inboundPeers:    make(map[int32]*serverPeer),
		outboundPeers:   make(map[int32]*serverPeer),
		persistentPeers: make(map[int32]*serverPeer),
		bannedAddr:      make(map[string]*BannedInfo),
		bannedIPNet:     make(map[string]*BannedInfo),
		outboundGroups:  make(map[string]int),
	}
	ret := svr.handleAddPeerMsg(&ps, sp)
	assert.True(t, ret)

	bmsg := &broadcastMsg{message: msg}
	svr.handleBroadcastMsg(&ps, bmsg)

	bmsg.excludePeers = []*serverPeer{sp}
	svr.handleBroadcastMsg(&ps, bmsg)

	sp.Disconnect()
	svr.handleBroadcastMsg(&ps, bmsg)
}

func TestConnectedCount(t *testing.T) {
	chn := make(chan struct{})
	svr, err := NewServer(model.ActiveNetParams, nil, chn)
	assert.Nil(t, err)

	svr.Start()
	defer svr.Stop()

	c := svr.ConnectedCount()
	assert.Equal(t, int32(0), c)

	r, w := io.Pipe()
	inConn := &conn{raddr: "127.0.0.1:18334", Writer: w, Reader: r}
	sp := newServerPeer(svr, false)
	sp.isWhitelisted = isWhitelisted(inConn.RemoteAddr())
	sp.Peer = peer.NewInboundPeer(newPeerConfig(sp))
	sp.AssociateConnection(inConn, svr.MsgChan, func(peer *peer.Peer) {
		svr.syncManager.NewPeer(peer)
	})

	ps := peerState{
		inboundPeers:    make(map[int32]*serverPeer),
		outboundPeers:   make(map[int32]*serverPeer),
		persistentPeers: make(map[int32]*serverPeer),
		bannedAddr:      make(map[string]*BannedInfo),
		bannedIPNet:     make(map[string]*BannedInfo),
		outboundGroups:  make(map[string]int),
	}
	replyChan := make(chan int32)
	getMsg := getConnCountMsg{reply: replyChan}
	getReplyFunc := func(reply chan int32, quit chan struct{}, ret *int32) {
		*ret = <-reply
		close(quit)
	}

	finishChan := make(chan struct{})
	go getReplyFunc(replyChan, finishChan, &c)
	svr.handleQuery(&ps, getMsg)
	<-finishChan
	assert.Equal(t, int32(0), c)

	ret := svr.handleAddPeerMsg(&ps, sp)
	assert.True(t, ret)

	finishChan = make(chan struct{})
	go getReplyFunc(replyChan, finishChan, &c)
	svr.handleQuery(&ps, getMsg)
	<-finishChan
	assert.True(t, sp.Connected())
	assert.Equal(t, int32(1), c)

	sp.Disconnect()
	svr.peerDoneHandler(sp)
}

func TestOutboundGroupCount(t *testing.T) {
	if c := s.OutboundGroupCount(""); c != 0 {
		t.Errorf("OutboundGroupCount should be 0")
	}
}

func TestRelayInventory(t *testing.T) {

	iv := wire.NewInvVect(wire.InvTypeTx, &util.HashZero)
	s.RelayInventory(iv, 1)

	r, w := io.Pipe()
	inConn := &conn{raddr: "127.0.0.1:18334", Writer: w, Reader: r}
	sp := newServerPeer(s, false)
	sp.isWhitelisted = isWhitelisted(inConn.RemoteAddr())
	sp.Peer = peer.NewInboundPeer(newPeerConfig(sp))
	sp.AssociateConnection(inConn, s.MsgChan, func(peer *peer.Peer) {
		s.syncManager.NewPeer(peer)
	})
	ps := peerState{
		inboundPeers:    make(map[int32]*serverPeer),
		outboundPeers:   make(map[int32]*serverPeer),
		persistentPeers: make(map[int32]*serverPeer),
		bannedAddr:      make(map[string]*BannedInfo),
		bannedIPNet:     make(map[string]*BannedInfo),
		outboundGroups:  make(map[string]int),
	}
	ret := s.handleAddPeerMsg(&ps, sp)
	assert.True(t, ret)

	txMsgInvalid := relayMsg{invVect: iv, data: 1}
	s.handleRelayInvMsg(&ps, txMsgInvalid)

	txn := tx.NewTx(0, 1)
	txnSize := txn.SerializeSize()
	txFee := int64(1)
	txEntry := &mempool.TxEntry{Tx: txn, TxSize: int(txnSize), TxFee: txFee}
	txMsg := relayMsg{invVect: iv, data: txEntry}
	s.handleRelayInvMsg(&ps, txMsg)
	// can not check the channel in peer

	sp.setDisableRelayTx(true)
	s.handleRelayInvMsg(&ps, txMsg)

	sp.Disconnect()
}

func TestTransactionConfirmed(t *testing.T) {

	s.wg.Add(1)
	go s.rebroadcastHandler()

	tmpTx, err := makeTx()
	if err != nil {
		t.Fatalf("makeTx() failed: %v\n", err)
	}
	s.TransactionConfirmed(tmpTx)
}

func TestPeerState(t *testing.T) {
	ps := peerState{
		inboundPeers:    make(map[int32]*serverPeer),
		outboundPeers:   make(map[int32]*serverPeer),
		persistentPeers: make(map[int32]*serverPeer),
		bannedAddr:      make(map[string]*BannedInfo),
		bannedIPNet:     make(map[string]*BannedInfo),
		outboundGroups:  make(map[string]int),
	}
	ps.inboundPeers[1] = newServerPeer(nil, true)
	ps.inboundPeers[2] = newServerPeer(nil, false)
	ps.outboundPeers[10] = newServerPeer(nil, false)
	ps.persistentPeers[14] = newServerPeer(nil, true)
	if ps.Count() != 4 {
		t.Errorf("expect peer count equal 4")
	}
	count := 0
	ps.forAllPeers(func(sp *serverPeer) {
		count++
	})
	if count != 4 {
		t.Errorf("expect count equal 4")
	}
}

func TestNewestBlock(t *testing.T) {
	sp := newServerPeer(nil, true)
	hash, height, _ := sp.newestBlock()
	if hash.String() != "0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206" && height != 0 {
		t.Errorf("tip hash should be regtest genesis block hash")
	}
}

func TestKnownAddresses(t *testing.T) {
	sp := newServerPeer(nil, true)
	addrs := []*wire.NetAddress{
		{
			Timestamp: time.Unix(0x495fab29, 0),
			Services:  wire.SFNodeNetwork,
			IP:        net.ParseIP("127.0.0.1"),
			Port:      8333,
		},
		{
			Timestamp: time.Unix(0x5beda160, 0),
			Services:  wire.SFNodeNetwork,
			IP:        net.ParseIP("204.124.8.100"),
			Port:      8333,
		},
	}
	sp.addKnownAddresses(addrs)
	if !sp.addressKnown(&wire.NetAddress{
		Timestamp: time.Unix(0x5beda160, 0),
		Services:  wire.SFNodeNetwork,
		IP:        net.ParseIP("204.124.8.100"),
		Port:      8333,
	}) {
		t.Errorf("not find expect addr")
	}
}

func TestDisableRelayTx(t *testing.T) {
	sp := newServerPeer(nil, true)
	sp.setDisableRelayTx(true)
	if !sp.relayTxDisabled() {
		t.Errorf("relayTx should be true")
	}
}

type conn struct {
	io.Reader
	io.Writer
	io.Closer

	// local network, address for the connection.
	lnet, laddr string

	// remote network, address for the connection.
	rnet, raddr string
}

// LocalAddr returns the local address for the connection.
func (c conn) LocalAddr() net.Addr {
	return &addr{c.lnet, c.laddr}
}

// Remote returns the remote address for the connection.
func (c conn) RemoteAddr() net.Addr {
	return &addr{c.rnet, c.raddr}
}

// Close handles closing the connection.
func (c conn) Close() error {
	if c.Closer == nil {
		return nil
	}
	return c.Closer.Close()
}

func (c conn) SetDeadline(t time.Time) error      { return nil }
func (c conn) SetReadDeadline(t time.Time) error  { return nil }
func (c conn) SetWriteDeadline(t time.Time) error { return nil }

// addr mocks a network address
type addr struct {
	net, address string
}

func (m addr) Network() string { return m.net }
func (m addr) String() string  { return m.address }

// pipe turns two mock connections into a full-duplex connection similar to
// net.Pipe to allow pipe's with (fake) addresses.
func pipe(c1, c2 *conn) (*conn, *conn) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	c1.Writer = w1
	c1.Closer = w1
	c2.Reader = r1
	c1.Reader = r2
	c2.Writer = w2
	c2.Closer = w2

	return c1, c2
}

func TestInboundPeerConnected(t *testing.T) {
	inConn, _ := pipe(
		&conn{raddr: "10.0.0.1:8333"},
		&conn{raddr: "10.0.0.2:8333"},
	)
	s.inboundPeerConnected(inConn)
}

func TestOutboundPeerConnected(t *testing.T) {
	cq := &connmgr.ConnReq{
		Addr: &net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18555,
		},
		Permanent: true,
	}
	inConn, _ := pipe(
		&conn{raddr: "10.0.0.1:8333"},
		&conn{raddr: "10.0.0.2:8333"},
	)
	s.outboundPeerConnected(cq, inConn)

}

func TestPushAddrMsg(t *testing.T) {
	sp := newServerPeer(nil, true)
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp.Peer = in

	addrs := []*wire.NetAddress{
		{
			Timestamp: time.Unix(0x495fab29, 0),
			Services:  wire.SFNodeNetwork,
			IP:        net.ParseIP("127.0.0.1"),
			Port:      8333,
		},
		{
			Timestamp: time.Unix(0x5beda160, 0),
			Services:  wire.SFNodeNetwork,
			IP:        net.ParseIP("204.124.8.100"),
			Port:      8333,
		},
	}
	sp.pushAddrMsg(addrs)
}

func TestMergeCheckpoints(t *testing.T) {
	a := []model.Checkpoint{
		{Height: 478558, Hash: util.HashFromString("0000000000000000011865af4122fe3b144e2cbeea86142e8ff2fb4107352d43")},
		{Height: 504031, Hash: util.HashFromString("0000000000000000011ebf65b60d0a3de80b8175be709d653b4c1a1beeb6ab9c")},
		{Height: 530359, Hash: util.HashFromString("0000000000000000011ada8bd08f46074f44a8f155396f43e38acf9501c49103")},
		{Height: 105000, Hash: util.HashFromString("00000000000291ce28027faea320c8d2b054b2e0fe44a773f3eefb151d6bdc97")},
		{Height: 134444, Hash: util.HashFromString("00000000000005b12ffd4cd315cd34ffd4a594f430ac814c91184a0d42d2b0fe")},
		{Height: 168000, Hash: util.HashFromString("000000000000099e61ea72015e79632f216fe6cb33d7899acb35b75c8303b763")},
		{Height: 193000, Hash: util.HashFromString("000000000000059f452a5f7340de6682a977387c17010ff6e6c3bd83ca8b1317")},
		{Height: 295000, Hash: util.HashFromString("00000000000000004d9b4ef50f0f9d686fd69db2e03af35a100370c64632a983")},
		{Height: 11111, Hash: util.HashFromString("0000000069e244f73d78e8fd29ba2fd2ed618bd6fa2ee92559f542fdb26e7c1d")},
	}
	b := []model.Checkpoint{
		{Height: 11111, Hash: util.HashFromString("0000000069e244f73d78e8fd29ba2fd2ed618bd6fa2ee92559f542fdb26e7c1d")},
		{Height: 33333, Hash: util.HashFromString("000000002dd5588a74784eaa7ab0507a18ad16a236e7b1ce69f00d7ddfb5d0a6")},
		{Height: 74000, Hash: util.HashFromString("0000000000573993a3c9e41ce34471c079dcf5f52a0e824a81e7f953b8661a20")},
		{Height: 279000, Hash: util.HashFromString("0000000000000001ae8c72a0b0c301f67e3afca10e819efa9041e458e9bd7e40")},
		{Height: 210000, Hash: util.HashFromString("000000000000048b95347e83192f69cf0366076336c639f9b7228e9ba171342e")},
		{Height: 216116, Hash: util.HashFromString("00000000000001b4f4b433e81ee46494af945cf96014816a4e2370f11b23df4e")},
		{Height: 225430, Hash: util.HashFromString("00000000000001c108384350f74090433e7fcf79a606b8e797f065b130575932")},
		{Height: 250000, Hash: util.HashFromString("000000000000003887df1f29024b06fc2200b55f8af8f35453d7be294df2d214")},
	}
	c := mergeCheckpoints(a, b)
	if len(c) != 16 || !sort.IsSorted(checkpointSorter(c)) {
		t.Errorf("after merge checkpoint, length should be 16, result should be sorted")
	}
}

func TestIsWhitelisted(t *testing.T) {
	configWhitelists := conf.Cfg.P2PNet.Whitelists
	// restore
	defer func() {
		conf.Cfg.P2PNet.Whitelists = configWhitelists
	}()

	_, ipnet, err := net.ParseCIDR("127.0.0.1/8")
	assert.Nil(t, err)

	conf.Cfg.P2PNet.Whitelists = []*net.IPNet{}

	tests := []struct {
		addr    string
		isWhite bool
	}{
		{
			"127.0.0.20:18555",
			true,
		},
		{
			"128.0.0.1:18555",
			false,
		},
		{
			"127.0.0.1",
			false,
		},
		{
			":18555",
			false,
		},
	}

	for _, test := range tests {
		t.Logf("testing isWhitelisted:%s", test.addr)
		addr := simpleAddr{"net", test.addr}
		conf.Cfg.P2PNet.Whitelists = []*net.IPNet{}
		assert.False(t, isWhitelisted(addr))
		conf.Cfg.P2PNet.Whitelists = []*net.IPNet{ipnet}
		assert.Equal(t, test.isWhite, isWhitelisted(addr))
	}
}

func TestDynamicTickDuration(t *testing.T) {
	tests := []struct {
		in  time.Duration
		out time.Duration
	}{
		{3 * time.Second, time.Second},
		{10 * time.Second, 5 * time.Second},
		{30 * time.Second, time.Second * 15},
		{5 * time.Minute, time.Minute},
		{10 * time.Minute, 5 * time.Minute},
		{30 * time.Minute, 15 * time.Minute},
		{2 * time.Hour, time.Hour},
	}
	for i, test := range tests {
		if out := dynamicTickDuration(test.in); out != test.out {
			t.Errorf("failed at test %d, expect got %d, but got %d\n", i, test.out, out)
		}
	}
}

func TestOnVersion(t *testing.T) {
	lastBlock := int32(234234)
	tcpAddrMe := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	me := wire.NewNetAddress(tcpAddrMe, wire.SFNodeNetwork)
	tcpAddrYou := &net.TCPAddr{IP: net.ParseIP("192.168.0.1"), Port: 8333}
	you := wire.NewNetAddress(tcpAddrYou, wire.SFNodeNetwork)
	nonce, err := util.RandomUint64()
	if err != nil {
		t.Fatalf("RandomUint64: error generating nonce: %v", err)
	}
	config := peer.Config{}
	out, _ := peer.NewOutboundPeer(&config, "seed.bitcoinabc.org:8333")
	msg := wire.NewMsgVersion(me, you, nonce, lastBlock)

	sp := newServerPeer(s, false)
	sp.Peer = out
	sp.OnVersion(out, msg)
}

func TestOnMemPool(t *testing.T) {
	chn := make(chan struct{})
	svr, err := NewServer(model.ActiveNetParams, nil, chn)
	assert.Nil(t, err)

	svr.Start()
	defer svr.Stop()

	r, w := io.Pipe()
	inConn := &conn{raddr: "127.0.0.1:18334", Writer: w, Reader: r}
	sp := newServerPeer(svr, false)
	sp.isWhitelisted = isWhitelisted(inConn.RemoteAddr())
	sp.Peer = peer.NewInboundPeer(newPeerConfig(sp))
	sp.AssociateConnection(inConn, svr.MsgChan, func(peer *peer.Peer) {
		svr.syncManager.NewPeer(peer)
	})
	assert.True(t, sp.Connected())

	svr.services = wire.SFNodeBloom
	msg := wire.NewMsgMemPool()
	sp.OnMemPool(sp.Peer, msg)
	assert.True(t, sp.Connected())

	svr.services = wire.SFNodeNetwork
	sp.OnMemPool(sp.Peer, msg)
	assert.False(t, sp.Connected())
}

func TestOnTx(t *testing.T) {
	msgTx := (*wire.MsgTx)(tx.NewTx(0, 1))
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	done := make(chan struct{})
	sp.OnTx(in, msgTx, done)
	conf.Cfg.P2PNet.BlocksOnly = true
	sp.OnTx(in, msgTx, done)
	conf.Cfg.P2PNet.BlocksOnly = false
	<-done
}

func TestOnBlock(t *testing.T) {
	msgBlock := (*wire.MsgBlock)(&block.Block{})
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	done := make(chan struct{})
	buf := make([]byte, 100)
	sp.OnBlock(in, msgBlock, buf, done)
	<-done
}

func TestOnInv(t *testing.T) {
	hashStr := "3264bc2ac36a60840790ba1d475d01367e7c723da941069e9dc"
	blockHash, err := util.GetHashFromStr(hashStr)
	if err != nil {
		t.Errorf("GetHashFromStr: %v", err)
	}

	hashStr = "d28a3dc7392bf00a9855ee93dd9a81eff82a2c4fe57fbd42cfe71b487accfaf0"
	txHash, err := util.GetHashFromStr(hashStr)
	if err != nil {
		t.Errorf("GetHashFromStr: %v", err)
	}

	iv := wire.NewInvVect(wire.InvTypeBlock, blockHash)
	iv2 := wire.NewInvVect(wire.InvTypeTx, txHash)

	msgInv := wire.NewMsgInv()
	msgInv.AddInvVect(iv)
	msgInv.AddInvVect(iv2)
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	sp.OnInv(in, msgInv)
	conf.Cfg.P2PNet.BlocksOnly = true
	sp.OnInv(in, msgInv)
	conf.Cfg.P2PNet.BlocksOnly = false
}

func TestOnHeaders(t *testing.T) {
	msgHeaders := wire.NewMsgHeaders()
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	sp.OnHeaders(in, msgHeaders)
}

func TestOnGetData(t *testing.T) {
	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	hashStr := "3264bc2ac36a60840790ba1d475d01367e7c723da941069e9dc"
	blockHash, err := util.GetHashFromStr(hashStr)
	if err != nil {
		t.Errorf("GetHashFromStr: %v", err)
	}

	hashStr = "d28a3dc7392bf00a9855ee93dd9a81eff82a2c4fe57fbd42cfe71b487accfaf0"
	txHash, err := util.GetHashFromStr(hashStr)
	if err != nil {
		t.Errorf("GetHashFromStr: %v", err)
	}

	iv := wire.NewInvVect(wire.InvTypeBlock, blockHash)
	iv2 := wire.NewInvVect(wire.InvTypeTx, txHash)

	m1 := wire.NewMsgGetData()
	m2 := wire.NewMsgGetData()
	m1.AddInvVect(iv)
	m2.AddInvVect(iv2)

	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	go func() {
		sp.OnGetData(in, m1)
		sp.OnGetData(in, m2)
	}()
}

func TestTransferMsgToBusinessPro(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	mempoolMsg := wire.NewMsgMemPool()
	getdataMsg := wire.NewMsgGetData()
	getblocksMsg := wire.NewMsgGetBlocks(&util.HashZero)
	buf := make([]byte, 100)
	done := make(chan struct{})
	msgs := []*peer.PeerMessage{
		peer.NewPeerMessage(in, mempoolMsg, buf, done),
		peer.NewPeerMessage(in, getdataMsg, buf, done),
		peer.NewPeerMessage(in, getblocksMsg, buf, done),
	}
	go func() {
		for range done {
		}
	}()
	for _, msg := range msgs {
		sp.TransferMsgToBusinessPro(msg, done)
	}
}

func TestOnGetBlocks(t *testing.T) {
	// Block 99499 hash.
	hashStr := "2710f40c87ec93d010a6fd95f42c59a2cbacc60b18cf6b7957535"
	hashLocator, err := util.GetHashFromStr(hashStr)
	if err != nil {
		t.Errorf("GetHashFromStr: %v", err)
	}

	// Block 99500 hash.
	hashStr = "2e7ad7b9eef9479e4aabc65cb831269cc20d2632c13684406dee0"
	hashLocator2, err := util.GetHashFromStr(hashStr)
	if err != nil {
		t.Errorf("GetHashFromStr: %v", err)
	}

	// Block 100000 hash.
	hashStr = "3ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506"
	hashStop, err := util.GetHashFromStr(hashStr)
	if err != nil {
		t.Errorf("GetHashFromStr: %v", err)
	}

	msgGetBlocks := wire.NewMsgGetBlocks(hashStop)
	msgGetBlocks.AddBlockLocatorHash(hashLocator2)
	msgGetBlocks.AddBlockLocatorHash(hashLocator)
	msgGetBlocks.ProtocolVersion = wire.BIP0035Version
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	sp.OnGetBlocks(in, msgGetBlocks)
}

func TestOnFilterAdd(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	data := []byte{0x01, 0x02}
	msg := wire.NewMsgFilterAdd(data)
	sp.OnFilterAdd(in, msg)
}

func TestOnFeeFilter(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	tests := []*wire.MsgFeeFilter{
		wire.NewMsgFeeFilter(0),
		wire.NewMsgFeeFilter(323),
		wire.NewMsgFeeFilter(util.MaxSatoshi),
	}
	for _, test := range tests {
		sp.OnFeeFilter(in, test)
	}
}

func TestOnFilterClear(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	msg := wire.NewMsgFilterClear()
	sp.OnFilterClear(in, msg)
}

func TestOnFilterLoad(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	data := []byte{0x01, 0x02}
	msg := wire.NewMsgFilterLoad(data, 10, 0, 0)

	sp.OnFilterLoad(in, msg)
}

func TestOnGetAddr(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	msg := wire.NewMsgGetAddr()
	sp.OnGetAddr(in, msg)
}

func TestOnAddr(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	msg := wire.NewMsgAddr()
	tcpAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8333}
	na := wire.NewNetAddress(tcpAddr, wire.SFNodeNetwork)
	err := msg.AddAddress(na)
	if err != nil {
		t.Errorf("AddAddress: %v", err)
	}
	sp.OnAddr(in, msg)
}

func TestOnReadWrite(t *testing.T) {
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	atomic.StoreUint64(&s.bytesReceived, 0)
	atomic.StoreUint64(&s.bytesSent, 0)
	sp.OnRead(nil, 10, nil, nil)
	sp.OnWrite(nil, 20, nil, nil)
	if r, w := sp.server.NetTotals(); r != 10 || w != 20 {
		t.Errorf("set read write number failed")
	}
}

func TestRandomUint16Number(t *testing.T) {
	tests := []struct {
		in uint16
	}{
		{in: 50},
		{in: 100},
		{in: 1000},
		{in: 65000},
	}
	for _, test := range tests {
		if randomUint16Number(test.in) > test.in {
			t.Errorf("randomUint16Number failed return expected value")
		}
	}
}

func TestHandleAddPeerMsg(t *testing.T) {
	config := peer.Config{}
	out, _ := peer.NewOutboundPeer(&config, "192.168.1.32:8333")
	sp := newServerPeer(s, false)
	sp.Peer = out
	ps := peerState{
		inboundPeers:    make(map[int32]*serverPeer),
		outboundPeers:   make(map[int32]*serverPeer),
		persistentPeers: make(map[int32]*serverPeer),
		bannedAddr:      make(map[string]*BannedInfo),
		bannedIPNet:     make(map[string]*BannedInfo),
		outboundGroups:  make(map[string]int),
	}
	ret := s.handleAddPeerMsg(&ps, nil)
	assert.False(t, ret)

	ret = s.handleAddPeerMsg(&ps, sp)
	assert.True(t, ret)

	sp.persistent = true
	ret = s.handleAddPeerMsg(&ps, sp)
	assert.True(t, ret)

	// check full
	for i := 0; i < conf.Cfg.P2PNet.MaxPeers; i++ {
		ps.persistentPeers[int32(i)] = sp
	}
	ret = s.handleAddPeerMsg(&ps, sp)
	assert.False(t, ret)
	ps.persistentPeers = make(map[int32]*serverPeer)

	// check ban peer
	host, _, err := net.SplitHostPort(sp.Addr())
	assert.Nil(t, err)
	ps.bannedAddr[host] = &BannedInfo{
		Address:    host,
		BanUntil:   util.GetTime() + 60,
		CreateTime: util.GetTime(),
		Reason:     BanReasonManuallyAdded,
	}

	ret = s.handleAddPeerMsg(&ps, sp)
	assert.False(t, ret)

	ps.bannedAddr[host] = &BannedInfo{
		Address:    host,
		BanUntil:   util.GetTime() - 60,
		CreateTime: util.GetTime() - 120,
		Reason:     BanReasonManuallyAdded,
	}
	ret = s.handleAddPeerMsg(&ps, sp)
	assert.True(t, ret)
	assert.Equal(t, 0, len(ps.bannedAddr))
}

func TestParseListeners(t *testing.T) {
	tests := []struct {
		addr    string
		isValid bool
	}{
		{
			"",
			false,
		},
		{
			"127.0.0.1", // miss port
			false,
		},
		{
			":18833", // 0.0.0.0:port and [::]:port
			true,
		},
		{
			"127.0.0.1:18833", // ipv4
			true,
		},
		{
			"[fe80::c24:f21a:77ed:c4e5%en0]:18833", // ipv6
			true,
		},
		{
			"hello.world:18833", // invalid port
			false,
		},
	}
	for _, test := range tests {
		t.Logf("testing parseListeners:%s", test.addr)
		_, err := parseListeners([]string{test.addr})
		if test.isValid {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}

func TestAddrStringToNetAddr(t *testing.T) {
	tests := []struct {
		addr    string
		isValid bool
	}{
		{
			"",
			false,
		},
		{
			":18833",
			false,
		},
		{
			"127.0.0.1", // miss port
			false,
		},
		{
			"127.0.0.1:18833", // ipv4
			true,
		},
		{
			"127.0.0.1:abc", // invalid port
			false,
		},
	}
	for _, test := range tests {
		t.Logf("testing addrStringToNetAddr:%s", test.addr)
		_, err := addrStringToNetAddr(test.addr)
		if test.isValid {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}

func TestOnionAddr(t *testing.T) {
	onionAddrStr := "3g2upl4pq6kufc4m.onion:8333"

	conf.Cfg.P2PNet.NoOnion = false
	onionAddr, err := addrStringToNetAddr(onionAddrStr)
	assert.Nil(t, err)
	assert.Equal(t, onionAddrStr, onionAddr.String())
	assert.Equal(t, "onion", onionAddr.Network())

	conf.Cfg.P2PNet.NoOnion = true
	_, err = addrStringToNetAddr(onionAddrStr)
	assert.NotNil(t, err)
}

func TestInitListeners(t *testing.T) {
	configUpnp := conf.Cfg.P2PNet.Upnp
	configExternalIPs := conf.Cfg.P2PNet.ExternalIPs
	//configDefaultPort := model.ActiveNetParams.DefaultPort

	// restore
	defer func() {
		conf.Cfg.P2PNet.Upnp = configUpnp
		conf.Cfg.P2PNet.ExternalIPs = configExternalIPs
		//model.ActiveNetParams.DefaultPort = configDefaultPort
	}()

	var err error
	var listeners []net.Listener
	var nat upnp.NAT
	services := defaultServices
	listenAddrs := make([]string, 1)
	conf.Cfg.P2PNet.ExternalIPs = make([]string, 0)
	conf.Cfg.P2PNet.Upnp = false

	// external IP list is empty
	// address is invalid
	listenAddrs[0] = ""
	listeners, _, err = initListeners(s.addrManager, listenAddrs, services)
	assert.NotNil(t, err)
	for _, listener := range listeners {
		listener.Close()
	}

	// address is valid
	listenAddrs[0] = "127.0.0.1:18833"
	listeners, nat, err = initListeners(s.addrManager, listenAddrs, services)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listeners))
	assert.Nil(t, nat)

	// port is occupied
	listeners2, _, err := initListeners(s.addrManager, listenAddrs, services)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(listeners2))

	for _, listener := range listeners {
		listener.Close()
	}

	conf.Cfg.P2PNet.Upnp = true
	listeners, _, err = initListeners(s.addrManager, listenAddrs, services)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listeners))
	// not sure nat is discovered. take 3s
	for _, listener := range listeners {
		listener.Close()
	}

	externalIPs := make([]string, 0)
	externalIPs = append(externalIPs, "")
	externalIPs = append(externalIPs, "abc")
	externalIPs = append(externalIPs, ":123")
	externalIPs = append(externalIPs, "127.0.0.1")
	externalIPs = append(externalIPs, "127.0.0.1:abc")
	externalIPs = append(externalIPs, "127.0.0.1:18834")
	externalIPs = append(externalIPs, "seed.bitcoinabc.org:8333")
	conf.Cfg.P2PNet.ExternalIPs = externalIPs

	// external IP list is not empty
	listeners, _, err = initListeners(s.addrManager, listenAddrs, services)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listeners))
	for _, listener := range listeners {
		listener.Close()
	}

	// invalid port
	/*
		model.ActiveNetParams.DefaultPort = "abc"
		listeners, _, err = initListeners(s.addrManager, listenAddrs, services)
		assert.NotNil(t, err)
		for _, listener := range listeners {
			listener.Close()
		}*/
}

func getBlock(blockstr string) *block.Block {
	blk := block.NewBlock()
	blkBuf, _ := hex.DecodeString(blockstr)
	err := blk.Unserialize(bytes.NewReader(blkBuf))
	if err != nil {
		return nil
	}
	return blk
}

func TestServer_RelayUpdatedTipBlocks(t *testing.T) {
	blk1str := "0100000043497FD7F826957108F4A30FD9CEC3AEBA79972084E90EAD01EA330900000000" +
		"BAC8B0FA927C0AC8234287E33C5F74D38D354820E24756AD709D7038FC5F31F020E7494DFFFF00" +
		"1D03E4B67201010000000100000000000000000000000000000000000000000000000000000000" +
		"00000000FFFFFFFF0E0420E7494D017F062F503253482FFFFFFFFF0100F2052A01000000232102" +
		"1AEAF2F8638A129A3156FBE7E5EF635226B0BAFD495FF03AFE2C843D7E3A4B51AC00000000"
	blk2str := "0100000006128E87BE8B1B4DEA47A7247D5528D2702C96826C7A648497E773B800000000" +
		"E241352E3BEC0A95A6217E10C3ABB54ADFA05ABB12C126695595580FB92E222032E7494DFFFF00" +
		"1D00D2353401010000000100000000000000000000000000000000000000000000000000000000" +
		"00000000FFFFFFFF0E0432E7494D010E062F503253482FFFFFFFFF0100F2052A01000000232103" +
		"8A7F6EF1C8CA0C588AA53FA860128077C9E6C11E6830F4D7EE4E763A56B7718FAC00000000"
	blk1 := getBlock(blk1str)
	blk2 := getBlock(blk2str)

	blk1Idx := blockindex.NewBlockIndex(&blk1.Header)
	blk2Idx := blockindex.NewBlockIndex(&blk2.Header)
	blk2Idx.Prev = blk1Idx

	event := &chain.TipUpdatedEvent{
		TipIndex:          blk2Idx,
		ForkIndex:         blk1Idx,
		IsInitialDownload: true,
	}
	s.RelayUpdatedTipBlocks(event)

	event.IsInitialDownload = false
	s.RelayUpdatedTipBlocks(event)
}

func TestServer_UpdatePeerHeights(t *testing.T) {
	chn := make(chan struct{})
	svr, err := NewServer(model.ActiveNetParams, nil, chn)
	assert.Nil(t, err)

	peerCfg := &peer.Config{
		Listeners:         peer.MessageListeners{},
		UserAgentName:     "peer",
		UserAgentVersion:  "1.0",
		UserAgentComments: []string{"comment"},
		ChainParams:       &model.MainNetParams,
		Services:          wire.SFNodeBloom,
	}
	inPeer := peer.NewInboundPeer(peerCfg)

	hashStr := "3264bc2ac36a60840790ba1d475d01367e7c723da941069e9dc"
	blockHash, err := util.GetHashFromStr(hashStr)
	assert.Nil(t, err)

	r, w := io.Pipe()
	inConn := &conn{raddr: "127.0.0.1:18334", Writer: w, Reader: r}
	sp := newServerPeer(svr, false)
	sp.isWhitelisted = isWhitelisted(inConn.RemoteAddr())
	sp.Peer = peer.NewInboundPeer(newPeerConfig(sp))
	sp.AssociateConnection(inConn, svr.MsgChan, func(peer *peer.Peer) {
		svr.syncManager.NewPeer(peer)
	})

	svr.Start()
	defer svr.Stop()

	svr.AddPeer(sp)
	svr.UpdatePeerHeights(blockHash, 1, inPeer)
	assert.Nil(t, sp.LastAnnouncedBlock())

	// check update peer height logic
	ps := peerState{
		inboundPeers:    make(map[int32]*serverPeer),
		outboundPeers:   make(map[int32]*serverPeer),
		persistentPeers: make(map[int32]*serverPeer),
		bannedAddr:      make(map[string]*BannedInfo),
		bannedIPNet:     make(map[string]*BannedInfo),
		outboundGroups:  make(map[string]int),
	}
	ret := svr.handleAddPeerMsg(&ps, sp)
	assert.True(t, ret)

	updateMsg := updatePeerHeightsMsg{newHash: blockHash}

	// not update for origin peer
	updateMsg.newHeight = 2
	updateMsg.originPeer = sp.Peer
	sp.UpdateLastAnnouncedBlock(blockHash)
	sp.UpdateLastBlockHeight(1)
	svr.handleUpdatePeerHeights(&ps, updateMsg)
	assert.NotNil(t, sp.LastAnnouncedBlock())
	assert.Equal(t, int32(1), sp.LastBlock())

	// update for valid peer
	updateMsg.newHeight = 4
	updateMsg.originPeer = inPeer
	sp.UpdateLastAnnouncedBlock(blockHash)
	sp.UpdateLastBlockHeight(3)
	svr.handleUpdatePeerHeights(&ps, updateMsg)
	assert.Nil(t, sp.LastAnnouncedBlock())
	assert.Equal(t, int32(4), sp.LastBlock())

	// not update for no last announced block
	updateMsg.newHeight = 6
	updateMsg.originPeer = inPeer
	sp.UpdateLastAnnouncedBlock(nil)
	sp.UpdateLastBlockHeight(5)
	svr.handleUpdatePeerHeights(&ps, updateMsg)
	assert.Nil(t, sp.LastAnnouncedBlock())
	assert.Equal(t, int32(5), sp.LastBlock())

	sp.Disconnect()
	svr.handleDonePeerMsg(&ps, sp)
}

func TestServer_Stop(t *testing.T) {
	chn := make(chan struct{})
	svr, err := NewServer(model.ActiveNetParams, nil, chn)
	assert.Nil(t, err)

	go svr.Stop()

	select {
	case <-svr.quit:
	case <-time.After(3 * time.Second):
		t.Errorf("server stop timeout")
	}
}

func TestServer_ScheduleShutdown(t *testing.T) {
	chn := make(chan struct{})
	svr, err := NewServer(model.ActiveNetParams, nil, chn)
	assert.Nil(t, err)

	startTime := time.Now().Unix()
	endTime := startTime
	svr.ScheduleShutdown(2 * time.Second)

	select {
	case <-svr.quit:
		endTime = time.Now().Unix()
	case <-time.After(5 * time.Second):
		t.Error("ScheduleShutdown time out")
	}
	if endTime == startTime {
		t.Error("ScheduleShutdown too quick")
	}
}

func TestServer_disconnectPeer(t *testing.T) {
	var ret bool
	peerList := make(map[int32]*serverPeer)
	ret = disconnectPeer(peerList, nil, nil)
	assert.False(t, ret)

	r, w := io.Pipe()
	inConn := &conn{raddr: "127.0.0.1:18334", Writer: w, Reader: r}
	sp := newServerPeer(s, false)
	sp.isWhitelisted = isWhitelisted(inConn.RemoteAddr())
	sp.Peer = peer.NewInboundPeer(newPeerConfig(sp))
	sp.AssociateConnection(inConn, s.MsgChan, func(peer *peer.Peer) {
		s.syncManager.NewPeer(peer)
	})
	peerList[1] = sp

	compareFunc := func(*serverPeer) bool { return true }
	whenFound := func(*serverPeer) {}
	ret = disconnectPeer(peerList, compareFunc, whenFound)
	assert.True(t, ret)
	assert.Equal(t, 0, len(peerList))
	s.peerDoneHandler(sp)
}

func TestServer_addLocalAddress(t *testing.T) {
	tests := []struct {
		addr    string
		isValid bool
	}{
		{
			"",
			false,
		},
		{
			"127.0.0.1", // miss port
			false,
		},
		{
			"0.0.0.0:18833", // :port
			true,
		},
		{
			"127.0.0.1:18833", // ipv4
			true,
		},
		{
			"127.0.0.1:abc", // invalid port
			false,
		},
		{
			"XXXXXXXXXXXXXXXXYYYZZZ.onion:8333", // invalid address
			false,
		},
	}
	for _, test := range tests {
		t.Logf("testing addLocalAddress:%s", test.addr)
		err := addLocalAddress(s.addrManager, test.addr, wire.SFNodeNetwork)
		if test.isValid {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}

}
