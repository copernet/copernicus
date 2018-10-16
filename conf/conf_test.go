package conf

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/viper"
)

var confData = []byte(`
GoVersion: 1.9.2
Version: 1.0.0
BuildDate: 20180428
Service:
  Address: 10.0.0.0/8
HTTP:
  Host: 127.0.0.1
  Port: 8080
  Mode: test
RPC:
  Host: 127.0.0.1
  Port: 9552
Log:
  Level: error
  Format: json
`)

func initConfig() *configuration {
	config := &configuration{}
	viper.SetConfigType("yaml")

	filename := fmt.Sprintf("conf_test%04d.yml", rand.Intn(9999))
	err := ioutil.WriteFile(filename, confData, 0664)
	if err != nil {
		fmt.Printf("write config file failed:%s", err)
	}

	//parse struct tag
	c := configuration{}
	t := reflect.TypeOf(c)
	v := reflect.ValueOf(c)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if v.Field(i).Type().Kind() != reflect.Struct {
			key := field.Name
			value := field.Tag.Get(tagName)
			//set default value
			viper.SetDefault(key, value)
			//log.Printf("key is: %v,value is: %v\n", key, value)
		} else {
			structField := v.Field(i).Type()
			for j := 0; j < structField.NumField(); j++ {
				key := structField.Field(j).Name
				values := structField.Field(j).Tag.Get(tagName)
				viper.SetDefault(key, values)
			}
			continue
		}
	}

	// parse config
	file := must(os.Open(filename)).(*os.File)
	defer file.Close()
	defer os.Remove(filename)
	must(nil, viper.ReadConfig(file))
	must(nil, viper.Unmarshal(config))

	return config
}

type configuration struct {
	GoVersion string
	Version   string
	BuildDate string
	Service   struct {
		Address string
	}
	HTTP struct {
		Host string
		Port int
		Mode string
	}
	RPC struct {
		Host string
		Port int
	}
	Log struct {
		Level  string
		Format string
	}
}

