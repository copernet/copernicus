package main

// please ensure init `github.com/btcboost/copernicus/log` firstly,
// or you will get an error log output.
import (
	"fmt"
	"os"
	"syscall"
	"errors"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/net/p2p"
	"github.com/btcboost/copernicus/utils"

	_ "github.com/btcboost/copernicus/log"
	"github.com/astaxie/beego/logs"
	"github.com/btcboost/copernicus/rpc/rpcserver"
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
	rpcServer, err := setupRPCServer()
	if err != nil {
		panic(err)
	}
	rpcServer.Start()
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

func setupRPCServer() (*rpcserver.RPCServer, error) {
	if !conf.CFG.DisableRPC {
		// Setup listeners for the configured RPC listen addresses and
		// TLS settings.
		rpcListeners, err := rpcserver.SetupRPCListeners()
		if err != nil {
			return nil, err
		}
		if len(rpcListeners) == 0 {
			return nil, errors.New("RPCS: No valid listen address")
		}

		rpcServer, err := rpcserver.NewRPCServer(&rpcserver.RPCServerConfig{
			Listeners: rpcListeners,
			// todo open
			//StartupTime: s.startupTime,
			//ConnMgr: &rpcConnManager{&s},
			//SyncMgr:     &rpcSyncMgr{&s, s.syncManager},
			//TimeSource:  s.timeSource,
			//Chain:       s.chain,
			//ChainParams: chainParams,
			//DB:          db,
			//TxMemPool:   s.txMemPool,
			//Generator:   blockTemplateGenerator,
			//CPUMiner:    s.cpuMiner,
			//TxIndex:     s.txIndex,
			//AddrIndex:   s.addrIndex,
		})
		if err != nil {
			return nil, err
		}

		// Signal process shutdown when the RPC server requests it.
		go func() {
			<-rpcServer.RequestedProcessShutdown()
			shutdownRequestChannel <- struct{}{}
		}()
		return rpcServer, nil
	}
	return nil, nil
}





