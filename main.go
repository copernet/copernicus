package main

import (
	_ "btcboost/log"
	"github.com/astaxie/beego/logs"
)

func main() {
	log := logs.NewLogger()
	log.Info("application is runing")

}