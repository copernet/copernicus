package main

import (
	_ "copernicus/conf"
	_ "copernicus/log"
	_ "copernicus/storage"
	"os"
	"syscall"

	"github.com/astaxie/beego/logs"
)

var log *logs.BeeLogger

func init() {
	log = logs.NewLogger()
	interruptSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
}

func btcMain() error {
	interruptChan := interruptListener()
	<-interruptChan
	return nil
}

func main() {
	log.Info("application is runing")

	if err := btcMain(); err != nil {
		os.Exit(1)
	}
}
