package main

// please ensure init `github.com/btcboost/copernicus/log` firstly,
// or you will get an error log output.
import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/database"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/net/p2p"
	"github.com/btcboost/copernicus/rpc"
	"github.com/btcboost/copernicus/utils"

	_ "github.com/btcboost/copernicus/log"

	"github.com/astaxie/beego/logs"
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

func setupRPCServer() (*rpc.RPCServer, error) {
	if !conf.Cfg.DisableRPC {
		// Setup listeners for the configured RPC listen addresses and
		// TLS settings.
		rpcListeners, err := setupRPCListeners()
		if err != nil {
			return nil, err
		}
		if len(rpcListeners) == 0 {
			return nil, errors.New("RPCS: No valid listen address")
		}

		rpcServer, err := rpc.NewRPCServer(&rpc.RPCServerConfig{
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

// setupRPCListeners returns a slice of listeners that are configured for use
// with the RPC server depending on the configuration settings for listen
// addresses and TLS.
func setupRPCListeners() ([]net.Listener, error) {
	// Setup TLS if not disabled.
	listenFunc := net.Listen
	// todo open
	if !conf.Cfg.DisableTLS {
		// Generate the TLS cert and key file if both don't already
		// exist.
		if !fileExists(conf.Cfg.RPCKey) && !fileExists(conf.Cfg.RPCCert) {
			err := rpc.GenCertPair(conf.Cfg.RPCCert, conf.Cfg.RPCKey)
			if err != nil {
				return nil, err
			}
		}
		keypair, err := tls.LoadX509KeyPair(conf.Cfg.RPCCert, conf.Cfg.RPCKey)
		if err != nil {
			return nil, err
		}

		tlsConfig := tls.Config{
			Certificates: []tls.Certificate{keypair},
			MinVersion:   tls.VersionTLS12,
		}

		// Change the standard net.Listen function to the tls one.
		listenFunc = func(net string, laddr string) (net.Listener, error) {
			return tls.Listen(net, laddr, &tlsConfig)
		}
	}

	netAddrs, err := parseListeners(conf.Cfg.RPCListeners)
	if err != nil {
		return nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := listenFunc(addr.Network(), addr.String())
		if err != nil {
			logs.Warn("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	return listeners, nil
}

// filesExists reports whether the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// parseListeners determines whether each listen address is IPv4 and IPv6 and
// returns a slice of appropriate net.Addrs to listen on with TCP. It also
// properly detects addresses which apply to "all interfaces" and adds the
// address as both IPv4 and IPv6.
func parseListeners(addrs []string) ([]net.Addr, error) {
	netAddrs := make([]net.Addr, 0, len(addrs)*2)
	for _, addr := range addrs {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			// Shouldn't happen due to already being normalized.
			return nil, err
		}

		// Empty host or host of * on plan9 is both IPv4 and IPv6.
		if host == "" || (host == "*" && runtime.GOOS == "plan9") {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
			continue
		}

		// Strip IPv6 zone id if present since net.ParseIP does not
		// handle it.
		zoneIndex := strings.LastIndex(host, "%")
		if zoneIndex > 0 {
			host = host[:zoneIndex]
		}

		// Parse the IP.
		ip := net.ParseIP(host)
		if ip == nil {
			return nil, fmt.Errorf("'%s' is not a valid IP address", host)
		}

		// To4 returns nil when the IP is not an IPv4 address, so use
		// this determine the address type.
		if ip.To4() == nil {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
		} else {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
		}
	}
	return netAddrs, nil
}

// onionAddr implements the net.Addr interface with two struct fields
type simpleAddr struct {
	net, addr string
}

// String returns the address.
//
// This is part of the net.Addr interface.
func (a simpleAddr) String() string {
	return a.addr
}

// Network returns the network.
//
// This is part of the net.Addr interface.
func (a simpleAddr) Network() string {
	return a.net
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = simpleAddr{}
