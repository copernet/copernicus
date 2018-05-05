package conf

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
)

var confData = []byte(`
GoVersion: 1.9.2
Version: 1.0.0
BuildDate: 20180428
Service:
  Address: 10.0.0.0/8
HTTP:
  Host: 127.0.0.1
  Port: 8080
  Mode: test
RPC:
  Host: 127.0.0.1
  Port: 9552
Log:
  Format: json
`)

func TestInitConfig(t *testing.T) {
	//Convey("Given config file", t, func() {
	//	filename := fmt.Sprintf("conf_test%04d.yml", rand.Intn(9999))
	//	ioutil.WriteFile(filename, confData, 0664)
	//
	//	Convey("When init configuration", func() {
	//		config := initConfig()
	//
	//		Convey("Configuration should resemble default configuration", func() {
	//			expected := &Configuration{}
	//			expected.Service.Address = "10.0.0.0/8"
	//			expected.HTTP.Host = "127.0.0.1"
	//			expected.HTTP.Port = 8080
	//			expected.HTTP.Mode = "test"
	//			expected.Log.Format = "json"
	//			//expected.Log.Level = "info"
	//			expected.GoVersion = "1.9.2"
	//			expected.Version = "1.0.0"
	//			expected.BuildDate = "20180428"
	//			expected.RPC.Host = "127.0.0.1"
	//			expected.RPC.Port = 9552
	//			So(config, ShouldResemble, expected)
	//		})
	//	})
	//
	//	v := viper.GetString("HTTP.Host")
	//	fmt.Println(v)
	//
	//	Reset(func() {
	//		os.Remove(filename)
	//	})
	//})
}

func TestSetDefault(t *testing.T) {
	viper.SetDefault("test2", 100)
	if viper.GetInt("test2") != 100 {
		t.Error("set default error")
	}

	viper.SetDefault("rpc.user", "qshuai")
	if viper.GetString("rpc.user") != "qshuai" {
		t.Error("set default error")
	}
	fmt.Println(viper.GetString("rpc.user"))

	viper.SetDefault("Log.Level", "debug")

	fmt.Println(viper.GetString("Log.Level"))
}
