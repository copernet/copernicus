package rpc

import "github.com/btcsuite/btcd/btcjson"

// API version constants
const (
	jsonrpcSemverString = "1.3.0"
	jsonrpcSemverMajor  = 1
	jsonrpcSemverMinor  = 3
	jsonrpcSemverPatch  = 0
)

// Commands that are available to a limited user
var rpcLimited = map[string]struct{}{
	// Websockets commands
	"loadtxfilter":          {},
	"notifyblocks":          {},
	"notifynewtransactions": {},
	"notifyreceived":        {},
	"notifyspent":           {},
	"rescan":                {},
	"rescanblocks":          {},
	"session":               {},

	// Websockets AND HTTP/S commands
	"help": {},

	// HTTP/S-only commands
	"createrawtransaction":  {},
	"decoderawtransaction":  {},
	"decodescript":          {},
	"getbestblock":          {},
	"getbestblockhash":      {},
	"getblock":              {},
	"getblockcount":         {},
	"getblockhash":          {},
	"getblockheader":        {},
	"getcurrentnet":         {},
	"getdifficulty":         {},
	"getheaders":            {},
	"getinfo":               {},
	"getnettotals":          {},
	"getnetworkhashps":      {},
	"getrawmempool":         {},
	"getrawtransaction":     {},
	"gettxout":              {},
	"searchrawtransactions": {},
	"sendrawtransaction":    {},
	"submitblock":           {},
	"uptime":                {},
	"validateaddress":       {},
	"verifymessage":         {},
	"version":               {},
}

type commandHandler func(*RRCServer, interface{}, <-chan struct{}) (interface{}, error)

// rpcHandlers maps RPC command strings to appropriate handler functions.
// This is set by init because help references rpcHandlers and thus causes
// a dependency loop.
var rpcHandlers map[string]commandHandler

var rpcHandlersBeforeInit = map[string]commandHandler{
	//"addnode":               handleAddNode,
	//"createrawtransaction":  handleCreateRawTransaction,
	//"debuglevel":            handleDebugLevel,
	//"decoderawtransaction":  handleDecodeRawTransaction,
	//"decodescript":          handleDecodeScript,
	//"generate":              handleGenerate,
	//"getaddednodeinfo":      handleGetAddedNodeInfo,
	//"getbestblock":          handleGetBestBlock,
	//"getbestblockhash":      handleGetBestBlockHash,
	//"getblock":              handleGetBlock,
	//"getblockchaininfo":     handleGetBlockChainInfo,
	//"getblockcount":         handleGetBlockCount,
	//"getblockhash":          handleGetBlockHash,
	//"getblockheader":        handleGetBlockHeader,
	//"getblocktemplate":      handleGetBlockTemplate,
	//"getconnectioncount":    handleGetConnectionCount,
	//"getcurrentnet":         handleGetCurrentNet,
	//"getdifficulty":         handleGetDifficulty,
	//"getgenerate":           handleGetGenerate,
	//"gethashespersec":       handleGetHashesPerSec,
	//"getheaders":            handleGetHeaders,
	//"getinfo":               handleGetInfo,
	//"getmempoolinfo":        handleGetMempoolInfo,
	//"getmininginfo":         handleGetMiningInfo,
	//"getnettotals":          handleGetNetTotals,
	//"getnetworkhashps":      handleGetNetworkHashPS,
	//"getpeerinfo":           handleGetPeerInfo,
	//"getrawmempool":         handleGetRawMempool,
	//"getrawtransaction":     handleGetRawTransaction,
	//"gettxout":              handleGetTxOut,
	//"help":                  handleHelp,
	//"node":                  handleNode,
	//"ping":                  handlePing,
	//"searchrawtransactions": handleSearchRawTransactions,
	//"sendrawtransaction":    handleSendRawTransaction,
	//"setgenerate":           handleSetGenerate,
	//"stop":                  handleStop,
	//"submitblock":           handleSubmitBlock,
	//"uptime":                handleUptime,
	//"validateaddress":       handleValidateAddress,
	//"verifychain":           handleVerifyChain,
	//"verifymessage":         handleVerifyMessage,
	"version": handleVersion,
}

// handleVersion implements the version command.
//
// NOTE: This is a btcsuite extension ported from github.com/decred/dcrd.
func handleVersion(s *RRCServer, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	result := map[string]btcjson.VersionResult{
		"btcdjsonrpcapi": {
			VersionString: jsonrpcSemverString,
			Major:         jsonrpcSemverMajor,
			Minor:         jsonrpcSemverMinor,
			Patch:         jsonrpcSemverPatch,
		},
	}
	return result, nil
}
