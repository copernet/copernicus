package conf

import (
	"os"

	"github.com/spf13/viper"
	"gopkg.in/go-playground/validator.v8"
)

const (
	ConfEnv = "DSP_ALLOT_CONF"
)

func initConfig() *Configuration {
	config := &Configuration{}

	viper.SetEnvPrefix("copernicus")
	viper.AutomaticEnv()
	viper.SetConfigType("yaml")
	viper.SetDefault("conf", "./conf.yml")

	// get config file path from environment
	conf := viper.GetString("conf")

	// parse config
	file := Must(os.Open(conf)).(*os.File)
	defer file.Close()

	Must(nil, viper.ReadConfig(file))
	Must(nil, viper.Unmarshal(config))
	//TODO:
	Must(nil, config.Validate())

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
		Host string `mapstructure:"host" validate:"required,ip"`
		Port int    `mapstructure:"port" validate:"required"`
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

// Validate validates configuration
func (c Configuration) Validate() error {
	validate := validator.New(&validator.Config{TagName: "validate"})

	return validate.Struct(c)
}
