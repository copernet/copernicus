package conf

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/go-playground/validator.v8"
)

const (
	tagName = "default"

	defaultConfigFilename       = "conf.yml"
	defaultDataDirname          = "coper"
	defaultProjectDir           = "github.com/copernet/copernicus"
	defaultLogLevel             = "info"
	defaultLogDirname           = "logs"
	defaultLogFilename          = "coper.log"
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
	sampleConfigFilename         = "sample-coper.conf"
	defaultTxIndex               = false
	defaultAddrIndex             = false
	defaultDescendantLimit       = 25
	defaultDescendantSizeLimit   = 101
	defaultAncestorSizeLimit     = 101
	defaultAncestorLimit         = 25
	defaultMempoolExpiry         = 336
	defaultMaxMempoolSize        = 300
)

var (
	Cfg     *Configuration
	DataDir string
)

// InitConfig init configuration
func InitConfig(args []string) *Configuration {
	// parse command line parameter to set program datadir
	defaultDataDir := AppDataDir(defaultDataDirname, false)
	DataDir = defaultDataDir
	opts := InitArgs(args)
	if len(opts.DataDir) > 0 {
		DataDir = opts.DataDir
	}

	discover := opts.Discover
	fmt.Printf("discover:%d/n", discover)

	if !ExistDataDir(DataDir) {
		err := os.MkdirAll(DataDir, os.ModePerm)
		if err != nil {
			panic("datadir create failed: " + err.Error())
		}

		// get GOPATH environment and copy conf file to dst dir
		gopath := os.Getenv("GOPATH")
		if gopath != "" {
			// first try
			projectPath := gopath + "/src/" + defaultProjectDir
			filePath := projectPath + "/conf/" + defaultConfigFilename
			_, err = os.Stat(filePath)
			if !os.IsNotExist(err) {
				CopyFile(filePath, DataDir+"/"+defaultConfigFilename)
			} else {
				// second try
				projectPath = gopath + "/src/copernicus"
				filePath = projectPath + "/conf/" + defaultConfigFilename
				CopyFile(filePath, DataDir+"/"+defaultConfigFilename)
			}
		}
	}

	config := &Configuration{}
	viper.SetConfigType("yaml")

	//parse struct tag
	c := Configuration{}
	t := reflect.TypeOf(c)
	v := reflect.ValueOf(c)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if v.Field(i).Type().Kind() != reflect.Struct {
			key := field.Name
			value, ok := field.Tag.Lookup(tagName)
			if !ok {
				continue
			}
			//set default value
			viper.SetDefault(key, value)
			//log.Printf("key is: %v,value is: %v\n", key, value)
		} else {
			structField := v.Field(i).Type()
			structName := t.Field(i).Name
			for j := 0; j < structField.NumField(); j++ {
				fieldName := structField.Field(j).Name
				key := fmt.Sprintf("%s.%s", structName, fieldName)
				values, ok := structField.Field(j).Tag.Lookup(tagName)
				if !ok {
					continue
				}
				viper.SetDefault(key, values)
				//log.Printf("key is: %v,value is: %v\n", key, values)
			}
			continue
		}
	}

	// parse config
	file := must(os.Open(DataDir + "/conf.yml")).(*os.File)
	defer file.Close()
	must(nil, viper.ReadConfig(file))
	must(nil, viper.Unmarshal(config))

	// set data dir
	config.DataDir = DataDir

	config.RPC.RPCKey = filepath.Join(defaultDataDir, "rpc.key")
	config.RPC.RPCCert = filepath.Join(defaultDataDir, "rpc.cert")
	return config
}

