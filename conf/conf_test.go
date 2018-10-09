package conf

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
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
  Level: error
  Format: json
`)

func initConfig() *configuration {
	config := &configuration{}
	viper.SetConfigType("yaml")

	filename := fmt.Sprintf("conf_test%04d.yml", rand.Intn(9999))
	err := ioutil.WriteFile(filename, confData, 0664)
	if err != nil {
		fmt.Printf("write config file failed:%s", err)
	}

	//parse struct tag
	c := configuration{}
	t := reflect.TypeOf(c)
	v := reflect.ValueOf(c)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if v.Field(i).Type().Kind() != reflect.Struct {
			key := field.Name
			value := field.Tag.Get(tagName)
			//set default value
			viper.SetDefault(key, value)
			//log.Printf("key is: %v,value is: %v\n", key, value)
		} else {
			structField := v.Field(i).Type()
			for j := 0; j < structField.NumField(); j++ {
				key := structField.Field(j).Name
				values := structField.Field(j).Tag.Get(tagName)
				viper.SetDefault(key, values)
			}
			continue
		}
	}

	// parse config
	file := must(os.Open(filename)).(*os.File)
	defer file.Close()
	defer os.Remove(filename)
	must(nil, viper.ReadConfig(file))
	must(nil, viper.Unmarshal(config))

	return config
}

type configuration struct {
	GoVersion string
	Version   string
	BuildDate string
	Service   struct {
		Address string
	}
	HTTP struct {
		Host string
		Port int
		Mode string
	}
	RPC struct {
		Host string
		Port int
	}
	Log struct {
		Level  string
		Format string
	}
}

func TestInitConfig(t *testing.T) {
	config := initConfig()
	expected := &configuration{}
	expected.Service.Address = "10.0.0.0/8"
	expected.HTTP.Host = "127.0.0.1"
	expected.HTTP.Port = 8080
	expected.HTTP.Mode = "test"
	expected.Log.Format = "json"
	expected.Log.Level = "error"
	expected.GoVersion = "1.9.2"
	expected.Version = "1.0.0"
	expected.BuildDate = "20180428"
	expected.RPC.Host = "127.0.0.1"
	expected.RPC.Port = 9552

	if !reflect.DeepEqual(config, expected) {
		t.Error("Expected value is not equal to the actual value obtained")
	}
}

func TestSetDefault(t *testing.T) {
	viper.SetDefault("key", 100)
	if viper.GetInt("key") != 100 {
		t.Error("set default(key) error")
	}

	viper.SetDefault("rpc.user", "admin")
	if viper.GetString("rpc.user") != "admin" {
		t.Error("set default(rpc.user) error")
	}
}

func TestCopyFile(t *testing.T) {
	nameSRC := "conf.txt"
	nameDES := "copy_conf.txt"
	content := "hello,copernicus"
	data := []byte(content)
	err := ioutil.WriteFile(nameSRC, data, 0644)
	if err != nil {
		t.Errorf("write conf file failed: %s\n ", err)
	}
	defer os.Remove(nameSRC)

	writeNum, err := CopyFile(nameSRC, nameDES)
	if err != nil {
		t.Errorf("copy file failed: %s\n", err)
	}

	readNum, err := ioutil.ReadFile(nameDES)
	if int64(len(readNum)) != writeNum {
		t.Errorf("error copying the contents of the file: %s\n", err)
	}
	defer os.Remove(nameDES)
}

func TestExistDataDir(t *testing.T) {
	fileTrue := "conf.txt"
	fileFalse := "confNo.txt"

	fileTrue, err := ioutil.TempDir("", fileTrue)
	if err != nil {
		t.Fatalf("generate temp db path failed: %s\n", err)
	}
	defer os.Remove(fileTrue)

	if !ExistDataDir(fileTrue) {
		t.Errorf("the fileTrue file should exist!")
	}

	if ExistDataDir(fileFalse) {
		t.Errorf("the fileFalse file shouldn't exist!")
	}
}
