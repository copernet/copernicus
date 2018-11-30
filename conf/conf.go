package conf

import (
	"errors"
	"fmt"
	"gopkg.in/go-playground/validator.v8"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	AppMajor uint = 0
	AppMinor uint = 0
	AppPatch uint = 6

	// AppPreRelease MUST only contain characters from semanticAlphabet
	// per the semantic versioning spec.
	AppPreRelease = "beta"
	AppName       = "Copernicus"
)

const (
	tagName = "default"

	defaultConfigFilename = "bitcoincash.yml"
	defaultDataDirname    = "bitcoincash"
	defaultProjectDir     = "github.com/copernet/copernicus"

	OneMegaByte = 1000000
)

// Configuration defines all configurations for application
type Configuration struct {
	GoVersion          string `validate:"require"` //description:"Display version information and exit"
	Version            string `validate:"require"` //description:"Display version information of copernicus"
	BuildDate          string `validate:"require"` //description:"Display build date of copernicus"
	DataDir            string `default:"data"`
	Reindex            bool
	Excessiveblocksize uint64

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
		MinFeeRate           int64  //
		LimitAncestorCount   int    // Default for -limitancestorcount, max number of in-mempool ancestors
		LimitAncestorSize    int    // Default for -limitancestorsize, maximum kilobytes of tx + all in-mempool ancestors
		LimitDescendantCount int    // Default for -limitdescendantcount, max number of in-mempool descendants
		LimitDescendantSize  int    // Default for -limitdescendantsize, maximum kilobytes of in-mempool descendants
		MaxPoolSize          int64  `default:"300000000"` // Default for MaxPoolSize, maximum megabytes of mempool memory usage
		MaxPoolExpiry        int    `default:"336"`       // Default for -mempoolexpiry, expiration time for mempool transactions in hours
		CheckFrequency       uint64 `default:"4294967296"`
	}
	P2PNet struct {
		ListenAddrs         []string `validate:"require" default:"1234"`
		MaxPeers            int      `default:"128"`
		TargetOutbound      int      `default:"64"`
		ConnectPeersOnStart []string
		DisableBanning      bool   `default:"true"`
		BanThreshold        uint32 `default:"100"`
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
		MaxTimeAdjustment   uint64   `default:"4200"`
		//AddCheckpoints      []model.Checkpoint
	}
	AddrMgr struct {
		SimNet       bool
		ConnectPeers []string
	}
	Protocol struct {
		NoPeerBloomFilters bool `default:"true"`
		DisableCheckpoints bool `default:"true"`
	}
	Script struct {
		AcceptDataCarrier   bool `default:"true"`
		MaxDatacarrierBytes uint `default:"223"`
		IsBareMultiSigStd   bool `default:"true"`
		//use promiscuousMempoolFlags to make more or less check of script, the type of value is uint
		PromiscuousMempoolFlags string
		Par                     int `default:"32"`
	}
	TxOut struct {
		DustRelayFee int64 `default:"83"`
	}
	Chain struct {
		AssumeValid         string
		UtxoHashStartHeight int32 `default:"-1"`
		UtxoHashEndHeight   int32 `default:"-1"`
	}
	Mining struct {
		BlockMinTxFee int64  // default DefaultBlockMinTxFee
		BlockMaxSize  uint64 // default DefaultMaxGeneratedBlockSize
		Strategy      string `default:"ancestorfeerate"` // option:ancestorfee/ancestorfeerate
	}
	PProf struct {
		IP   string `default:"localhost"`
		Port string `default:"6060"`
	}
	BlockIndex struct {
		CheckBlockIndex bool
	}
	Wallet struct {
		Enable              bool `default:"false"`
		Broadcast           bool `default:"false"`
		SpendZeroConfChange bool `default:"true"`
	}
}

var (
	Cfg     *Configuration
	Args    *Opts
	DataDir string
)

