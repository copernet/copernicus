package log

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/astaxie/beego/logs"
)

var mlog *logs.BeeLogger

//todo config color of debug log
//todo output log to elasticSearch

type LogConfig struct {
	Filename string `json:"filename"`
	Level    int    `json:"level,omitempty"`
	Rotate   bool   `json:"rotate,omitempty"`
	Daily    bool   `json:"daily,omitempty"`
	MaxDays  int64  `json:"maxdays,omitempty"`
	MaxLines int    `json:"maxlines,omitempty"`
	MaxSize  int    `json:"maxsize,omitempty"`
}


func init() {
	mlog = logs.NewLogger()
	mlog.EnableFuncCallDepth(true)
	logs.Async()
}

func validLogLevel(str_level string) (level int, ok bool) {
	ok = true
	switch str_level {
	case "emergecy":
		level = logs.LevelEmergency
	case "alert":
		level = logs.LevelAlert
	case "critical":
		level = logs.LevelCritical
	case "error":
		level = logs.LevelError
	case "warn":
		level = logs.LevelWarn
	case "info":
		level = logs.LevelInfo
	case "debug":
		level = logs.LevelDebug
	case "notice":
		level = logs.LevelNotice
	default:
		ok = false
	}
	return
}

func InitLogger(dir, strLevel string) (err error) {
	logLevel, ok := validLogLevel(strLevel)
	if !ok {
		return fmt.Errorf("mismatch the logLevel %s", strLevel)
	}
	config, err := json.Marshal(LogConfig{
		Filename: path.Join(dir, "debug.log"),
		Rotate:   true,
		Daily:    true,
		Level:    logLevel,
	})
	if err != nil {
		return err
	}
	mlog.SetLogger(logs.AdapterFile, string(config))
	mlog.Debug(string(config))
	return nil
}

func GetLogger() *logs.BeeLogger {
	return mlog
}
