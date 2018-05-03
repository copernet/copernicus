package conf

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/astaxie/beego/config"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcsuite/btcd/chaincfg"
)

var AppConf *AppConfig

const (
	DefaultConnectTimeout = time.Second * 30
)

type AppConfig struct {
	DataDir            string        `short:"b" long:"datadir" description:"Directory to store data"`
	ShowVersion        bool          `short:"v" long:"version" description:"Display version in"`
	NoPeerBloomFilters bool          `long:"nopeerbloomfilters" description:"Disable bloom filtering support"`
	MaxPeers           int           `long:"maxpeers" description:"Max number of inbound and outbound peers"`
	DisableBanning     bool          `long:"nobanning" description:"Disable banning of misbehaving peers"`
	BanDuration        time.Duration `long:"banduration" description:"How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second"`
	BanThreshold       uint32        `long:"banthreshold" description:"Maximum allowed ban score before disconnecting and banning misbehaving peers."`

	Listeners []string `long:"listen" description:"Add an interface/port to listen for connections (default all interfaces port: 8333, testnet: 18333)"`

	NoOnion        bool `long:"noonion" description:"Disable connecting to tor hidden services"`
	TorIsolation   bool `long:"torisolation" description:"Enable Tor stream isolation by randomizing user credentials for each connection."`
	TestNet3       bool `long:"testnet" description:"Use the test network"`
	RegressionTest bool `long:"regtest" description:"Use the regression test network"`
	SimNet         bool `long:"simnet" description:"Use the simulation test network"`

	DisableListen bool `long:"nolisten" description:"Disable listening for incoming connections -- NOTE: Listening is automatically disabled if the --conn or --proxy options are used without also specifying listen interfaces via --listen"`

	lookup         utils.LookupFunc
	DisableDNSSeed bool `long:"nodnsseed" description:"Disable DNS seeding for peers"`

	oniondial func(string, string, time.Duration) (net.Conn, error)
	dial      func(string, string, time.Duration) (net.Conn, error)

	LogModule []string

	ConfigFile           string   `short:"C" long:"configfile" description:"Path to configuration file"`
	LogDir               string   `long:"logdir" description:"Directory to log output."`
	AddPeers             []string `short:"a" long:"addpeer" description:"Add a peer to connect with at startup"`
	ConnectPeers         []string `long:"connect" description:"Connect only to the specified peers at startup"`
	Whitelists           []string `long:"whitelist" description:"Add an IP network or IP that will not be banned. (eg. 192.168.1.0/24 or ::1)"`
	RPCUser              string   `short:"u" long:"rpcuser" description:"Username for RPC connections"`
	RPCPass              string   `short:"P" long:"rpcpass" default-mask:"-" description:"Password for RPC connections"`
	RPCLimitUser         string   `long:"rpclimituser" description:"Username for limited RPC connections"`
	RPCLimitPass         string   `long:"rpclimitpass" default-mask:"-" description:"Password for limited RPC connections"`
	RPCListeners         []string `long:"rpclisten" description:"Add an interface/port to listen for RPC connections (default port: 8334, testnet: 18334)"`
	RPCCert              string   `long:"rpccert" description:"File containing the certificate file"`
	RPCKey               string   `long:"rpckey" description:"File containing the certificate key"`
	RPCMaxClients        int      `long:"rpcmaxclients" description:"Max number of RPC clients for standard connections"`
	RPCMaxWebsockets     int      `long:"rpcmaxwebsockets" description:"Max number of RPC websocket connections"`
	RPCMaxConcurrentReqs int      `long:"rpcmaxconcurrentreqs" description:"Max number of concurrent RPC requests that may be processed concurrently"`
	RPCQuirks            bool     `long:"rpcquirks" description:"Mirror some JSON-RPC quirks of Bitcoin Core -- NOTE: Discouraged unless interoperability issues need to be worked around"`
	DisableRPC           bool     `long:"norpc" description:"Disable built-in RPC server -- NOTE: The RPC server is disabled by default if no rpcuser/rpcpass or rpclimituser/rpclimitpass is specified"`
	DisableTLS           bool     `long:"notls" description:"Disable TLS for the RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	ExternalIPs          []string `long:"externalip" description:"Add an ip to the list of local addresses we claim to listen on to peers"`
	Proxy                string   `long:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	ProxyUser            string   `long:"proxyuser" description:"Username for proxy server"`
	ProxyPass            string   `long:"proxypass" default-mask:"-" description:"Password for proxy server"`
	OnionProxy           string   `long:"onion" description:"Connect to tor hidden services via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	OnionProxyUser       string   `long:"onionuser" description:"Username for onion proxy server"`
	OnionProxyPass       string   `long:"onionpass" default-mask:"-" description:"Password for onion proxy server"`
	AddCheckpoints       []string `long:"addcheckpoint" description:"Add a custom checkpoint.  Format: '<height>:<hash>'"`
	DisableCheckpoints   bool     `long:"nocheckpoints" description:"Disable built-in checkpoints.  Don't do this unless you know what you're doing."`
	DbType               string   `long:"dbtype" description:"Database backend to use for the Block Chain"`
	Profile              string   `long:"profile" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65536"`
	CPUProfile           string   `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	DebugLevel           string   `short:"d" long:"debuglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list available subsystems"`
	Upnp                 bool     `long:"upnp" description:"Use UPnP to map our listening port outside of NAT"`
	MinRelayTxFee        float64  `long:"minrelaytxfee" description:"The minimum transaction fee in BTC/kB to be considered a non-zero fee."`
	FreeTxRelayLimit     float64  `long:"limitfreerelay" description:"Limit relay of transactions with no transaction fee to the given amount in thousands of bytes per minute"`
	NoRelayPriority      bool     `long:"norelaypriority" description:"Do not require free or low-fee transactions to have high priority for relaying"`
	MaxOrphanTxs         int      `long:"maxorphantx" description:"Max number of orphan transactions to keep in memory"`
	Generate             bool     `long:"generate" description:"Generate (mine) bitcoins using the CPU"`
	MiningAddrs          []string `long:"miningaddr" description:"Add the specified payment address to the list of addresses to use for generated blocks -- At least one address is required if the generate option is set"`
	BlockMinSize         uint32   `long:"blockminsize" description:"Mininum block size in bytes to be used when creating a block"`
	BlockMaxSize         uint32   `long:"blockmaxsize" description:"Maximum block size in bytes to be used when creating a block"`
	BlockMinWeight       uint32   `long:"blockminweight" description:"Mininum block weight to be used when creating a block"`
	BlockMaxWeight       uint32   `long:"blockmaxweight" description:"Maximum block weight to be used when creating a block"`
	BlockPrioritySize    uint32   `long:"blockprioritysize" description:"Size in bytes for high-priority/low-fee transactions when creating a block"`
	UserAgentComments    []string `long:"uacomment" description:"Comment to add to the user agent -- See BIP 14 for more information."`
	SigCacheMaxSize      uint     `long:"sigcachemaxsize" description:"The maximum number of entries in the signature verification cache"`
	BlocksOnly           bool     `long:"blocksonly" description:"Do not accept transactions from remote peers."`
	TxIndex              bool     `long:"txindex" description:"Maintain a full hash-based transaction index which makes all transactions available via the getrawtransaction RPC"`
	DropTxIndex          bool     `long:"droptxindex" description:"Deletes the hash-based transaction index from the database on start up and then exits."`
	AddrIndex            bool     `long:"addrindex" description:"Maintain a full address-based transaction index which makes the searchrawtransactions RPC available"`
	DropAddrIndex        bool     `long:"dropaddrindex" description:"Deletes the address-based transaction index from the database on start up and then exits."`
	RelayNonStd          bool     `long:"relaynonstd" description:"Relay non-standard transactions regardless of the default settings for the active network."`
	RejectNonStd         bool     `long:"rejectnonstd" description:"Reject non-standard transactions regardless of the default settings for the active network."`
	addCheckpoints       []chaincfg.Checkpoint
	whitelists           []*net.IPNet
}

func init() {
	_, err := config.NewConfig("ini", "init.conf")
	if err != nil {
		fmt.Println(err)
	}
	// todo unable to pass in unit test
	// if appConf != nil {
	// 	contentTimeout := appConf.String("Timeout::connectTimeout")
	// 	log.Info("read conf timeout is  %s", contentTimeout)
	// 	logDir := appConf.String("Log::dir")
	// 	log.Info("logger dir is %s", logDir)
	// 	logLevel := appConf.String("Log::level")
	// 	log.Info("logger dir is %s", logLevel)
	// }
	AppConf = loadConfig()
}

func loadConfig() *AppConfig {
	appConfig := AppConfig{
		ShowVersion:        true,
		NoPeerBloomFilters: true,
		DataDir:            GetDataPath(),
	}
	appConfig.dial = net.DialTimeout
	appConfig.lookup = net.LookupIP
	return &appConfig
}

func GetDataPath() string {
	dataPath := filepath.Clean(utils.MergePath("cp"))
	if utils.PathExists(dataPath) {
		err := utils.MakePath(dataPath)
		if err != nil {
			panic(err)
		}
	}
	return dataPath
}

func AppLookup(host string) ([]net.IP, error) {
	if strings.HasSuffix(host, ".onion") {
		return nil, fmt.Errorf("attempt to resolve tor address %s", host)
	}
	return AppConf.lookup(host)
}
func AppDial(address net.Addr) (net.Conn, error) {
	if strings.Contains(address.String(), ".onion:") {
		return AppConf.oniondial(address.Network(), address.String(), DefaultConnectTimeout)
	}
	return AppConf.dial(address.Network(), address.String(), DefaultConnectTimeout)
}
