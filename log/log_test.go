package log

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/copernet/copernicus/conf"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

const result = "success"

var (
	level  = []string{"emergency", "Alert", "critical", "error", "warn", "info", "debug", "Notice"}
	module = []string{"conf", "errcode", "logic", "model", "net", "peer", "persist", "rpc", "service", "util"}
)

func TestMain(m *testing.M) {
	conf.Cfg = conf.InitConfig([]string{})
	m.Run()
}

func TestLog(t *testing.T) {
	path, err := ioutil.TempDir("", "logtest")
	if err != nil {
		panic("generate temp path failed")
	}
	defer os.Remove(path)

	logConf := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
		Daily    bool   `json:"daily"`
	}{
		FileName: path + ".log",
		Level:    GetLevel(conf.Cfg.Log.Level),
		Daily:    false,
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}

	//init the log instance
	logs.EnableFuncCallDepth(true)
	logs.SetLogger("file", string(configuration))
	logs.SetLogFuncCallDepth(4)

	//run Print() if logic
	for _, moduleStr := range module {
		mapModule = map[string]struct{}{
			moduleStr: {},
		}
		for _, levelStr := range level {
			format := fmt.Sprintf("module[%s]: %s", moduleStr, levelStr)
			Print(moduleStr, levelStr, format)
		}
	}

	//test Alert log level
	Alert("print alert level log", result)
	//test Critical log level
	Critical("print critical level log", result)
	//test Debug log level
	Debug("print debug level log", result)
	//test Notice log level
	Notice("print notice level log", result)
	//test Trace log level
	Trace("print trace level log", result)
	//test Warn log level
	Warn("print warn level log", result)
	//test Emergency log level
	Emergency("print emergency level log", result)
	//test Error log level
	Error("print error level log", result)
	//test Info log level
	Info("print info level log", result)
	//test Warning log level
	Warning("print warning level log", result)
	//test Informational log level
	Informational("print information level log", result)

	//read file
	str, err := ioutil.ReadFile(path + ".log")
	if err != nil {
		t.Errorf("read temp path failed: %s\n", err)
	}
	if strings.Contains(string(str), errModuleNotFound) {
		t.Fatalf("the Print() implement error:%s\n", string(str))
	}

	if !strings.Contains(string(str), string(str)) {
		t.Fatalf("print Alert log level failed: %s\n", string(str))
	}

	//reset map module
	mapModule = make(map[string]struct{})
	level = append(level, "default")

	path1, err := ioutil.TempDir("", "logtest1")
	if err != nil {
		panic("generate temp path failed")
	}
	defer os.Remove(path1)

	logConf1 := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
		Daily    bool   `json:"daily"`
	}{
		FileName: path1 + ".log",
		Level:    GetLevel(conf.Cfg.Log.Level),
		Daily:    false,
	}

	conf1, err := json.Marshal(logConf1)
	if err != nil {
		panic(err)
	}

	//init the log instance
	logs.EnableFuncCallDepth(true)
	logs.SetLogger("file", string(conf1))
	logs.SetLogFuncCallDepth(4)

	//run Print() else logic
	for _, moduleStr := range module {
		for _, levelStr := range level {
			format := fmt.Sprintf("module[%s]: %s", moduleStr, levelStr)
			Print(moduleStr, levelStr, format)
		}
	}

	//read file
	strPath, err := ioutil.ReadFile(path + ".log")
	if err != nil {
		t.Errorf("read temp path failed: %s\n", err)
	}
	if !strings.Contains(string(strPath), errModuleNotFound) {
		t.Fatalf("the Print() implement error:%s\n", string(strPath))
	}

}

func pathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func TestInit(t *testing.T) {
	conf.Cfg = conf.InitConfig([]string{})

	path, err := ioutil.TempDir("", "initLog")
	if err != nil {
		panic("generate temp path failed")
	}
	defer os.RemoveAll(path)

	logConf := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
		Daily    bool   `json:"daily"`
	}{
		FileName: path + ".log",
		Level:    GetLevel(conf.Cfg.Log.Level),
		Daily:    false,
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}
	Init(string(configuration))
	if !pathExist(string(path) + ".log") {
		t.Error("not init log file")
	}
}
