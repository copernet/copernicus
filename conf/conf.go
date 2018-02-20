package conf

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/astaxie/beego/config"
	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/utils"
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
}

func init() {
	log := logs.NewLogger()
	_, err := config.NewConfig("ini", "init.conf")
	if err != nil {
		log.Error(err.Error())
	}
	//todo unable to pass in unit test
	//if appConf != nil {
	//	contentTimeout := appConf.String("Timeout::connectTimeout")
	//	log.Info("read conf timeout is  %s", contentTimeout)
	//	logDir := appConf.String("Log::dir")
	//	log.Info("logger dir is %s", logDir)
	//	logLevel := appConf.String("Log::level")
	//	log.Info("logger dir is %s", logLevel)
	//}
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
