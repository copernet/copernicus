package log

import (
	"path/filepath"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcsuite/btcutil"
	"github.com/gin-gonic/gin/json"
)

const defaultLogDirname = "logs"

func Print(module string, level string, format string, reason ...interface{}) {
	if isIncludeModule(module) {
		switch level {
		case "emergency":
			logs.Emergency(format, reason)
		case "alert":
			logs.Alert(format, reason)
		case "critical":
			logs.Critical(format, reason)
		case "error":
			logs.Error(format, reason)
		case "warn":
			logs.Warn(format, reason)
		case "info":
			logs.Info(format, reason)
		case "debug":
			logs.Debug(format, reason)
		case "notice":
			logs.Notice(format, reason)
		}
	}
}

func isIncludeModule(module string) bool {
	for _, item := range conf.AppConf.LogModule {
		if item == module {
			return true
		}
	}
	return false
}

func init() {
	defaultHomeDir := btcutil.AppDataDir("copernicus", false)
	logDir := filepath.Join(defaultHomeDir, defaultLogDirname)

	logConf := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
		Daily    bool   `json:"daily"`
	}{
		FileName: logDir,
		Level:    getLevel("debug"),
		Daily:    false,
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}
	logs.SetLogger(logs.AdapterFile, string(configuration))
}
