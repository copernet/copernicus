package log

import (
	"encoding/json"
	"fmt"
	"path"
	"runtime"
	"strings"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/conf"
)

func init() {
	InitLogger(conf.GetDataPath(), "debug") // todo configure via config file
}

type logConfig struct {
	Filename string `json:"filename"`
	Level    int    `json:"level,omitempty"`
	Rotate   bool   `json:"rotate,omitempty"`
	Daily    bool   `json:"daily,omitempty"`
	MaxDays  int64  `json:"maxdays,omitempty"`
	MaxLines int    `json:"maxlines,omitempty"`
	MaxSize  int    `json:"maxsize,omitempty"`
}

func validLogLevel(strLevel string) (level int, ok bool) {
	ok = true
	strLevel = strings.ToLower(strLevel)
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

func TraceLog() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	_, line := f.FileLine(pc[0])
	return fmt.Sprintf("%s line : %d\n", f.Name(), line)
}

func Print(module string, level string, format string, reason ...interface{}) {
	if IsIncludeModule(module) {
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

func IsIncludeModule(module string) bool {
	for _, item := range conf.AppConf.LogModule {
		if item == module {
			return true
		}
	}
	return false
}

func InitLogger(dir, strLevel string) (err error) {
	logLevel, ok := validLogLevel(strLevel)
	if !ok {
		return fmt.Errorf("mismatch the logLevel %s", strLevel)
	}
	config, err := json.Marshal(logConfig{
		Filename: path.Join(dir, "debug.logger"),
		Rotate:   true,
		Daily:    true,
		Level:    logLevel,
	})
	if err != nil {
		return err
	}
	logs.SetLogger(logs.AdapterFile, string(config))
	logs.Debug(string(config))
	return nil
}

type Closure func() string

func (c Closure) ToString() string {
	return c()
}
func InitLogClosure(c func() string) Closure {
	return Closure(c)
}
