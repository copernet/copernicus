package log

import (
	"strings"

	"fmt"
	"github.com/astaxie/beego/logs"
	"github.com/copernet/copernicus/conf"
)

const (
	DefaultLogDirname = "logs"
	errModuleNotFound = "specified module not found"
)

var mapModule map[string]struct{}

func Print(module string, level string, format string, reason ...interface{}) {
	level = strings.ToLower(level)
	if isIncludeModule(module) {
		format = fmt.Sprintf("module[%s]: level[%s]", module, level)
		switch level {
		case "emergency":
			logs.Emergency(format, reason...)
		case "alert":
			logs.Alert(format, reason...)
		case "critical":
			logs.Critical(format, reason...)
		case "error":
			logs.Error(format, reason...)
		case "warn":
			logs.Warn(format, reason...)
		case "info":
			logs.Info(format, reason...)
		case "debug":
			logs.Debug(format, reason...)
		case "notice":
			logs.Notice(format, reason...)
		}
	} else {
		logs.GetLogger()
		logs.Debug("module(%s): %v", module, errModuleNotFound)
	}
}

func isIncludeModule(module string) bool {
	module = strings.ToLower(module)
	if _, ok := mapModule[module]; ok {
		return true
	}
	return false
}

// Emergency logs a message at emergency level.
func Emergency(f interface{}, v ...interface{}) {
	logs.Emergency(f, v...)
}

// Alert logs a message at alert level.
func Alert(f interface{}, v ...interface{}) {
	logs.Alert(f, v...)
}

// Critical logs a message at critical level.
func Critical(f interface{}, v ...interface{}) {
	logs.Critical(f, v...)
}

// Error logs a message at error level.
func Error(f interface{}, v ...interface{}) {
	logs.Error(f, v...)
}

// Warning logs a message at warning level.
func Warning(f interface{}, v ...interface{}) {
	logs.Warning(f, v...)
}

// Warn compatibility alias for Warning()
func Warn(f interface{}, v ...interface{}) {
	logs.Warn(f, v...)
}

// Notice logs a message at notice level.
func Notice(f interface{}, v ...interface{}) {
	logs.Notice(f, v...)
}

// Informational logs a message at info level.
func Informational(f interface{}, v ...interface{}) {
	logs.Informational(f, v...)
}

// Info compatibility alias for Warning()
func Info(f interface{}, v ...interface{}) {
	logs.Info(f, v...)
}

// Debug logs a message at debug level.
func Debug(f interface{}, v ...interface{}) {
	logs.Debug(f, v...)
}

// Trace logs a message at trace level.
// compatibility alias for Warning()
func Trace(f interface{}, v ...interface{}) {
	logs.Trace(f, v...)
}

func GetLogger() *logs.BeeLogger {
	return logs.GetBeeLogger()
}

func Init(logConf string) {
	logs.SetLogger(logs.AdapterFile, logConf)

	// output filename and line number
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(4)
	// output async buffer
	// logs.Async(1e3)

	// init mapModule
	mapModule = make(map[string]struct{})
	for _, module := range conf.Cfg.Log.Module {
		module = strings.ToLower(module)
		mapModule[module] = struct{}{}
	}
}
