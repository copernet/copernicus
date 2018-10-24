package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lblockindex"
	"github.com/copernet/copernicus/logic/lchain"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/peer"
	"github.com/copernet/copernicus/persist"
	"github.com/copernet/copernicus/persist/blkdb"
	"github.com/copernet/copernicus/persist/db"
	"github.com/copernet/copernicus/util"
)

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
	utxoConfig := utxo.UtxoConfig{Do: &db.DBOption{FilePath: conf.Cfg.DataDir + "/chainstate", CacheSize: (1 << 20) * 8}}
	utxo.InitUtxoLruTip(&utxoConfig)

	chain.InitGlobalChain()

	// Init blocktree DB
	blkdbCfg := blkdb.BlockTreeDBConfig{Do: &db.DBOption{FilePath: conf.Cfg.DataDir + "/blocks/index", CacheSize: (1 << 20) * 8}}
	blkdb.InitBlockTreeDB(&blkdbCfg)

	persist.InitPersistGlobal()

	// Load blockindex DB
	lblockindex.LoadBlockIndexDB()
	lchain.InitGenesisChain()

	mempool.InitMempool()
	crypto.InitSecp256()
	ltx.ScriptVerifyInit()
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

func TestNewServer(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	s.Stop()
	s.WaitForShutdown()
}

func TestCountBytes(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.AddBytesReceived(10)
	s.AddBytesSent(20)
	if r, s := s.NetTotals(); r != 10 || s != 20 {
		t.Fatalf("server received and sent bytes failed")
	}
}

func TestAddPeer(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	sp := newServerPeer(s, false)
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp.Peer = in

	s.Start()
	s.AddPeer(sp)
	s.Stop()
	s.WaitForShutdown()
}

func TestBanPeer(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	sp := newServerPeer(s, false)
	config := peer.Config{}
	in := peer.NewInboundPeer(&config)
	sp.Peer = in

	s.Start()
	s.BanPeer(sp)
	s.Stop()
	s.WaitForShutdown()
}

func TestAddRebroadcastInventory(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	s.wg.Add(1)
	go s.rebroadcastHandler()
	iv := wire.NewInvVect(wire.InvTypeTx, &util.HashZero)
	s.AddRebroadcastInventory(iv, 10)
	s.Stop()
	s.WaitForShutdown()
}

func TestRemoveRebroadcastInventory(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	s.wg.Add(1)
	go s.rebroadcastHandler()
	iv := wire.NewInvVect(wire.InvTypeTx, &util.HashZero)
	s.RemoveRebroadcastInventory(iv)
	s.Stop()
	s.WaitForShutdown()
}

func makeTx(t *testing.T) (*tx.Tx, error) {
	tx := tx.Tx{}
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
	if err := tx.Unserialize(bytes.NewReader(originBytes)); err != nil {
		return nil, err
	}
	return &tx, nil
}

func TestAnnounceNewTransactions(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	tx, err := makeTx(t)
	if err != nil {
		t.Fatalf("makeTx() failed: %v\n", err)
	}
	s.AnnounceNewTransactions([]*mempool.TxEntry{&mempool.TxEntry{Tx: tx}})
	s.Stop()
	s.WaitForShutdown()
}

func TestBroadcastMessage(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	msg := wire.NewMsgPing(10)
	s.BroadcastMessage(msg)
	s.Stop()
	s.WaitForShutdown()
}

func TestConnectedCount(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	if c := s.ConnectedCount(); c != 0 {
		t.Errorf("ConnectedCount should be 0")
	}
	s.Stop()
	s.WaitForShutdown()
}

func TestOutboundGroupCount(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	if c := s.OutboundGroupCount(""); c != 0 {
		t.Errorf("OutboundGroupCount should be 0")
	}
	s.Stop()
	s.WaitForShutdown()
}

func TestRelayInventory(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	iv := wire.NewInvVect(wire.InvTypeTx, &util.HashZero)
	s.RelayInventory(iv, 1)
	s.Stop()
	s.WaitForShutdown()
}

func TestTransactionConfirmed(t *testing.T) {
	s, dir, _, err := makeTestServer()
	if err != nil {
		t.Fatalf("makeTestServer() failed: %v\n", err)
	}
	defer os.RemoveAll(dir)
	s.Start()
	s.wg.Add(1)
	go s.rebroadcastHandler()

	tx, err := makeTx(t)
	if err != nil {
		t.Fatalf("makeTx() failed: %v\n", err)
	}
	s.TransactionConfirmed(tx)
	s.Stop()
	s.WaitForShutdown()
}
