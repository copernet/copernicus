package conf

import (
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

}
