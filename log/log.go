package log

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/util"
	"runtime"
	"fmt"
)

const (
	defaultLogDirname = "logs"

	errModuleNotFound = "specified module not found"
)

func Print(module string, level string, format string, reason ...interface{}) {
	level = strings.ToLower(level)
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
	} else {
		logs.GetLogger()
		logs.Error(errModuleNotFound)
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
	defaultHomeDir := util.AppDataDir("copernicus", false)
	logDir := filepath.Join(defaultHomeDir, defaultLogDirname)

	logConf := struct {
		FileName string `json:"filename"`
		Level    int    `json:"level"`
		Daily    bool   `json:"daily"`
	}{
		FileName: logDir,
		Level:    getLevel(conf.AppConf.LogLevel),
		Daily:    false,
	}

	configuration, err := json.Marshal(logConf)
	if err != nil {
		panic(err)
	}
	logs.SetLogger(logs.AdapterFile, string(configuration))
}

// Emergency logs a message at emergency level.
func Emergency(f interface{}, v ...interface{}) {
	logs.Emergency(f, v)
}

// Alert logs a message at alert level.
func Alert(f interface{}, v ...interface{}) {
	logs.Alert(f, v)
}

// Critical logs a message at critical level.
func Critical(f interface{}, v ...interface{}) {
	logs.Critical(f, v)
}

// Error logs a message at error level.
func Error(f interface{}, v ...interface{}) {
	logs.Error(f, v)
}

// Warning logs a message at warning level.
func Warning(f interface{}, v ...interface{}) {
	logs.Warning(f, v)
}

// Warn compatibility alias for Warning()
func Warn(f interface{}, v ...interface{}) {
	logs.Warn(f, v)
}

// Notice logs a message at notice level.
func Notice(f interface{}, v ...interface{}) {
	logs.Notice(f, v)
}

// Informational logs a message at info level.
func Informational(f interface{}, v ...interface{}) {
	logs.Informational(f, v)
}

// Info compatibility alias for Warning()
func Info(f interface{}, v ...interface{}) {
	logs.Info(f, v)
}

// Debug logs a message at debug level.
func Debug(f interface{}, v ...interface{}) {
	logs.Debug(f, v)
}

// Trace logs a message at trace level.
// compatibility alias for Warning()
func Trace(f interface{}, v ...interface{}) {
	logs.Trace(f, v)
}

func TraceLog() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	_, line := f.FileLine(pc[0])
	return fmt.Sprintf("%s line : %d\n", f.Name(), line)
}

func GetLogger() *logs.BeeLogger {
	return logs.GetBeeLogger()
}