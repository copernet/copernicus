package conf

import (
	mlog "copernicus/log"

	"github.com/astaxie/beego/config"
	"github.com/astaxie/beego/logs"
)

var appConf config.Configer

func init() {
	log := logs.NewLogger()
	appConf, err := config.NewConfig("ini", "init.conf")
	if err != nil {
		log.Error(err.Error())
	}
	contentTimeout := appConf.String("Timeout::connectTimeout")
	log.Info("read conf timeout is  %s", contentTimeout)
	logDir := appConf.String("Log::dir")
	log.Info("log dir is %s", logDir)
	logLevel := appConf.String("Log::level")
	log.Info("log dir is %s", logLevel)
	if err := mlog.InitLogger(logDir, logLevel); err != nil {
		log.Error(err.Error())
	}
}