func TestInitConfig(t *testing.T) {
	config := initConfig()
	expected := &configuration{}
	expected.Service.Address = "10.0.0.0/8"
	expected.HTTP.Host = "127.0.0.1"
	expected.HTTP.Port = 8080
	expected.HTTP.Mode = "test"
	expected.Log.Format = "json"
	expected.Log.Level = "error"
	expected.GoVersion = "1.9.2"
	expected.Version = "1.0.0"
	expected.BuildDate = "20180428"
	expected.RPC.Host = "127.0.0.1"
	expected.RPC.Port = 9552

	if !reflect.DeepEqual(config, expected) {
		t.Error("Expected value is not equal to the actual value obtained")
	}
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

func getDefaultConfiguration(dataDir string, testNet, regTestNet bool) *Configuration {
	defaultDataDir := AppDataDir(defaultDataDirname, false)

	return &Configuration{
		DataDir: dataDir,
		RPC: struct {
			RPCListeners         []string
			RPCUser              string
			RPCPass              string
			RPCLimitUser         string
			RPCLimitPass         string
			RPCCert              string `default:""`
			RPCKey               string
			RPCMaxClients        int
			RPCMaxWebsockets     int
			RPCMaxConcurrentReqs int
			RPCQuirks            bool
		}{
			RPCCert: filepath.Join(defaultDataDir, "rpc.cert"),
			RPCKey:  filepath.Join(defaultDataDir, "rpc.key"),
		},
		Mempool: struct {
			MinFeeRate           int64 //
			LimitAncestorCount   int   // Default for -limitancestorcount, max number of in-mempool ancestors
			LimitAncestorSize    int   // Default for -limitancestorsize, maximum kilobytes of tx + all in-mempool ancestors
			LimitDescendantCount int   // Default for -limitdescendantcount, max number of in-mempool descendants
			LimitDescendantSize  int   // Default for -limitdescendantsize, maximum kilobytes of in-mempool descendants
			MaxPoolSize          int64 `default:"300000000"` // Default for MaxPoolSize, maximum megabytes of mempool memory usage
			MaxPoolExpiry        int   // Default for -mempoolexpiry, expiration time for mempool transactions in hours
		}{
			MaxPoolSize: 300000000,
		},
		P2PNet: struct {
			ListenAddrs         []string `validate:"require" default:"1234"`
			MaxPeers            int      `default:"128"`
			TargetOutbound      int      `default:"8"`
			ConnectPeersOnStart []string
			DisableBanning      bool `default:"true"`
			BanThreshold        uint32
			TestNet             bool
			RegTest             bool `default:"false"`
			SimNet              bool
			DisableListen       bool          `default:"true"`
			BlocksOnly          bool          `default:"false"` //Do not accept transactions from remote peers.
			BanDuration         time.Duration // How long to ban misbehaving peers
			Proxy               string        // Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)
			UserAgentComments   []string      // Comment to add to the user agent -- See BIP 14 for more information.
			DisableDNSSeed      bool          //Disable DNS seeding for peers
			DisableRPC          bool          `default:"false"`
			DisableTLS          bool          `default:"false"`
			Whitelists          []*net.IPNet
			NoOnion             bool     `default:"true"`  // Disable connecting to tor hidden services
			Upnp                bool     `default:"false"` // Use UPnP to map our listening port outside of NAT
			ExternalIPs         []string // Add an ip to the list of local addresses we claim to listen on to peers
			//AddCheckpoints      []model.Checkpoint
		}{
			ListenAddrs:    []string{"1234"},
			MaxPeers:       128,
			TargetOutbound: 8,
			DisableBanning: true,
			DisableListen:  true,
			BlocksOnly:     false,
			DisableRPC:     false,
			Upnp:           false,
			DisableTLS:     false,
			NoOnion:        true,
			TestNet:        testNet,
			RegTest:        regTestNet,
		},
		Protocol: struct {
			NoPeerBloomFilters bool `default:"true"`
			DisableCheckpoints bool `default:"true"`
		}{NoPeerBloomFilters: true, DisableCheckpoints: true},
		Script: struct {
			AcceptDataCarrier   bool `default:"true"`
			MaxDatacarrierBytes uint `default:"223"`
			IsBareMultiSigStd   bool `default:"true"`
			//use promiscuousMempoolFlags to make more or less check of script, the type of value is uint
			PromiscuousMempoolFlags string `default:"0"`
			Par                     int    `default:"32"`
		}{
			AcceptDataCarrier:       true,
			MaxDatacarrierBytes:     223,
			IsBareMultiSigStd:       true,
			PromiscuousMempoolFlags: "0",
			Par: 32,
		},
		TxOut: struct {
			DustRelayFee int64 `default:"83"`
		}{DustRelayFee: 83},
		Chain: struct {
			AssumeValid         string
			UtxoHashStartHeight int32 `default:"-1"`
			UtxoHashEndHeight   int32 `default:"-1"`
		}{
			AssumeValid:         "",
			UtxoHashStartHeight: -1,
			UtxoHashEndHeight:   -1,
		},
		Mining: struct {
			BlockMinTxFee int64  // default DefaultBlockMinTxFee
			BlockMaxSize  uint64 // default DefaultMaxGeneratedBlockSize
			BlockVersion  int32  `default:"-1"`
			Strategy      string `default:"ancestorfeerate"` // option:ancestorfee/ancestorfeerate
		}{
			BlockVersion: -1,
			Strategy:     "ancestorfeerate",
		},
		PProf: struct {
			IP   string `default:"localhost"`
			Port string `default:"6060"`
		}{IP: "localhost", Port: "6060"},
		AddrMgr: struct {
			SimNet       bool
			ConnectPeers []string
		}{SimNet: false},
		BlockIndex: struct{ CheckBlockIndex bool }{CheckBlockIndex: regTestNet},
	}
}

func createTmpFile() {
	confFile := os.Getenv("GOPATH") + "/src/" + defaultProjectDir + "/conf/"
	CopyFile(confFile+"conf.yml", confFile+"conf.yml.tmp")
	os.Remove(confFile + "conf.yml")
	f, err := os.Create(confFile + "conf.yml")
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
}

func revert() {
	confFile := os.Getenv("GOPATH") + "/src/" + defaultProjectDir + "/conf/"
	os.Remove(confFile + "conf.yml")
	CopyFile(confFile+"conf.yml.tmp", confFile+"conf.yml")
	os.Remove(confFile + "conf.yml.tmp")
}

func TestInitConfig2(t *testing.T) {
	tests := []struct {
		in   []string
		want *Configuration
	}{
		{[]string{"--datadir=/tmp/Coper"}, getDefaultConfiguration("/tmp/Coper", false, false)},
		{[]string{"--datadir=/tmp/Coper", "--testnet"}, getDefaultConfiguration("/tmp/Coper/testnet", true, false)},
		{[]string{"--datadir=/tmp/Coper", "--regtest"}, getDefaultConfiguration("/tmp/Coper/regtest", false, true)},
	}
	createTmpFile()
	defer os.RemoveAll("/tmp/Coper")
	defer revert()

	for i, v := range tests {
		value := v
		result := InitConfig(value.in)
		fmt.Println(*result)
		fmt.Println(*value.want)
		if !reflect.DeepEqual(result, value.want) {
			t.Errorf(" %d it not expect", i)
		}
	}
}

func TestSetUnitTestDataDir(t *testing.T) {
	args := []string{"--testnet"}
	Cfg = InitConfig(args)
	testDir, err := SetUnitTestDataDir(Cfg)
	if err != nil {
		fmt.Printf("Error: %s", err)
		os.Exit(1)
	}
	t.Logf("generated file path is: %v", testDir)
	defer os.RemoveAll(testDir)
	_, err = os.Stat(testDir)
	if err != nil && os.IsNotExist(err) {
		t.Errorf("SetUnitTestDataDir implementation error:%v", err)
	}
}
