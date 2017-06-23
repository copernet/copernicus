package conf

import (
	mlog "copernicus/log"
	
	"github.com/astaxie/beego/config"
	"github.com/astaxie/beego/logs"
	"net"
	"strings"
	"fmt"
	"copernicus/network"
	"time"
)

var AppConf *AppConfig

type AppConfig struct {
	DataDir            string        `short:"b" long:"datadir" description:"Directory to store data"`
	ShowVersion        bool `short:"v" long:"version" description "Disaplay version in"`
	NoPeerBloomFilters bool `long:"nopeerbloomfilters" descriptopn"Disable bloom filtering support"`
	MaxPeers           int           `long:"maxpeers" description:"Max number of inbound and outbound peers"`
	DisableBanning     bool          `long:"nobanning" description:"Disable banning of misbehaving peers"`
	BanDuration          time.Duration `long:"banduration" description:"How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second"`
	BanThreshold         uint32        `long:"banthreshold" description:"Maximum allowed ban score before disconnecting and banning misbehaving peers."`
	
	NoOnion              bool          `long:"noonion" description:"Disable connecting to tor hidden services"`
	TorIsolation         bool          `long:"torisolation" description:"Enable Tor stream isolation by randomizing user credentials for each connection."`
	TestNet3             bool          `long:"testnet" description:"Use the test network"`
	RegressionTest       bool          `long:"regtest" description:"Use the regression test network"`
	SimNet               bool          `long:"simnet" description:"Use the simulation test network"`
	
	DisableListen        bool          `long:"nolisten" description:"Disable listening for incoming connections -- NOTE: Listening is automatically disabled if the --connect or --proxy options are used without also specifying listen interfaces via --listen"`
	
	lookup             network.LookupFunc
}

func init() {
	log := logs.NewLogger()
	appConf, err := config.NewConfig("ini", "init.conf")
	if err != nil {
		log.Error(err.Error())
	}
	contentTimeout := appConf.String("Timeout::connectTimeout")
	log.Info("read conf timeout is  %s", contentTimeout)
	logDir := appConf.String("Log::dir")
	log.Info("log dir is %s", logDir)
	logLevel := appConf.String("Log::level")
	log.Info("log dir is %s", logLevel)
	if err := mlog.InitLogger(logDir, logLevel); err != nil {
		log.Error(err.Error())
	}
	AppConf, _ = loadConfig()
}

func loadConfig() (*AppConfig, error) {
	appConfig := AppConfig{
		ShowVersion:        true,
		NoPeerBloomFilters: true,
		DataDir:            "copernicus"}
	return &appConfig, nil
	
}

func AppLookup(host string) ([]net.IP, error) {
	if strings.HasSuffix(host, ".onion") {
		return nil, fmt.Errorf("attempt to resolve tor address %s", host)
	}
	return AppConf.lookup(host)
}
