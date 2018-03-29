package conf

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func NewConfig(configFilePath string) {
	viper.SetConfigName("config")
	viper.AddConfigPath(configFilePath)

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s ", err))
	}

	// use example
	ConfigOption("clientBroadcasts", true)

	if viper.GetBool("data") {
		ConfigOption("dataHost", "127.0.0.1")
	}
}

func ConfigOption(key string, defaultValue interface{}) string {
	viper.SetDefault(key, defaultValue)

	return key
}

// Asserts that the chosen value exists on the local file system by panicking if it doesn't
func fileOption(key string) {
	chosenValue := viper.GetString(key)

	if _, err := os.Stat(chosenValue); err != nil {
		panic(fmt.Errorf("chosen option %s does not exist ", chosenValue))
	}
}
