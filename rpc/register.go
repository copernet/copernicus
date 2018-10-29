package rpc

import (
	"math/rand"
	"time"
)

type commandHandler func(*Server, interface{}, <-chan struct{}) (interface{}, error)

// rpcHandlers maps RPC command strings to appropriate handler functions.
// This is set by init because help references rpcHandlers and thus causes
// a dependency loop.
var rpcHandlers = map[string]commandHandler{}

func appendCommand(name string, cmd commandHandler) bool {
	if _, ok := rpcHandlers[name]; ok {
		return false
	}
	rpcHandlers[name] = cmd
	return true
}

func registerAllRPCCommands() {
	registerABCRPCCommands()
	registerBlockchainRPCCommands()
	registerMiningRPCCommands()
	registerMiscRPCCommands()
	registerNetRPCCommands()
	registerRawTransactionRPCCommands()
	registerWalletRPCCommands()
}

func init() {
	registerAllRPCCommands()
	rand.Seed(time.Now().UnixNano())
}
