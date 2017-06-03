package main

import (
	"os"
	"os/signal"
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
	go func() {
		interruptChannel := make(chan os.Signal, 1)
		signal.Notify(interruptChannel, interruptSignals...)
		// Listen for initial shutdown signal
		select {
		case sig := <-interruptChannel:
			log.Informational("Received signal (%s).  Shutting down...",
				sig)

		case <-shutdownRequestChannel:
			log.Info("Shutdown requested.  Shutting down...")
		}
		close(c)

		// Listen for repeated signals
		for {
			select {
			case sig := <-interruptChannel:
				log.Informational("Received signal (%s).  Already "+
					"shutting down...", sig)

			case <-shutdownRequestChannel:
				log.Info("Shutdown requested.  Already " +
					"shutting down...")
			}
		}
	}()
	return c
}

func interruptRequested(interrupted <-chan struct{}) bool {
	select {
	case <-interrupted:
		return true
	default:
	}
	return false
}
