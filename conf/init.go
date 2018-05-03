package conf

import (
	"os"

	"github.com/spf13/viper"
	//"gopkg.in/go-playground/validator.v8"
	"path"
	"runtime"
)

const (
	ConfEnv = "DSP_ALLOT_CONF"
)

var Cfg *Configuration

// init configuration
func initConfig() *Configuration {
	config := &Configuration{}

	viper.SetEnvPrefix("copernicus")
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("get current file path failed.")
	}
	filePath := path.Join(path.Dir(filename), "./conf.yml")
	viper.SetDefault("conf", filePath)

	// get config file path from environment
	conf := viper.GetString("conf")

	// parse config
	file := Must(os.Open(conf)).(*os.File)
	defer file.Close()

	Must(nil, viper.ReadConfig(file))
	Must(nil, viper.Unmarshal(config))
	//TODO:
	//Must(nil, config.Validate())

	return config
}

type Configuration struct {
	GoVersion string `mapstructure:"go_version" validate:"required"`
	Version   string `mapstructure:"version" validate:"required"`
	BuildDate string `mapstructure:"build_date" validate:"required"`
	Service   struct {
		Address string `mapstructure:"address" validate:"required,cidr"`
	} `mapstructure:"service" validate:"required"`
	HTTP struct {
		Host string `mapstructure:"host" validate:"required,ip"`
		Port int    `mapstructure:"port" validate:"required"`
		Mode string `mapstructure:"mode" validate:"required,eq=release|eq=test|eq=debug"`
	} `mapstructure:"http" validate:"required"`
	RPC struct {
		RPCUser              string
		RPCPass              string
		RPCLimitUser         string
		RPCLimitPass         string
		RPCListeners         []string
		RPCCert              string
		RPCKey               string
		RPCMaxClients        int
		RPCMaxWebsockets     int
		RPCMaxConcurrentReqs int
		RPCQuirks            bool
		DisableRPC           bool
		DisableTLS           bool
	} `mapstructure:"rpc" validate:"required"`
	Log struct {
		Level  string `mapstructure:"level" validate:"required,eq=debug|eq=info|eq=warn|eq=error|eq=fatal|eq=panic"`
		Format string `mapstructure:"format" validate:"required,eq=text|eq=json"`
	} `mapstructure:"log" validate:"required"`
}

func Must(i interface{}, err error) interface{} {
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
//
//	return validate.Struct(c)
//}
