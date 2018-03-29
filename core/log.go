package core

import "github.com/btcsuite/btclog"

var log btclog.Logger

func init() {
	DisableLog()
}

func DisableLog() {
	log = btclog.Disabled
}

func UseLogger(logger btclog.Logger) {
	log = logger
}
