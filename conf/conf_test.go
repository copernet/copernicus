package conf

//
//import (
//	"fmt"
//	"io/ioutil"
//	"math/rand"
//	"os"
//	"testing"
//
//	. "github.com/smartystreets/goconvey/convey"
//)
//
//var confData = []byte(`
//go_version: 1.9.2
//version: 1.0.0
//build_date: 20180428
//service:
//  address: 10.0.0.0/8
//http:
//  host: 127.0.0.1
//  port: 8080
//  mode: test
//rpc:
//  host: 127.0.0.1
//  port: 9552
//log:
//  level: error
//  format: json
//`)
//
//func TestInitConfig(t *testing.T) {
//	Convey("Given config file", t, func() {
//		filename := fmt.Sprintf("conf_test%04d.yml", rand.Intn(9999))
//		os.Setenv(ConfEnv, filename)
//		ioutil.WriteFile(filename, confData, 0664)
//
//		Convey("When init configuration", func() {
//			config := initConfig()
//
//			Convey("Configuration should resemble default configuration", func() {
//				expected := &Configuration{}
//				expected.Service.Address = "10.0.0.0/8"
//				expected.HTTP.Host = "127.0.0.1"
//				expected.HTTP.Port = 8080
//				expected.HTTP.Mode = "test"
//				expected.Log.Format = "json"
//				expected.Log.Level = "error"
//				expected.GoVersion = "1.9.2"
//				expected.Version = "1.0.0"
//				expected.BuildDate = "20180428"
//				expected.RPC.Host = "127.0.0.1"
//				expected.RPC.Port = 9552
//				So(config, ShouldResemble, expected)
//			})
//		})
//
//		Reset(func() {
//			os.Unsetenv(ConfEnv)
//			os.Remove(filename)
//		})
//	})
//}
