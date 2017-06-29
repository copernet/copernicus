package main

import (
	"github.com/astaxie/beego/logs"
	"github.com/btccom/copernicus/conf"
	"github.com/btccom/copernicus/msg"
	"github.com/btccom/copernicus/peer"
	"os"
	"syscall"
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
	startBitcoin()

	if err := btcMain(); err != nil {
		os.Exit(1)
	}
}

func startBitcoin() error {

	peerManager, err := peer.NewPeerManager(conf.AppConf.Listeners, nil, msg.ActiveNetParams)
	if err != nil {
		log.Error("unable to start server on %v:%v", conf.AppConf.Listeners, err)
		return err
	}
	defer func() {
		log.Info("gracefully shtting down the server ....")
		peerManager.Stop()
		peerManager.WaitForShutdown()
		log.Info("server shtdown compltete")
	}()
	peerManager.Start()
	return nil
}
