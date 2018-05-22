package conf

import (
	"log"
	"os"
	"path"
	"reflect"
	"runtime"

	"github.com/spf13/viper"
	"time"
	"net"
)

const (
	tagName = "default"

	defaultConfigFilename       = "cps.conf"
	defaultDataDirname          = "data"
	defaultLogLevel             = "info"
	defaultLogDirname           = "logs"
	defaultLogFilename          = "btcd.log"
	defaultMaxPeers             = 125
	defaultBanDuration          = time.Hour * 24
	defaultBanThreshold         = 100
	defaultConnectTimeout       = time.Second * 30
	defaultMaxRPCClients        = 10
	defaultMaxRPCWebsockets     = 25
	defaultMaxRPCConcurrentReqs = 20
	defaultDbType               = "ffldb"
	defaultFreeTxRelayLimit     = 15.0
	defaultBlockMinSize         = 0
	defaultBlockMaxSize         = 750000
	defaultBlockMinWeight       = 0
	defaultBlockMaxWeight       = 3000000
	blockMaxSizeMin             = 1000
	blockMaxWeightMin           = 4000
	// blockMaxSizeMax              = blockchain.MaxBlockBaseSize - 1000
	// blockMaxWeightMax            = blockchain.MaxBlockWeight - 4000
	defaultGenerate              = false
	defaultMaxOrphanTransactions = 100
	defaultMaxOrphanTxSize       = 100000
	defaultSigCacheMaxSize       = 100000
	sampleConfigFilename         = "sample-btcd.conf"
	defaultTxIndex               = false
	defaultAddrIndex             = false
)

var Cfg *Configuration

// init configuration
func initConfig() *Configuration {
	config := &Configuration{}
	viper.SetEnvPrefix("copernicus")
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")

	// find out where the sample config lives
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("get current file path failed.")
	}
	filePath := path.Join(path.Dir(filename), "./conf.yml")
	viper.SetDefault("conf", filePath)

	//parse struct tag
	c := Configuration{}
	t := reflect.TypeOf(c)
	v := reflect.ValueOf(c)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if v.Field(i).Type().Kind() != reflect.Struct {
			key := field.Name
			value := field.Tag.Get(tagName)
			//set default value
			viper.SetDefault(key, value)
			log.Printf("key is: %v,value is: %v\n", key, value)
		} else {
			structField := v.Field(i).Type()
			for j := 0; j < structField.NumField(); j++ {
				key := structField.Field(j).Name
				values := structField.Field(j).Tag.Get(tagName)
				viper.SetDefault(key, values)
				log.Printf("key is: %v,value is: %v\n", key, values)
			}
			continue
		}
	}

	// get config file path from environment
	conf := viper.GetString("conf")

	// parse config
	file := must(os.Open(conf)).(*os.File)
	defer file.Close()
	must(nil, viper.ReadConfig(file))
	must(nil, viper.Unmarshal(config))

	return config
}

// Configuration defines all configurations for application
type Configuration struct {
	GoVersion string `validate:"require"` //description:"Display version information and exit"
	Version   string `validate:"require"` //description:"Display version information of copernicus"
	BuildDate string `validate:"require"` //description:"Display build date of copernicus"
	DataDir   string `default:"data"`

	Service struct {
		Address string `default:"1.0.0.1:80"`
	}
	HTTP struct {
		Host string `validate:"require"`
		Port int
		Mode string
	}
	RPC struct {
		RPCListeners         []string            // Add an interface/port to listen for RPC connections (default port: 8334, testnet: 18334)
		RPCUser              string              // Username for RPC connections
		RPCPass              string              // Password for RPC connections
		RPCLimitUser         string              //Username for limited RPC connections
		RPCLimitPass         string              //Password for limited RPC connections
		RPCCert              string `default:""` //File containing the certificate file
		RPCKey               string              //File containing the certificate key
		RPCMaxClients        int                 //Max number of RPC clients for standard connections
		RPCMaxWebsockets     int                 //Max number of RPC websocket connections
		RPCMaxConcurrentReqs int                 //Max number of concurrent RPC requests that may be processed concurrently
		RPCQuirks            bool                //Mirror some JSON-RPC quirks of Bitcoin Core -- NOTE: Discouraged unless interoperability issues need to be worked around
	}
	Log struct {
		Level  string //description:"Define level of log,include trace, debug, info, warn, error"
		Format string
	}
	Mempool struct {
		MinFeeRate int64
	}
	P2PNet struct {
		ListenAddrs         []string `validate:"require" default:"1234"`
		MaxPeers            int      `default:"128"`
		TargetOutbound      int      `default:"8"`
		ConnectPeersOnStart []string
		DisableBanning      bool     `default:"true"`
		BanThreshold        uint32
		SimNet              bool     `default:"false"`
		DisableListen       bool     `default:"true"`
		BlocksOnly          bool     `default:"true"`
		BanDuration         time.Duration // How long to ban misbehaving peers
		Proxy               string        // Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)
		UserAgentComments   []string      // Comment to add to the user agent -- See BIP 14 for more information.
		DisableDNSSeed      bool          //Disable DNS seeding for peers
		DisableRPC          bool     `default:"true"`
		DisableTLS          bool
		Whitelists          []*net.IPNet
		NoOnion             bool     `default:"true"` // Disable connecting to tor hidden services
		Upnp                bool                      // Use UPnP to map our listening port outside of NAT
		ExternalIPs         []string                  // Add an ip to the list of local addresses we claim to listen on to peers
	}
	AddrMgr struct {
		SimNet       bool
		ConnectPeers []string
	}
	Protocal struct {
		NoPeerBloomFilters bool `default:"false"`
	}
	Script struct {
		AcceptDataCarrier bool `default:"true"`
		MaxDatacarrierBytes uint `default:"83"`
		IsBareMultiSigStd bool `default:"true"`
	}
	TxOut struct {
		DustRelayFee int64 `default:"83"`
	}
}

func must(i interface{}, err error) interface{} {
	if err != nil {
		panic(err)
	}
	return i
}

func init() {
	Cfg = initConfig()
}

// Validate validates configuration
//func (c Configuration) Validate() error {
//	validate := validator.New(&validator.Config{TagName: "validate"})
//	return validate.Struct(c)
//}
