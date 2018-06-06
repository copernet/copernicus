// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/copernet/copernicus/log"
	"os"
	"os/signal"
	"sync"
)

// shutdownRequestChannel is used to initiate shutdown from one of the
// subsystems using the same code paths as when an interrupt signal is received.
var shutdownRequestChannel = make(chan struct{})

// interruptSignals defines the default signals to catch in order to do a proper
// shutdown.  This may be modified during init depending on the platform.
var interruptSignals = []os.Signal{os.Interrupt}

// interruptListener listens for OS Signals such as SIGINT (Ctrl+C) and shutdown
// requests from shutdownRequestChannel.  It returns a channel that is closed
// when either signal is received.
func interruptListener() <-chan struct{} {
	c := make(chan struct{})
	closeOnce := sync.Once{}
	go func() {
		interruptChannel := make(chan os.Signal, 1)
		signal.Notify(interruptChannel, interruptSignals...)

		for {
			select {
			case sig := <-interruptChannel:
				log.Info("Received signal (%s).  "+
					"shutting down...", sig)

			case <-shutdownRequestChannel:
				log.Info("Shutdown requested.  " +
					"shutting down...")
			}

			closeOnce.Do(
				func() {
					close(c)
				})
		}
	}()

	return c
}

// interruptRequested returns true when the channel returned by
// interruptListener was closed.  This simplifies early shutdown slightly since
// the caller can just use an if statement instead of a select.
func interruptRequested(interrupted <-chan struct{}) bool {
	select {
	case <-interrupted:
		return true
	default:
	}

	return false
}
