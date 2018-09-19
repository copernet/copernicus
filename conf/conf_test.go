package conf

import (
	"os"
	"fmt"
	"testing"
	"io/ioutil"
	"math/rand"

	"github.com/spf13/viper"
	. "github.com/smartystreets/goconvey/convey"
)

var confData = []byte(`
GoVersion: 1.9.2
Version: 1.0.0
BuildDate: 20180428
RPC:
  RPCListeners: [127.0.0.1:8334, 127.0.0.1:18334]
  RPCUser: copernicus
  RPCPass: doXT3DXgAQCNU0Li0pujQ6zR3Y
  RPCMaxClients: 1000
Log:
  FileName: copernicus
  Level: debug
  Module: [mempool,utxo,bench,service]
Mining:
  BlockMinTxFee: 100
  BlockMaxSize: 2000000
  BlockVersion: 1
  Strategy: ancestorfeerate
Chain:
  AssumeValid:
P2PNet:
  ListenAddrs: ["127.0.0.1:8333","127.0.0.1:18333"]
  MaxPeers: 5
  TargetOutbound: 3
  ConnectPeersOnStart:
  DisableBanning: true
  SimNet: false
  DisableListen: false
  BlocksOnly: false
  DisableDNSSeed: false
  DisableRPC: false
  Upnp: false
  DisableTLS: false
Protocal:
  NoPeerBloomFilters: true
  DisableCheckpoints: true
AddrMgr:
  SimNet: false
  ConnectPeers:
Script:
  AcceptDataCarrier:
  MaxDatacarrierBytes:
  IsBareMultiSigStd:
  PromiscuousMempoolFlags:
TxOut:
  DustRelayFee:
`)

func TestInitConfig(t *testing.T) {
	Convey("Given config file", t, func() {
		filename := fmt.Sprintf("conf_test%04d.yml", rand.Intn(9999))
		err := ioutil.WriteFile(filename, confData, 0664)
		if err != nil {
			t.Error("write config file failed", err)
		}

		Convey("When init configuration", func() {
			config := initConfig()
			defaultDataDir := AppDataDir(defaultDataDirname, false)

			Convey("Configuration should resemble default configuration", func() {
				expected := &Configuration{}
				expected.GoVersion = "1.9.2"
				expected.Version = "1.0.0"
				expected.BuildDate = "20180428"
				expected.DataDir = defaultDataDir

				//rpc
				str := make([]string, 0)
				listeners := append(str, "127.0.0.1:8334")
				listeners = append(listeners, "127.0.0.1:18334")
				expected.RPC.RPCListeners = listeners
				expected.RPC.RPCUser = "copernicus"
				expected.RPC.RPCPass = "doXT3DXgAQCNU0Li0pujQ6zR3Y"
				expected.RPC.RPCCert = defaultDataDir + "/rpc.cert"
				expected.RPC.RPCKey = defaultDataDir + "/rpc.key"
				expected.RPC.RPCMaxClients = 1000

				//mining
				expected.Mining.BlockMaxSize = 2000000
				expected.Mining.BlockMinTxFee = 100
				expected.Mining.BlockVersion = 1
				expected.Mining.Strategy = "ancestorfeerate"

				//log
				log := make([]string, 0)
				logList := append(log, "mempool")
				logList = append(logList, "utxo")
				logList = append(logList, "bench")
				logList = append(logList, "service")
				expected.Log.Module = logList
				expected.Log.FileName = "copernicus"
				expected.Log.Level = "debug"

				//net
				net := make([]string, 0)
				netList := append(net, "127.0.0.1:8333")
				netList = append(netList, "127.0.0.1:18333")
				expected.P2PNet.ListenAddrs = netList
				expected.P2PNet.MaxPeers = 5
				expected.P2PNet.TargetOutbound = 3
				expected.P2PNet.DisableBanning = true

				expected.Protocal.NoPeerBloomFilters = true
				expected.Protocal.DisableCheckpoints = true

				So(config, ShouldResemble, expected)
			})
		})

		Reset(func() {
			os.Remove(filename)
		})
	})
}

func TestSetDefault(t *testing.T) {
	viper.SetDefault("key", 100)
	if viper.GetInt("key") != 100 {
		t.Error("set default(key) error")
	}

	viper.SetDefault("rpc.user", "admin")
	if viper.GetString("rpc.user") != "admin" {
		t.Error("set default(rpc.user) error")
	}
	viper.SetDefault("Log.Level", "debug")

	if viper.GetString("Log.Level") != "debug" {
		t.Error("set default(Log.Level) error")
	}
}

func TestCopyFile(t *testing.T) {
	nameSRC := "conf.txt"
	nameDES := "copy_conf.txt"
	content := "hello,copernicus"
	data := []byte(content)
	err := ioutil.WriteFile(nameSRC, data, 0644)
	if err != nil {
		t.Errorf("write conf file failed: %s\n ", err)
	}
	defer os.Remove(nameSRC)

	writeNum, err := CopyFile(nameSRC, nameDES)
	if err != nil {
		t.Errorf("copy file failed: %s\n", err)
	}

	readNum, err := ioutil.ReadFile(nameDES)
	if int64(len(readNum)) != writeNum {
		t.Errorf("error copying the contents of the file: %s\n", err)
	}
	defer os.Remove(nameDES)
}

func TestExistDataDir(t *testing.T) {
	fileTrue := "conf.txt"
	fileFalse := "confNo.txt"

	fileTrue, err := ioutil.TempDir("", fileTrue)
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.Remove(fileTrue)

	if !ExistDataDir(fileTrue) {
		t.Errorf("the fileTrue file should exist!")
	}

	if ExistDataDir(fileFalse) {
		t.Errorf("the fileFalse file shouldn't exist!")
	}
}
