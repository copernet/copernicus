package conf

import (
	"log"
	"os"
	"path"
	"reflect"
	"runtime"

	"github.com/spf13/viper"
)

const (
	tagName = "default"
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

type Configuration struct {
	GoVersion string `validate:"require"` //description:"Display version information and exit"
	Version   string `validate:"require"` //description:"Display version information of copernicus"
	BuildDate string `validate:"require"` //description:"Display build date of copernicus"
	Service   struct {
		Address string `default:"1.0.0.1:80"`
	}
	HTTP struct {
		Host string `validate:"require"`
		Port int
		Mode string
	}
	RPC struct {
		Host string `validate:"require"` //description:"Define host of RPC"
		Port int    //description:"Define port of RPC"
	}
	Log struct {
		Level  string //description:"Define level of log,include trace, debug, info, warn, error"
		Format string
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
