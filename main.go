// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (

	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	// "github.com/btcsuite/btcd/blockchain/indexers"
	// "github.com/btcsuite/btcd/database"
	// "github.com/btcsuite/btcd/limits"

	"context"
	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/connmgr"
	"github.com/btcboost/copernicus/limits"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/addrmgr"
)

const (
	// blockDbNamePrefix is the prefix for the block database name.  The
	// database type is appended to this value to form the full block
	// database name.
	blockDbNamePrefix = "blocks"
)

// winServiceMain is only invoked on Windows.  It detects when btcd is running
// as a service and reacts accordingly.
var winServiceMain func() (bool, error)

// btcdMain is the real main function for btcd.  It is necessary to work around
// the fact that deferred functions do not run when os.Exit() is called.  The
// optional serverChan parameter is mainly used by the service code to be
// notified with the server once it is setup so it can gracefully stop it when
// requested from the service control manager.
func bchMain(ctx context.Context, serverChan chan<- *server) error {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	cfg := conf.Cfg

	// Get a channel that will be closed when a shutdown signal has been
	// triggered either from an OS signal such as SIGINT (Ctrl+C) or from
	// another subsystem such as the RPC server.
	interrupt := interruptListener()

	// Return now if an interrupt signal was triggered.
	if interruptRequested(interrupt) {
		return nil
	}

	// Load the block database.
	// db, err := loadBlockDB()
	// if err != nil {
	// 	btcdLog.Errorf("%v", err)
	// 	return err
	// }
	// defer func() {
	// 	// Ensure the database is sync'd and closed on shutdown.
	// 	btcdLog.Infof("Gracefully shutting down the database...")
	// 	db.Close()
	// }()

	// Return now if an interrupt signal was triggered.
	// if interruptRequested(interrupt) {
	// 	return nil
	// }

	// Drop indexes and exit if requested.
	//
	// NOTE: The order is important here because dropping the tx index also
	// drops the address index since it relies on it.
	// if cfg.DropAddrIndex {
	// 	if err := indexers.DropAddrIndex(db, interrupt); err != nil {
	// 		btcdLog.Errorf("%v", err)
	// 		return err
	// 	}

	// 	return nil
	// }
	// if cfg.DropTxIndex {
	// 	if err := indexers.DropTxIndex(db, interrupt); err != nil {
	// 		btcdLog.Errorf("%v", err)
	// 		return err
	// 	}

	// 	return nil
	// }

	// Create server and start it.
	// server, err := newServer(cfg.P2PNet.ListenerAddrs, db, activeNetParams.Params,
	// 	interrupt)
	// if err != nil {
	// 	// TODO: this logging could do with some beautifying.
	// 	btcdLog.Errorf("Unable to start server on %v: %v",
	// 		cfg.Listeners, err)
	// 	return err
	// }
	// defer func() {
	// 	btcdLog.Infof("Gracefully shutting down the server...")
	// 	server.Stop()
	// 	server.WaitForShutdown()
	// 	srvrLog.Infof("Server shutdown complete")
	// }()
	// server.Start()
	// if serverChan != nil {
	// 	serverChan <- server
	// }

	// FIXME
	s := server{
		chainParams:          chainParams,
		addrManager:          amgr,
		newPeers:             make(chan *serverPeer, cfg.MaxPeers),
		donePeers:            make(chan *serverPeer, cfg.MaxPeers),
		banPeers:             make(chan *serverPeer, cfg.MaxPeers),
		query:                make(chan interface{}),
		relayInv:             make(chan relayMsg, cfg.MaxPeers),
		broadcast:            make(chan broadcastMsg, cfg.MaxPeers),
		quit:                 make(chan struct{}),
		modifyRebroadcastInv: make(chan interface{}),
		peerHeightsUpdate:    make(chan updatePeerHeightsMsg),
		nat:                  nat,
		db:                   db,
		timeSource:           blockchain.NewMedianTime(),
		services:             services,
		sigCache:             txscript.NewSigCache(cfg.SigCacheMaxSize),
		hashCache:            txscript.NewHashCache(cfg.SigCacheMaxSize),
	}

	amgr := addrmgr.New(cfg.DataDir, iplookup)

	cmgr, err := connmgr.New(&connmgr.Config{
		ListenAddr:     cfg.P2PNet.ListenAddrs,
		RetryDuration:  cfg.P2PNet.RetryDuration,
		RetgetOutbound: cfg.P2PNet.TargetOutbound,

		Dial: func(ctx context.Context, netaddr net.Addr) {
			return net.Dialer{}.DialContext(ctx, netaddr.Network(), netaddr.String())
		},
		OnAccept:      s.inboundPeerConnected,
		OnConnect:     s.outboundPeerConnected,
		GetNewAddress: addmgr.GetNewAddress(),
	})
	for _, addr := range cfg.PeersOnStart {
		netAddr, err := addrStringToNetAddr(addr)
		if err != nil {
			return err
		}
		go cmgr.Connect(ctx, netAddr, true)
	}

	// qiw: we must do Start() after connect PeersOnStart
	cmgr.Start(ctx)

	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt
	return nil
}

