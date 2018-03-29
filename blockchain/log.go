package blockchain

import (
	"fmt"
	"runtime"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcsuite/btclog"
)

var log btclog.Logger

func init() {
	DisableLog()
}

func DisableLog() {
	log = btclog.Disabled
}

func UseLogger(logger btclog.Logger) {
	log = logger
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
		case "critical":
			log.Critical(format, reason)
		case "error":
			log.Error(format, reason)
		case "warn":
			log.Warn(format, reason)
		case "info":
			log.Info(format, reason)
		case "debug":
			log.Debug(format, reason)
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
