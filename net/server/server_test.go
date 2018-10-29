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
	"testing"
	"time"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/block"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/net/connmgr"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/peer"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
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
		Daily    bool   `json:"daily"`
	}{
		FileName: logDir + "/" + conf.Cfg.Log.FileName + ".log",
		Level:    log.GetLevel(conf.Cfg.Log.Level),
		Daily:    false,
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

func makeTestServer() (*Server, string, chan struct{}, error) {
	dir, err := ioutil.TempDir("", "server")
	if err != nil {
		return nil, "", nil, err
	}
	appInitMain([]string{"--datadir", dir, "--regtest"})
	c := make(chan struct{})
	conf.Cfg.P2PNet.ListenAddrs = []string{"127.0.0.1:0"}
	s, err := NewServer(model.ActiveNetParams, c)
	if err != nil {
		return nil, "", nil, err
	}
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
	msg := wire.NewMsgPing(10)
	s.BroadcastMessage(msg)
}

func TestConnectedCount(t *testing.T) {
	if c := s.ConnectedCount(); c != 0 {
		t.Errorf("ConnectedCount should be 0")
	}
}

func TestOutboundGroupCount(t *testing.T) {
	if c := s.OutboundGroupCount(""); c != 0 {
		t.Errorf("OutboundGroupCount should be 0")
	}
}

func TestRelayInventory(t *testing.T) {

	iv := wire.NewInvVect(wire.InvTypeTx, &util.HashZero)
	s.RelayInventory(iv, 1)
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
	addr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18555,
	}
	if isWhitelisted(addr) {
		t.Errorf("shoule not in whitelist")
	}
	_, ipnet, _ := net.ParseCIDR("127.0.0.1/8")
	conf.Cfg.P2PNet.Whitelists = []*net.IPNet{ipnet}
	if !isWhitelisted(addr) {
		t.Errorf("shoule  in whitelist")
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
	in := peer.NewInboundPeer(&config)
	msg := wire.NewMsgVersion(me, you, nonce, lastBlock)

	sp := newServerPeer(s, false)
	sp.Peer = in
	sp.OnVersion(in, msg)
}

func TestOnMemPool(t *testing.T) {
	msg := wire.NewMsgMemPool()
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	sp.OnMemPool(in, msg)
}

func TestOnTx(t *testing.T) {
	msgTx := (*wire.MsgTx)(tx.NewTx(0, 1))
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	done := make(chan struct{})
	sp.OnTx(in, msgTx, done)
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
	msgInv := wire.NewMsgInv()
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	sp.OnInv(in, msgInv)
}

func TestOnHeaders(t *testing.T) {
	msgHeaders := wire.NewMsgHeaders()
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp := newServerPeer(s, false)
	sp.Peer = in
	sp.OnHeaders(in, msgHeaders)
}

/*
func TestOnGetData(t *testing.T) {
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
	sp.OnGetData(in, m1)
	sp.OnGetData(in, m2)
}
*/

func TestTransferMsgToBusinessPro(t *testing.T) {
	go func() {
		http.ListenAndServe(":6060", nil)
	}()
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
		for i := 0; i < 3; i++ {
			select {
			case <-done:
			}
		}
	}()
	for _, msg := range msgs {
		sp.TransferMsgToBusinessPro(msg, done)
	}
}
