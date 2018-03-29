package p2p

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

type LogClosure func() string

func (c LogClosure) ToString() string {
	return c()
}
func InitLogClosure(c func() string) LogClosure {
	return LogClosure(c)
}
