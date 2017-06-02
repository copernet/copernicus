package log

import (
	"github.com/astaxie/beego/logs"
)
//todo config color of debug log
//todo output log to elasticSearch
func init() {
	log := logs.NewLogger()
	log.SetLogger(logs.AdapterConsole)
	log.SetLogger(logs.AdapterFile, `{"filename":"debug.log","maxdays":"3650"}`)
	log.EnableFuncCallDepth(true)
	logs.Async()

}