// Configuration defines all configurations for application
type Configuration struct {
	GoVersion string `validate:"require"` //description:"Display version information and exit"
	Version   string `validate:"require"` //description:"Display version information of copernicus"
	BuildDate string `validate:"require"` //description:"Display build date of copernicus"
	DataDir   string `default:"data"`

	// Service struct {
	// 	Address string `default:"1.0.0.1:80"`
	// }
	RPC struct {
		RPCListeners         []string // Add an interface/port to listen for RPC connections (default port: 8334, testnet: 18334)
		RPCUser              string   // Username for RPC connections
		RPCPass              string   // Password for RPC connections
		RPCLimitUser         string   //Username for limited RPC connections
		RPCLimitPass         string   //Password for limited RPC connections
		RPCCert              string   `default:""` //File containing the certificate file
		RPCKey               string   //File containing the certificate key
		RPCMaxClients        int      //Max number of RPC clients for standard connections
		RPCMaxWebsockets     int      //Max number of RPC websocket connections
		RPCMaxConcurrentReqs int      //Max number of concurrent RPC requests that may be processed concurrently
		RPCQuirks            bool     //Mirror some JSON-RPC quirks of Bitcoin Core -- NOTE: Discouraged unless interoperability issues need to be worked around
	}
	Log struct {
		Level    string   //description:"Define level of log,include trace, debug, info, warn, error"
		Module   []string // only output the specified module's log when using log.Print(...)
		FileName string   // the name of log file
	}
	Mempool struct {
		MinFeeRate           int64 //
		LimitAncestorCount   int   // Default for -limitancestorcount, max number of in-mempool ancestors
		LimitAncestorSize    int   // Default for -limitancestorsize, maximum kilobytes of tx + all in-mempool ancestors
		LimitDescendantCount int   // Default for -limitdescendantcount, max number of in-mempool descendants
		LimitDescendantSize  int   // Default for -limitdescendantsize, maximum kilobytes of in-mempool descendants
		MaxPoolSize          int64 `default:"300000000"` // Default for MaxPoolSize, maximum megabytes of mempool memory usage
		MaxPoolExpiry        int   // Default for -mempoolexpiry, expiration time for mempool transactions in hours
	}
	P2PNet struct {
		ListenAddrs         []string `validate:"require" default:"1234"`
		MaxPeers            int      `default:"128"`
		TargetOutbound      int      `default:"8"`
		ConnectPeersOnStart []string
		DisableBanning      bool `default:"true"`
		BanThreshold        uint32
		SimNet              bool          `default:"false"`
		DisableListen       bool          `default:"true"`
		BlocksOnly          bool          `default:"true"` //Do not accept transactions from remote peers.
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
	}
	AddrMgr struct {
		SimNet       bool
		ConnectPeers []string
	}
	Protocal struct {
		NoPeerBloomFilters bool `default:"true"`
		DisableCheckpoints bool `default:"true"`
	}
	Script struct {
		AcceptDataCarrier   bool `default:"true"`
		MaxDatacarrierBytes uint `default:"223"`
		IsBareMultiSigStd   bool `default:"true"`
		//use promiscuousMempoolFlags to make more or less check of script, the type of value is uint
		PromiscuousMempoolFlags string
	}
	TxOut struct {
		DustRelayFee int64 `default:"83"`
	}
	Chain struct {
		AssumeValid    string
		StartLogHeight int32 `default:"2147483647"`
	}
	Mining struct {
		BlockMinTxFee int64  // default DefaultBlockMinTxFee
		BlockMaxSize  uint64 // default DefaultMaxGeneratedBlockSize
		BlockVersion  int32  `default:"-1"`
		Strategy      string `default:"ancestorfeerate"` // option:ancestorfee/ancestorfeerate
	}
	PProf struct {
		IP   string `default:"localhost"`
		Port string `default:"6060"`
	}
}

func must(i interface{}, err error) interface{} {
	if err != nil {
		panic(err)
	}
	return i
}

func CopyFile(src, des string) (w int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	desFile, err := os.Create(des)
	if err != nil {
		return 0, err
	}
	defer desFile.Close()

	return io.Copy(desFile, srcFile)
}

// Validate validates configuration
func (c Configuration) Validate() error {
	//validate := validator.New(&validator.Config{TagName: "validate"})
	validate := validator.New(&validator.Config{TagName: "validate"})
	return validate.Struct(c)
}

func ExistDataDir(datadir string) bool {
	_, err := os.Stat(datadir)
	if err == nil {
		return true
	}
	if os.IsExist(err) {
		return false
	}

	return false
}
