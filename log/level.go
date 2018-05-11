package log

import (
	"strings"

	"github.com/astaxie/beego/logs"
)

const defaultLogLevel = logs.LevelDebug

var levelMap = map[string]int{
	"emergency":     logs.LevelEmergency,
	"alert":         logs.LevelAlert,
	"critical":      logs.LevelCritical,
	"error":         logs.LevelError,
	"warning":       logs.LevelWarning,
	"notice":        logs.LevelNotice,
	"informational": logs.LevelInformational,
	"debug":         logs.LevelDebug,
}

func getLevel(level string) int {
	level = strings.ToLower(level)
	ele, ok := levelMap[level]
	if !ok {
		return defaultLogLevel
	}
	return ele
}
