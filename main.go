package main

// please ensure init `github.com/btcboost/copernicus/log` firstly,
// or you will get an error log output.
import (
	"fmt"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/net/p2p"
	"github.com/btcboost/copernicus/utils"
	"os"
	"syscall"

	_ "github.com/btcboost/copernicus/log"

	"github.com/astaxie/beego/logs"
	"copernicus/utxo"
)

func init() {
	interruptSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
}

func btcMain() error {
	interruptChan := interruptListener()
	<-interruptChan
	return nil
}

func main() {
	logs.Info("application is running")
	startBitcoin()
	if err := btcMain(); err != nil {
		os.Exit(1)
	}
}

func startBitcoin() error {
	path := conf.AppConf.DataDir + "/peer"
	exists := utils.PathExists(path)
	if !exists {
		utils.MakePath(path)
	}
	db, err := database.NewDBWrapper(&database.DBOption{
		FilePath:      path,
		CacheSize:     1 << 20,
		DontObfuscate: false,
	})
	if err != nil {
		fmt.Println("InitDB:", err.Error())
		return err
	}
	defer db.Close()
	fmt.Println("InitDB finish")
	utxo.InitUtxo(&utxo.UtxoConfig{})
	peerManager, err := p2p.NewPeerManager(conf.AppConf.Listeners, *db, msg.ActiveNetParams)
	if err != nil {
		fmt.Printf("unable to start server on %v:%v \n", conf.AppConf.Listeners, err)
		return err
	}
	fmt.Println("PeerManager Init")
	//defer func() {
	//	fmt.Println("gracefully shutting down the server ....")
	//	peerManager.Stop()
	//	peerManager.WaitForShutdown()
	//	fmt.Println("server shutdown complete")
	//}()

	peerManager.Start()

	return nil
}