// InitConfig init configuration
func InitConfig(args []string) *Configuration {
	// parse command line parameter to set program datadir
	defaultDataDir := AppDataDir(defaultDataDirname, false)
	DataDir = defaultDataDir

	opts, err := InitArgs(args)
	if err != nil {
		//fmt.Println("\033[0;31mparse cmd line fail: %v\033[0m\n")
		return nil
	}

	Args = opts

	if opts.ShowVersion {
		fmt.Println(AppName, "version", version())
		os.Exit(0)
	}

	if opts.RegTest && opts.TestNet {
		panic("Both testnet and regtest are true")
	}

	if len(opts.DataDir) > 0 {
		DataDir = opts.DataDir
	}

	destConfig := DataDir + "/" + defaultConfigFilename

	if opts.TestNet {
		DataDir = path.Join(DataDir, "testnet")
	} else if opts.RegTest {
		DataDir = path.Join(DataDir, "regtest")
	}

	if !FileExists(DataDir) {
		err := os.MkdirAll(DataDir, os.ModePerm)
		if err != nil {
			panic("datadir create failed: " + err.Error())
		}
	}

	if !FileExists(destConfig) {
		srcCfgDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		if err = tryCopyDefaultCfg(srcCfgDir, destConfig); err != nil {

			if gopath := os.Getenv("GOPATH"); gopath != "" {
				srcCfgDir = gopath + "/src/" + defaultProjectDir

				err = tryCopyDefaultCfg(srcCfgDir, destConfig)
				if err != nil {
					panic("copy default config failed, " + err.Error())
				}
			} else {
				panic("can not find config from gopath or ./conf")
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
	file := must(os.Open(destConfig)).(*os.File)
	defer file.Close()
	must(nil, viper.ReadConfig(file))
	must(nil, viper.Unmarshal(config))

	// set data dir
	config.DataDir = DataDir
	config.Reindex = opts.Reindex
	config.Excessiveblocksize = opts.Excessiveblocksize
	config.Mempool.LimitAncestorCount = opts.Limitancestorcount
	config.Script.PromiscuousMempoolFlags = opts.PromiscuousMempoolFlags
	config.Mempool.MaxPoolSize = opts.MaxMempool

	config.RPC.RPCKey = filepath.Join(defaultDataDir, "rpc.key")
	config.RPC.RPCCert = filepath.Join(defaultDataDir, "rpc.cert")

	if opts.RegTest {
		config.P2PNet.RegTest = true
		if !viper.IsSet("BlockIndex.CheckBlockIndex") {
			config.BlockIndex.CheckBlockIndex = true
		}
	}
	if opts.TestNet {
		config.P2PNet.TestNet = true
	}

	if opts.UtxoHashStartHeigh >= 0 && opts.UtxoHashEndHeigh <= opts.UtxoHashStartHeigh {
		panic("utxohashstartheight should less than utxohashendheight")
	}

	if opts.UtxoHashStartHeigh >= 0 {
		config.Chain.UtxoHashStartHeight = opts.UtxoHashStartHeigh
		config.Chain.UtxoHashEndHeight = opts.UtxoHashEndHeigh
	}
	if opts.Excessiveblocksize <= 1000000 {
		println("Error: Excessive block size must be > 1,000,000 bytes (1MB)")
		return nil
	}
	if opts.Excessiveblocksize < config.Mining.BlockMaxSize {
		println("Error: Max generated block size (blockmaxsize) cannot exceed the excessive block size (excessiveblocksize)")
		return nil
	}
	if len(opts.Whitelists) > 0 {
		initWhitelists(config, opts)
	}
	if opts.SpendZeroConfChange == 0 {
		config.Wallet.SpendZeroConfChange = false
	}
	if opts.BanScore > 0 {
		config.P2PNet.BanThreshold = opts.BanScore
	}
	if opts.MaxTimeAdjustment > 0 {
		config.P2PNet.MaxTimeAdjustment = opts.MaxTimeAdjustment
	}

	return config
}

func initWhitelists(config *Configuration, opts *Opts) {
	var ip net.IP
	config.P2PNet.Whitelists = make([]*net.IPNet, 0, len(opts.Whitelists))
	for _, addr := range opts.Whitelists {
		_, ipnet, err := net.ParseCIDR(addr)

		if err != nil {
			ip = net.ParseIP(addr)
			if ip == nil {
				fmt.Fprintln(os.Stderr, fmt.Sprintf("[Error]The whitelist value of '%s' is invalid", addr))
				continue
			}

			var bits int
			if ip.To4() == nil {
				// IPv6
				bits = 128
			} else {
				bits = 32
			}

			ipnet = &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(bits, bits),
			}
		}
		config.P2PNet.Whitelists = append(config.P2PNet.Whitelists, ipnet)
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
	validate := validator.New(&validator.Config{TagName: "validate"})
	return validate.Struct(c)
}

func FileExists(datadir string) bool {
	_, err := os.Stat(datadir)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return true
}

func SetUnitTestDataDir(config *Configuration) (dirPath string, err error) {
	oldDirParent := filepath.Dir(DataDir)
	testDataDir, err := ioutil.TempDir(oldDirParent, "unitTestDataDir")
	if err != nil {
		return "", errors.New("test data directory create failed: " + err.Error())
	}

	defaultDataDir := AppDataDir(defaultDataDirname, false)
	_, err = CopyFile(filepath.Join(defaultDataDir, defaultConfigFilename), filepath.Join(testDataDir, defaultConfigFilename))
	if err != nil {
		return "", err
	}

	DataDir = testDataDir
	config.DataDir = testDataDir

	return testDataDir, nil
}

// GetSubVersionEB converts MaxBlockSize from byte to
// MB with a decimal precision one digit rounded down
// E.g.
// 1660000 -> 1.6
// 2010000 -> 2.0
// 1000000 -> 1.0
// 230000  -> 0.2
// 50000   -> 0.0
// NB behavior for EB<1MB not standardized yet still
// the function applies the same algo used for
// EB greater or equal to 1MB
func GetSubVersionEB() string {
	// Prepare EB string we are going to add to SubVer:
	// 1) translate from byte to MB and convert to string
	// 2) limit the EB string to the first decimal digit (floored)
	ebMBs := Cfg.Excessiveblocksize / (OneMegaByte / 10)
	return "EB" + fmt.Sprintf("%.1f", float64(ebMBs)/10.0)
}

func GetUserAgent(name string, version string, comments []string) string {
	agentComments := make([]string, 0, 1+len(comments))
	// format excessive blocksize value
	ebMsg := GetSubVersionEB()
	agentComments = append(agentComments, ebMsg)
	agentComments = append(agentComments, comments...)
	userAgent := fmt.Sprintf("/%s:%s(%s)/", name, version, strings.Join(agentComments, "; "))
	return userAgent
}

func tryCopyDefaultCfg(dir string, destConfig string) error {
	srcConfig := dir + "/conf/" + defaultConfigFilename
	if !FileExists(srcConfig) {
		return errors.New("try copy, default config not in:" + srcConfig)
	}

	_, err := CopyFile(srcConfig, destConfig)
	return err
}

func version() string {
	version := fmt.Sprintf("%d.%d.%d", AppMajor, AppMinor, AppPatch)

	// Append pre-release version if there is one.  The hyphen called for
	// by the semantic versioning spec is automatically appended and should
	// not be contained in the pre-release string.  The pre-release version
	// is not appended if it contains invalid characters.
	if AppPreRelease != "" {
		version += "-" + AppPreRelease
	}

	return version
}
