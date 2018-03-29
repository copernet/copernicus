package logger

import (
	"encoding/json"
	"fmt"
	"path"
	"runtime"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/conf"
)

var mlog *logs.BeeLogger

// todo config color of debug logger
// todo output logger to elasticSearch

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

func validLogLevel(strLevel string) (level int, ok bool) {
	ok = true
	switch strLevel {
	case "emergency":
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
		Filename: path.Join(dir, "debug.logger"),
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

func ErrorLog(reason string, v ...interface{}) bool {
	mlog.Error(reason, v)
	return false
}

func TraceLog() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	_, line := f.FileLine(pc[0])
	return fmt.Sprintf("%s line : %d\n", f.Name(), line)
}

func LogPrint(module string, level string, format string, reason ...interface{}) {
	if IsIncludeModule(module) {
		switch level {
		case "emergecy":
			mlog.Emergency(format, reason)
		case "alert":
			mlog.Alert(format, reason)
		case "critical":
			mlog.Critical(format, reason)
		case "error":
			mlog.Error(format, reason)
		case "warn":
			mlog.Warn(format, reason)
		case "info":
			mlog.Info(format, reason)
		case "debug":
			mlog.Debug(format, reason)
		case "notice":
			mlog.Notice(format, reason)
		}
	}
}

func IsIncludeModule(module string) bool {
	for _, item := range conf.AppConf.LogModule {
		if item == module {
			return true
		}
	}
	return false
}