// removeRegressionDB removes the existing regression test database if running
// in regression test mode and it already exists.
func removeRegressionDB(dbPath string) error {
	// Don't do anything if not in regression test mode.
	if !cfg.RegressionTest {
		return nil
	}

	// Remove the old regression test database if it already exists.
	fi, err := os.Stat(dbPath)
	if err == nil {
		btcdLog.Infof("Removing regression test database from '%s'", dbPath)
		if fi.IsDir() {
			err := os.RemoveAll(dbPath)
			if err != nil {
				return err
			}
		} else {
			err := os.Remove(dbPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// dbPath returns the path to the block database given a database type.
func blockDbPath(dbType string) string {
	// The database name is based on the database type.
	dbName := blockDbNamePrefix + "_" + dbType
	if dbType == "sqlite" {
		dbName = dbName + ".db"
	}
	dbPath := filepath.Join(cfg.DataDir, dbName)
	return dbPath
}

// warnMultipleDBs shows a warning if multiple block database types are detected.
// This is not a situation most users want.  It is handy for development however
// to support multiple side-by-side databases.
func warnMultipleDBs() {
	// This is intentionally not using the known db types which depend
	// on the database types compiled into the binary since we want to
	// detect legacy db types as well.
	dbTypes := []string{"ffldb", "leveldb", "sqlite"}
	duplicateDbPaths := make([]string, 0, len(dbTypes)-1)
	for _, dbType := range dbTypes {
		if dbType == cfg.DbType {
			continue
		}

		// Store db path as a duplicate db if it exists.
		dbPath := blockDbPath(dbType)
		if fileExists(dbPath) {
			duplicateDbPaths = append(duplicateDbPaths, dbPath)
		}
	}

	// Warn if there are extra databases.
	if len(duplicateDbPaths) > 0 {
		selectedDbPath := blockDbPath(cfg.DbType)
		btcdLog.Warnf("WARNING: There are multiple block chain databases "+
			"using different database types.\nYou probably don't "+
			"want to waste disk space by having more than one.\n"+
			"Your current database is located at [%v].\nThe "+
			"additional database is located at %v", selectedDbPath,
			duplicateDbPaths)
	}
}

// loadBlockDB loads (or creates when needed) the block database taking into
// account the selected database backend and returns a handle to it.  It also
// contains additional logic such warning the user if there are multiple
// databases which consume space on the file system and ensuring the regression
// test database is clean when in regression test mode.
func loadBlockDB() (database.DB, error) {
	// The memdb backend does not have a file path associated with it, so
	// handle it uniquely.  We also don't want to worry about the multiple
	// database type warnings when running with the memory database.
	if cfg.DbType == "memdb" {
		btcdLog.Infof("Creating block database in memory.")
		db, err := database.Create(cfg.DbType)
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	warnMultipleDBs()

	// The database name is based on the database type.
	dbPath := blockDbPath(cfg.DbType)

	// The regression test is special in that it needs a clean database for
	// each run, so remove it now if it already exists.
	removeRegressionDB(dbPath)

	btcdLog.Infof("Loading block database from '%s'", dbPath)
	db, err := database.Open(cfg.DbType, dbPath, activeNetParams.Net)
	if err != nil {
		// Return the error if it's not because the database doesn't
		// exist.
		if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode !=
			database.ErrDbDoesNotExist {

			return nil, err
		}

		// Create the db if it does not exist.
		err = os.MkdirAll(cfg.DataDir, 0700)
		if err != nil {
			return nil, err
		}
		db, err = database.Create(cfg.DbType, dbPath, activeNetParams.Net)
		if err != nil {
			return nil, err
		}
	}

	btcdLog.Info("Block database loaded")
	return db, nil
}

func main() {
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Block and transaction processing can cause bursty allocations.  This
	// limits the garbage collector from excessively overallocating during
	// bursts.  This value was arrived at with the help of profiling live
	// usage.
	debug.SetGCPercent(10)

	// Up some limits.
	if err := limits.SetLimits(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set limits: %v\n", err)
		os.Exit(1)
	}

	// Call serviceMain on Windows to handle running as a service.  When
	// the return isService flag is true, exit now since we ran as a
	// service.  Otherwise, just fall through to normal operation.
	if runtime.GOOS == "windows" {
		isService, err := winServiceMain()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if isService {
			os.Exit(0)
		}
	}

	// Work around defer not working after os.Exit()
	if err := bchMain(nil); err != nil {
		os.Exit(1)
	}
}
