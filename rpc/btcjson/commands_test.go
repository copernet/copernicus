// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

// TestChainSvrCmds tests all of the chain server commands marshal and unmarshal
// into valid results include handling of optional fields being omitted in the
// marshalled command, while optional fields with defaults have the default
// assigned on unmarshalled commands.
func TestChainSvrCmds(t *testing.T) {
	t.Parallel()

	testID := int(1)
	tests := []struct {
		name         string
		newCmd       func() (interface{}, error)
		staticCmd    func() interface{}
		marshalled   string
		unmarshalled interface{}
	}{
		{
			name: "addnode",
			newCmd: func() (interface{}, error) {
				return NewCmd("addnode", "127.0.0.1", ANRemove)
			},
			staticCmd: func() interface{} {
				return NewAddNodeCmd("127.0.0.1", ANRemove)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"addnode","params":["127.0.0.1","remove"],"id":1}`,
			unmarshalled: &AddNodeCmd{Addr: "127.0.0.1", SubCmd: ANRemove},
		},
		{
			name: "createrawtransaction",
			newCmd: func() (interface{}, error) {
				return NewCmd("createrawtransaction", `[{"txid":"123","vout":1}]`,
					`{"456":0.0123}`)
			},
			staticCmd: func() interface{} {
				txInputs := []TransactionInput{
					{Txid: "123", Vout: 1},
				}
				amounts := map[string]AmountType{"456": .0123}
				return NewCreateRawTransactionCmd(txInputs, amounts, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"createrawtransaction","params":[[{"txid":"123","vout":1,"sequence":null}],{"456":0.0123}],"id":1}`,
			unmarshalled: &CreateRawTransactionCmd{
				Inputs:  []TransactionInput{{Txid: "123", Vout: 1}},
				Outputs: map[string]AmountType{"456": .0123},
			},
		},
		{
			name: "createrawtransaction optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("createrawtransaction", `[{"txid":"123","vout":1,"sequence":1}]`,
					`{"456":0.0123}`, int64(12312333333))
			},
			staticCmd: func() interface{} {
				txInputs := []TransactionInput{
					{Txid: "123", Vout: 1, Sequence: Int64(1)},
				}
				amounts := map[string]AmountType{"456": .0123}
				return NewCreateRawTransactionCmd(txInputs, amounts, Int64(12312333333))
			},
			marshalled: `{"jsonrpc":"1.0","method":"createrawtransaction","params":[[{"txid":"123","vout":1,"sequence":1}],{"456":0.0123},12312333333],"id":1}`,
			unmarshalled: &CreateRawTransactionCmd{
				Inputs:   []TransactionInput{{Txid: "123", Vout: 1, Sequence: Int64(1)}},
				Outputs:  map[string]AmountType{"456": .0123},
				LockTime: Int64(12312333333),
			},
		},

		{
			name: "decoderawtransaction",
			newCmd: func() (interface{}, error) {
				return NewCmd("decoderawtransaction", "123")
			},
			staticCmd: func() interface{} {
				return NewDecodeRawTransactionCmd("123")
			},
			marshalled:   `{"jsonrpc":"1.0","method":"decoderawtransaction","params":["123"],"id":1}`,
			unmarshalled: &DecodeRawTransactionCmd{HexTx: "123"},
		},
		{
			name: "decodescript",
			newCmd: func() (interface{}, error) {
				return NewCmd("decodescript", "00")
			},
			staticCmd: func() interface{} {
				return NewDecodeScriptCmd("00")
			},
			marshalled:   `{"jsonrpc":"1.0","method":"decodescript","params":["00"],"id":1}`,
			unmarshalled: &DecodeScriptCmd{HexScript: "00"},
		},
		{
			name: "getaddednodeinfo",
			newCmd: func() (interface{}, error) {
				return NewCmd("getaddednodeinfo")
			},
			staticCmd: func() interface{} {
				return NewGetAddedNodeInfoCmd(nil)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getaddednodeinfo","params":[],"id":1}`,
			unmarshalled: &GetAddedNodeInfoCmd{Node: nil},
		},
		{
			name: "getaddednodeinfo optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("getaddednodeinfo", "127.0.0.1")
			},
			staticCmd: func() interface{} {
				return NewGetAddedNodeInfoCmd(String("127.0.0.1"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getaddednodeinfo","params":["127.0.0.1"],"id":1}`,
			unmarshalled: &GetAddedNodeInfoCmd{
				Node: String("127.0.0.1"),
			},
		},
		{
			name: "getbestblockhash",
			newCmd: func() (interface{}, error) {
				return NewCmd("getbestblockhash")
			},
			staticCmd: func() interface{} {
				return NewGetBestBlockHashCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getbestblockhash","params":[],"id":1}`,
			unmarshalled: &GetBestBlockHashCmd{},
		},
		{
			name: "getblock",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblock", "123")
			},
			staticCmd: func() interface{} {
				return NewGetBlockCmd("123", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblock","params":["123"],"id":1}`,
			unmarshalled: &GetBlockCmd{
				Hash:    "123",
				Verbose: Bool(true),
				//VerboseTx: Bool(false),
			},
		},
		{
			name: "getblock required optional1",
			newCmd: func() (interface{}, error) {
				// Intentionally use a source param that is
				// more pointers than the destination to
				// exercise that path.
				verbosePtr := Bool(true)
				return NewCmd("getblock", "123", &verbosePtr)
			},
			staticCmd: func() interface{} {
				return NewGetBlockCmd("123", Bool(true))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblock","params":["123",true],"id":1}`,
			unmarshalled: &GetBlockCmd{
				Hash:    "123",
				Verbose: Bool(true),
				//VerboseTx: Bool(false),
			},
		},
		{
			name: "getblock required optional2",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblock", "123", true)
			},
			staticCmd: func() interface{} {
				return NewGetBlockCmd("123", Bool(true))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblock","params":["123",true],"id":1}`,
			unmarshalled: &GetBlockCmd{
				Hash:    "123",
				Verbose: Bool(true),
				//VerboseTx: Bool(true),
			},
		},
		{
			name: "getblockchaininfo",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblockchaininfo")
			},
			staticCmd: func() interface{} {
				return NewGetBlockChainInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getblockchaininfo","params":[],"id":1}`,
			unmarshalled: &GetBlockChainInfoCmd{},
		},
		{
			name: "getblockcount",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblockcount")
			},
			staticCmd: func() interface{} {
				return NewGetBlockCountCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getblockcount","params":[],"id":1}`,
			unmarshalled: &GetBlockCountCmd{},
		},
		{
			name: "getblockhash",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblockhash", 123)
			},
			staticCmd: func() interface{} {
				return NewGetBlockHashCmd(123)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getblockhash","params":[123],"id":1}`,
			unmarshalled: &GetBlockHashCmd{Height: 123},
		},
		{
			name: "getblockheader",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblockheader", "123")
			},
			staticCmd: func() interface{} {
				return NewGetBlockHeaderCmd("123", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblockheader","params":["123"],"id":1}`,
			unmarshalled: &GetBlockHeaderCmd{
				Hash:    "123",
				Verbose: Bool(true),
			},
		},
		{
			name: "getblocktemplate",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblocktemplate")
			},
			staticCmd: func() interface{} {
				return NewGetBlockTemplateCmd(nil)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getblocktemplate","params":[],"id":1}`,
			unmarshalled: &GetBlockTemplateCmd{Request: nil},
		},
		{
			name: "getblocktemplate optional - template request",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"]}`)
			},
			staticCmd: func() interface{} {
				template := TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
				}
				return NewGetBlockTemplateCmd(&template)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"]}],"id":1}`,
			unmarshalled: &GetBlockTemplateCmd{
				Request: &TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
				},
			},
		},
		{
			name: "getblocktemplate optional - template request with tweaks",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":500,"sizelimit":100000000,"maxversion":2}`)
			},
			staticCmd: func() interface{} {
				template := TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   500,
					SizeLimit:    100000000,
					MaxVersion:   2,
				}
				return NewGetBlockTemplateCmd(&template)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":500,"sizelimit":100000000,"maxversion":2}],"id":1}`,
			unmarshalled: &GetBlockTemplateCmd{
				Request: &TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   int64(500),
					SizeLimit:    int64(100000000),
					MaxVersion:   2,
				},
			},
		},
		{
			name: "getblocktemplate optional - template request with tweaks 2",
			newCmd: func() (interface{}, error) {
				return NewCmd("getblocktemplate", `{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":true,"sizelimit":100000000,"maxversion":2}`)
			},
			staticCmd: func() interface{} {
				template := TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   true,
					SizeLimit:    100000000,
					MaxVersion:   2,
				}
				return NewGetBlockTemplateCmd(&template)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getblocktemplate","params":[{"mode":"template","capabilities":["longpoll","coinbasetxn"],"sigoplimit":true,"sizelimit":100000000,"maxversion":2}],"id":1}`,
			unmarshalled: &GetBlockTemplateCmd{
				Request: &TemplateRequest{
					Mode:         "template",
					Capabilities: []string{"longpoll", "coinbasetxn"},
					SigOpLimit:   true,
					SizeLimit:    int64(100000000),
					MaxVersion:   2,
				},
			},
		},

		{
			name: "getchaintips",
			newCmd: func() (interface{}, error) {
				return NewCmd("getchaintips")
			},
			staticCmd: func() interface{} {
				return NewGetChainTipsCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getchaintips","params":[],"id":1}`,
			unmarshalled: &GetChainTipsCmd{},
		},
		{
			name: "getconnectioncount",
			newCmd: func() (interface{}, error) {
				return NewCmd("getconnectioncount")
			},
			staticCmd: func() interface{} {
				return NewGetConnectionCountCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getconnectioncount","params":[],"id":1}`,
			unmarshalled: &GetConnectionCountCmd{},
		},
		{
			name: "getdifficulty",
			newCmd: func() (interface{}, error) {
				return NewCmd("getdifficulty")
			},
			staticCmd: func() interface{} {
				return NewGetDifficultyCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getdifficulty","params":[],"id":1}`,
			unmarshalled: &GetDifficultyCmd{},
		},
		{
			name: "getgenerate",
			newCmd: func() (interface{}, error) {
				return NewCmd("getgenerate")
			},
			staticCmd: func() interface{} {
				return NewGetGenerateCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getgenerate","params":[],"id":1}`,
			unmarshalled: &GetGenerateCmd{},
		},
		{
			name: "gethashespersec",
			newCmd: func() (interface{}, error) {
				return NewCmd("gethashespersec")
			},
			staticCmd: func() interface{} {
				return NewGetHashesPerSecCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"gethashespersec","params":[],"id":1}`,
			unmarshalled: &GetHashesPerSecCmd{},
		},
		{
			name: "getinfo",
			newCmd: func() (interface{}, error) {
				return NewCmd("getinfo")
			},
			staticCmd: func() interface{} {
				return NewGetInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getinfo","params":[],"id":1}`,
			unmarshalled: &GetInfoCmd{},
		},
		{
			name: "getmempoolentry",
			newCmd: func() (interface{}, error) {
				return NewCmd("getmempoolentry", "txhash")
			},
			staticCmd: func() interface{} {
				return NewGetMempoolEntryCmd("txhash")
			},
			marshalled: `{"jsonrpc":"1.0","method":"getmempoolentry","params":["txhash"],"id":1}`,
			unmarshalled: &GetMempoolEntryCmd{
				TxID: "txhash",
			},
		},
		{
			name: "getmempoolinfo",
			newCmd: func() (interface{}, error) {
				return NewCmd("getmempoolinfo")
			},
			staticCmd: func() interface{} {
				return NewGetMempoolInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getmempoolinfo","params":[],"id":1}`,
			unmarshalled: &GetMempoolInfoCmd{},
		},
		{
			name: "getmininginfo",
			newCmd: func() (interface{}, error) {
				return NewCmd("getmininginfo")
			},
			staticCmd: func() interface{} {
				return NewGetMiningInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getmininginfo","params":[],"id":1}`,
			unmarshalled: &GetMiningInfoCmd{},
		},
		{
			name: "getnetworkinfo",
			newCmd: func() (interface{}, error) {
				return NewCmd("getnetworkinfo")
			},
			staticCmd: func() interface{} {
				return NewGetNetworkInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getnetworkinfo","params":[],"id":1}`,
			unmarshalled: &GetNetworkInfoCmd{},
		},
		{
			name: "getnettotals",
			newCmd: func() (interface{}, error) {
				return NewCmd("getnettotals")
			},
			staticCmd: func() interface{} {
				return NewGetNetTotalsCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getnettotals","params":[],"id":1}`,
			unmarshalled: &GetNetTotalsCmd{},
		},
		{
			name: "getnetworkhashps",
			newCmd: func() (interface{}, error) {
				return NewCmd("getnetworkhashps")
			},
			staticCmd: func() interface{} {
				return NewGetNetworkHashPSCmd(nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[],"id":1}`,
			unmarshalled: &GetNetworkHashPSCmd{
				Blocks: Int32(120),
				Height: Int32(-1),
			},
		},
		{
			name: "getnetworkhashps optional1",
			newCmd: func() (interface{}, error) {
				return NewCmd("getnetworkhashps", 200)
			},
			staticCmd: func() interface{} {
				return NewGetNetworkHashPSCmd(Int32(200), nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[200],"id":1}`,
			unmarshalled: &GetNetworkHashPSCmd{
				Blocks: Int32(200),
				Height: Int32(-1),
			},
		},
		{
			name: "getnetworkhashps optional2",
			newCmd: func() (interface{}, error) {
				return NewCmd("getnetworkhashps", 200, 123)
			},
			staticCmd: func() interface{} {
				return NewGetNetworkHashPSCmd(Int32(200), Int32(123))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getnetworkhashps","params":[200,123],"id":1}`,
			unmarshalled: &GetNetworkHashPSCmd{
				Blocks: Int32(200),
				Height: Int32(123),
			},
		},
		{
			name: "getpeerinfo",
			newCmd: func() (interface{}, error) {
				return NewCmd("getpeerinfo")
			},
			staticCmd: func() interface{} {
				return NewGetPeerInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getpeerinfo","params":[],"id":1}`,
			unmarshalled: &GetPeerInfoCmd{},
		},
		{
			name: "getrawmempool",
			newCmd: func() (interface{}, error) {
				return NewCmd("getrawmempool")
			},
			staticCmd: func() interface{} {
				return NewGetRawMempoolCmd(nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getrawmempool","params":[],"id":1}`,
			unmarshalled: &GetRawMempoolCmd{
				Verbose: Bool(false),
			},
		},
		{
			name: "getrawmempool optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("getrawmempool", false)
			},
			staticCmd: func() interface{} {
				return NewGetRawMempoolCmd(Bool(false))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getrawmempool","params":[false],"id":1}`,
			unmarshalled: &GetRawMempoolCmd{
				Verbose: Bool(false),
			},
		},
		{
			name: "getrawtransaction",
			newCmd: func() (interface{}, error) {
				return NewCmd("getrawtransaction", "123")
			},
			staticCmd: func() interface{} {
				return NewGetRawTransactionCmd("123", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getrawtransaction","params":["123"],"id":1}`,
			unmarshalled: &GetRawTransactionCmd{
				Txid:    "123",
				Verbose: Bool(false),
			},
		},
		{
			name: "getrawtransaction optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("getrawtransaction", "123", true)
			},
			staticCmd: func() interface{} {
				return NewGetRawTransactionCmd("123", Bool(true))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getrawtransaction","params":["123",true],"id":1}`,
			unmarshalled: &GetRawTransactionCmd{
				Txid:    "123",
				Verbose: Bool(true),
			},
		},
		{
			name: "gettxout",
			newCmd: func() (interface{}, error) {
				return NewCmd("gettxout", "123", 1)
			},
			staticCmd: func() interface{} {
				return NewGetTxOutCmd("123", 1, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettxout","params":["123",1],"id":1}`,
			unmarshalled: &GetTxOutCmd{
				Txid:           "123",
				Vout:           1,
				IncludeMempool: Bool(true),
			},
		},
		{
			name: "gettxout optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("gettxout", "123", 1, true)
			},
			staticCmd: func() interface{} {
				return NewGetTxOutCmd("123", 1, Bool(true))
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettxout","params":["123",1,true],"id":1}`,
			unmarshalled: &GetTxOutCmd{
				Txid:           "123",
				Vout:           1,
				IncludeMempool: Bool(true),
			},
		},
		{
			name: "gettxoutproof",
			newCmd: func() (interface{}, error) {
				return NewCmd("gettxoutproof", []string{"123", "456"})
			},
			staticCmd: func() interface{} {
				return NewGetTxOutProofCmd([]string{"123", "456"}, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettxoutproof","params":[["123","456"]],"id":1}`,
			unmarshalled: &GetTxOutProofCmd{
				TxIDs: []string{"123", "456"},
			},
		},
		{
			name: "gettxoutproof optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("gettxoutproof", []string{"123", "456"},
					String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"))
			},
			staticCmd: func() interface{} {
				return NewGetTxOutProofCmd([]string{"123", "456"},
					String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettxoutproof","params":[["123","456"],` +
				`"000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"],"id":1}`,
			unmarshalled: &GetTxOutProofCmd{
				TxIDs:     []string{"123", "456"},
				BlockHash: String("000000000000034a7dedef4a161fa058a2d67a173a90155f3a2fe6fc132e0ebf"),
			},
		},
		{
			name: "gettxoutsetinfo",
			newCmd: func() (interface{}, error) {
				return NewCmd("gettxoutsetinfo")
			},
			staticCmd: func() interface{} {
				return NewGetTxOutSetInfoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"gettxoutsetinfo","params":[],"id":1}`,
			unmarshalled: &GetTxOutSetInfoCmd{},
		},
		{
			name: "getwork",
			newCmd: func() (interface{}, error) {
				return NewCmd("getwork")
			},
			staticCmd: func() interface{} {
				return NewGetWorkCmd(nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getwork","params":[],"id":1}`,
			unmarshalled: &GetWorkCmd{
				Data: nil,
			},
		},
		{
			name: "getwork optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("getwork", "00112233")
			},
			staticCmd: func() interface{} {
				return NewGetWorkCmd(String("00112233"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getwork","params":["00112233"],"id":1}`,
			unmarshalled: &GetWorkCmd{
				Data: String("00112233"),
			},
		},
		{
			name: "help",
			newCmd: func() (interface{}, error) {
				return NewCmd("help")
			},
			staticCmd: func() interface{} {
				return NewHelpCmd(nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"help","params":[],"id":1}`,
			unmarshalled: &HelpCmd{
				Command: nil,
			},
		},
		{
			name: "help optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("help", "getblock")
			},
			staticCmd: func() interface{} {
				return NewHelpCmd(String("getblock"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"help","params":["getblock"],"id":1}`,
			unmarshalled: &HelpCmd{
				Command: String("getblock"),
			},
		},
		{
			name: "invalidateblock",
			newCmd: func() (interface{}, error) {
				return NewCmd("invalidateblock", "123")
			},
			staticCmd: func() interface{} {
				return NewInvalidateBlockCmd("123")
			},
			marshalled: `{"jsonrpc":"1.0","method":"invalidateblock","params":["123"],"id":1}`,
			unmarshalled: &InvalidateBlockCmd{
				BlockHash: "123",
			},
		},
		{
			name: "ping",
			newCmd: func() (interface{}, error) {
				return NewCmd("ping")
			},
			staticCmd: func() interface{} {
				return NewPingCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"ping","params":[],"id":1}`,
			unmarshalled: &PingCmd{},
		},
		{
			name: "preciousblock",
			newCmd: func() (interface{}, error) {
				return NewCmd("preciousblock", "0123")
			},
			staticCmd: func() interface{} {
				return NewPreciousBlockCmd("0123")
			},
			marshalled: `{"jsonrpc":"1.0","method":"preciousblock","params":["0123"],"id":1}`,
			unmarshalled: &PreciousBlockCmd{
				BlockHash: "0123",
			},
		},
		{
			name: "reconsiderblock",
			newCmd: func() (interface{}, error) {
				return NewCmd("reconsiderblock", "123")
			},
			staticCmd: func() interface{} {
				return NewReconsiderBlockCmd("123")
			},
			marshalled: `{"jsonrpc":"1.0","method":"reconsiderblock","params":["123"],"id":1}`,
			unmarshalled: &ReconsiderBlockCmd{
				BlockHash: "123",
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return NewCmd("searchrawtransactions", "1Address")
			},
			staticCmd: func() interface{} {
				return NewSearchRawTransactionsCmd("1Address", nil, nil, nil, nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address"],"id":1}`,
			unmarshalled: &SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     Int(1),
				Skip:        Int(0),
				Count:       Int(100),
				VinExtra:    Int(0),
				Reverse:     Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return NewCmd("searchrawtransactions", "1Address", 0)
			},
			staticCmd: func() interface{} {
				return NewSearchRawTransactionsCmd("1Address",
					Int(0), nil, nil, nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0],"id":1}`,
			unmarshalled: &SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     Int(0),
				Skip:        Int(0),
				Count:       Int(100),
				VinExtra:    Int(0),
				Reverse:     Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return NewCmd("searchrawtransactions", "1Address", 0, 5)
			},
			staticCmd: func() interface{} {
				return NewSearchRawTransactionsCmd("1Address",
					Int(0), Int(5), nil, nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5],"id":1}`,
			unmarshalled: &SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     Int(0),
				Skip:        Int(5),
				Count:       Int(100),
				VinExtra:    Int(0),
				Reverse:     Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return NewCmd("searchrawtransactions", "1Address", 0, 5, 10)
			},
			staticCmd: func() interface{} {
				return NewSearchRawTransactionsCmd("1Address",
					Int(0), Int(5), Int(10), nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10],"id":1}`,
			unmarshalled: &SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     Int(0),
				Skip:        Int(5),
				Count:       Int(10),
				VinExtra:    Int(0),
				Reverse:     Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1)
			},
			staticCmd: func() interface{} {
				return NewSearchRawTransactionsCmd("1Address",
					Int(0), Int(5), Int(10), Int(1), nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1],"id":1}`,
			unmarshalled: &SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     Int(0),
				Skip:        Int(5),
				Count:       Int(10),
				VinExtra:    Int(1),
				Reverse:     Bool(false),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1, true)
			},
			staticCmd: func() interface{} {
				return NewSearchRawTransactionsCmd("1Address",
					Int(0), Int(5), Int(10), Int(1), Bool(true), nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1,true],"id":1}`,
			unmarshalled: &SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     Int(0),
				Skip:        Int(5),
				Count:       Int(10),
				VinExtra:    Int(1),
				Reverse:     Bool(true),
				FilterAddrs: nil,
			},
		},
		{
			name: "searchrawtransactions",
			newCmd: func() (interface{}, error) {
				return NewCmd("searchrawtransactions", "1Address", 0, 5, 10, 1, true, []string{"1Address"})
			},
			staticCmd: func() interface{} {
				return NewSearchRawTransactionsCmd("1Address",
					Int(0), Int(5), Int(10), Int(1), Bool(true), &[]string{"1Address"})
			},
			marshalled: `{"jsonrpc":"1.0","method":"searchrawtransactions","params":["1Address",0,5,10,1,true,["1Address"]],"id":1}`,
			unmarshalled: &SearchRawTransactionsCmd{
				Address:     "1Address",
				Verbose:     Int(0),
				Skip:        Int(5),
				Count:       Int(10),
				VinExtra:    Int(1),
				Reverse:     Bool(true),
				FilterAddrs: &[]string{"1Address"},
			},
		},
		{
			name: "sendrawtransaction",
			newCmd: func() (interface{}, error) {
				return NewCmd("sendrawtransaction", "1122")
			},
			staticCmd: func() interface{} {
				return NewSendRawTransactionCmd("1122", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"sendrawtransaction","params":["1122"],"id":1}`,
			unmarshalled: &SendRawTransactionCmd{
				HexTx:         "1122",
				AllowHighFees: Bool(false),
			},
		},
		{
			name: "sendrawtransaction optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("sendrawtransaction", "1122", false)
			},
			staticCmd: func() interface{} {
				return NewSendRawTransactionCmd("1122", Bool(false))
			},
			marshalled: `{"jsonrpc":"1.0","method":"sendrawtransaction","params":["1122",false],"id":1}`,
			unmarshalled: &SendRawTransactionCmd{
				HexTx:         "1122",
				AllowHighFees: Bool(false),
			},
		},
		/*
			{
				name: "setgenerate",
				newCmd: func() (interface{}, error) {
					return NewCmd("setgenerate", true)
				},
				staticCmd: func() interface{} {
					return NewSetGenerateCmd(true, nil)
				},
				marshalled: `{"jsonrpc":"1.0","method":"setgenerate","params":[true],"id":1}`,
				unmarshalled: &SetGenerateCmd{
					Generate:     true,
					GenProcLimit: Int(-1),
				},
			},
			{
				name: "setgenerate optional",
				newCmd: func() (interface{}, error) {
					return NewCmd("setgenerate", true, 6)
				},
				staticCmd: func() interface{} {
					return NewSetGenerateCmd(true, Int(6))
				},
				marshalled: `{"jsonrpc":"1.0","method":"setgenerate","params":[true,6],"id":1}`,
				unmarshalled: &SetGenerateCmd{
					Generate:     true,
					GenProcLimit: Int(6),
				},
			},
		*/
		{
			name: "stop",
			newCmd: func() (interface{}, error) {
				return NewCmd("stop")
			},
			staticCmd: func() interface{} {
				return NewStopCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"stop","params":[],"id":1}`,
			unmarshalled: &StopCmd{},
		},
		{
			name: "submitblock",
			newCmd: func() (interface{}, error) {
				return NewCmd("submitblock", "112233")
			},
			staticCmd: func() interface{} {
				return NewSubmitBlockCmd("112233", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"submitblock","params":["112233"],"id":1}`,
			unmarshalled: &SubmitBlockCmd{
				HexBlock: "112233",
				Options:  nil,
			},
		},
		{
			name: "submitblock optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("submitblock", "112233", `{"workid":"12345"}`)
			},
			staticCmd: func() interface{} {
				options := SubmitBlockOptions{
					WorkID: "12345",
				}
				return NewSubmitBlockCmd("112233", &options)
			},
			marshalled: `{"jsonrpc":"1.0","method":"submitblock","params":["112233",{"workid":"12345"}],"id":1}`,
			unmarshalled: &SubmitBlockCmd{
				HexBlock: "112233",
				Options: &SubmitBlockOptions{
					WorkID: "12345",
				},
			},
		},
		{
			name: "uptime",
			newCmd: func() (interface{}, error) {
				return NewCmd("uptime")
			},
			staticCmd: func() interface{} {
				return NewUptimeCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"uptime","params":[],"id":1}`,
			unmarshalled: &UptimeCmd{},
		},
		{
			name: "validateaddress",
			newCmd: func() (interface{}, error) {
				return NewCmd("validateaddress", "1Address")
			},
			staticCmd: func() interface{} {
				return NewValidateAddressCmd("1Address")
			},
			marshalled: `{"jsonrpc":"1.0","method":"validateaddress","params":["1Address"],"id":1}`,
			unmarshalled: &ValidateAddressCmd{
				Address: "1Address",
			},
		},
		{
			name: "verifychain",
			newCmd: func() (interface{}, error) {
				return NewCmd("verifychain")
			},
			staticCmd: func() interface{} {
				return NewVerifyChainCmd(nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifychain","params":[],"id":1}`,
			unmarshalled: &VerifyChainCmd{
				CheckLevel: Int32(3),
				CheckDepth: Int32(288),
			},
		},
		{
			name: "verifychain optional1",
			newCmd: func() (interface{}, error) {
				return NewCmd("verifychain", 2)
			},
			staticCmd: func() interface{} {
				return NewVerifyChainCmd(Int32(2), nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifychain","params":[2],"id":1}`,
			unmarshalled: &VerifyChainCmd{
				CheckLevel: Int32(2),
				CheckDepth: Int32(288),
			},
		},
		{
			name: "verifychain optional2",
			newCmd: func() (interface{}, error) {
				return NewCmd("verifychain", 2, 500)
			},
			staticCmd: func() interface{} {
				return NewVerifyChainCmd(Int32(2), Int32(500))
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifychain","params":[2,500],"id":1}`,
			unmarshalled: &VerifyChainCmd{
				CheckLevel: Int32(2),
				CheckDepth: Int32(500),
			},
		},
		{
			name: "verifymessage",
			newCmd: func() (interface{}, error) {
				return NewCmd("verifymessage", "1Address", "301234", "test")
			},
			staticCmd: func() interface{} {
				return NewVerifyMessageCmd("1Address", "301234", "test")
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifymessage","params":["1Address","301234","test"],"id":1}`,
			unmarshalled: &VerifyMessageCmd{
				Address:   "1Address",
				Signature: "301234",
				Message:   "test",
			},
		},
		{
			name: "verifytxoutproof",
			newCmd: func() (interface{}, error) {
				return NewCmd("verifytxoutproof", "test")
			},
			staticCmd: func() interface{} {
				return NewVerifyTxOutProofCmd("test")
			},
			marshalled: `{"jsonrpc":"1.0","method":"verifytxoutproof","params":["test"],"id":1}`,
			unmarshalled: &VerifyTxOutProofCmd{
				Proof: "test",
			},
		},
		{
			name: "waitforblockheight",
			newCmd: func() (interface{}, error) {
				return NewCmd("waitforblockheight", 123)
			},
			staticCmd: func() interface{} {
				return NewWaitForBlockHeightCmd(123, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"waitforblockheight","params":[123],"id":1}`,
			unmarshalled: &WaitForBlockHeightCmd{
				Height:  123,
				Timeout: Int(0),
			},
		},
		{
			name: "waitforblockheight optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("waitforblockheight", 123, 1)
			},
			staticCmd: func() interface{} {
				return NewWaitForBlockHeightCmd(123, Int(1))
			},
			marshalled: `{"jsonrpc":"1.0","method":"waitforblockheight","params":[123,1],"id":1}`,
			unmarshalled: &WaitForBlockHeightCmd{
				Height:  123,
				Timeout: Int(1),
			},
		},
		{
			name: "pruneblockchain",
			newCmd: func() (interface{}, error) {
				return NewCmd("pruneblockchain", 123)
			},
			staticCmd: func() interface{} {
				return NewPruneBlockChainCmd(123)
			},
			marshalled: `{"jsonrpc":"1.0","method":"pruneblockchain","params":[123],"id":1}`,
			unmarshalled: &PruneBlockChainCmd{
				Height: 123,
			},
		},
		{
			name: "echo",
			newCmd: func() (interface{}, error) {
				return NewCmd("echo")
			},
			staticCmd: func() interface{} {
				return NewEchoCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"echo","params":[],"id":1}`,
			unmarshalled: &EchoCmd{},
		},
		{
			name: "getchaintxstats",
			newCmd: func() (interface{}, error) {
				return NewCmd("getchaintxstats")
			},
			staticCmd: func() interface{} {
				return NewGetChainTxStatsCmd(nil, nil)
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getchaintxstats","params":[],"id":1}`,
			unmarshalled: &GetChainTxStatsCmd{},
		},
		{
			name: "getchaintxstats optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("getchaintxstats", 123, "test")
			},
			staticCmd: func() interface{} {
				return NewGetChainTxStatsCmd(Int32(123), String("test"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getchaintxstats","params":[123,"test"],"id":1}`,
			unmarshalled: &GetChainTxStatsCmd{
				Blocks:    Int32(123),
				BlockHash: String("test"),
			},
		},
		{
			name: "version",
			newCmd: func() (interface{}, error) {
				return NewCmd("version")
			},
			staticCmd: func() interface{} {
				return NewVersionCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"version","params":[],"id":1}`,
			unmarshalled: &VersionCmd{},
		},
		{
			name: "signmessagewithprivkey",
			newCmd: func() (interface{}, error) {
				return NewCmd("signmessagewithprivkey", "test", "abc")
			},
			staticCmd: func() interface{} {
				return NewSignMessageWithPrivkeyCmd("test", "abc")
			},
			marshalled: `{"jsonrpc":"1.0","method":"signmessagewithprivkey","params":["test","abc"],"id":1}`,
			unmarshalled: &SignMessageWithPrivkeyCmd{
				Privkey: "test",
				Message: "abc",
			},
		},
		{
			name: "generate",
			newCmd: func() (interface{}, error) {
				return NewCmd("generate", 123)
			},
			staticCmd: func() interface{} {
				return NewGenerateCmd(123, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"generate","params":[123],"id":1}`,
			unmarshalled: &GenerateCmd{
				NumBlocks: 123,
				MaxTries:  Uint64(1000000),
			},
		},
		{
			name: "generate optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("generate", 123, 666)
			},
			staticCmd: func() interface{} {
				return NewGenerateCmd(123, Uint64(666))
			},
			marshalled: `{"jsonrpc":"1.0","method":"generate","params":[123,666],"id":1}`,
			unmarshalled: &GenerateCmd{
				NumBlocks: 123,
				MaxTries:  Uint64(666),
			},
		},
		{
			name: "generatetoaddress",
			newCmd: func() (interface{}, error) {
				return NewCmd("generatetoaddress", 123, "test")
			},
			staticCmd: func() interface{} {
				return NewGenerateToAddressCmd(123, "test", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"generatetoaddress","params":[123,"test"],"id":1}`,
			unmarshalled: &GenerateToAddressCmd{
				NumBlocks: 123,
				Address:   "test",
				MaxTries:  Uint64(1000000),
			},
		},
		{
			name: "generatetoaddress optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("generatetoaddress", 123, "test", 666)
			},
			staticCmd: func() interface{} {
				return NewGenerateToAddressCmd(123, "test", Uint64(666))
			},
			marshalled: `{"jsonrpc":"1.0","method":"generatetoaddress","params":[123,"test",666],"id":1}`,
			unmarshalled: &GenerateToAddressCmd{
				NumBlocks: 123,
				Address:   "test",
				MaxTries:  Uint64(666),
			},
		},
		{
			name: "estimatefee",
			newCmd: func() (interface{}, error) {
				return NewCmd("estimatefee", 123)
			},
			staticCmd: func() interface{} {
				return NewEstimateFeeCmd(123)
			},
			marshalled: `{"jsonrpc":"1.0","method":"estimatefee","params":[123],"id":1}`,
			unmarshalled: &EstimateFeeCmd{
				NumBlocks: 123,
			},
		},
		{
			name: "getbestblock",
			newCmd: func() (interface{}, error) {
				return NewCmd("getbestblock")
			},
			staticCmd: func() interface{} {
				return NewGetBestBlockCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getbestblock","params":[],"id":1}`,
			unmarshalled: &GetBestBlockCmd{},
		},
		{
			name: "getheaders",
			newCmd: func() (interface{}, error) {
				return NewCmd("getheaders", []string{"test"}, "abc")
			},
			staticCmd: func() interface{} {
				return NewGetHeadersCmd([]string{"test"}, "abc")
			},
			marshalled: `{"jsonrpc":"1.0","method":"getheaders","params":[["test"],"abc"],"id":1}`,
			unmarshalled: &GetHeadersCmd{
				BlockLocators: []string{"test"},
				HashStop:      "abc",
			},
		},
		{
			name: "getcurrentnet",
			newCmd: func() (interface{}, error) {
				return NewCmd("getcurrentnet")
			},
			staticCmd: func() interface{} {
				return NewGetCurrentNetCmd()
			},
			marshalled:   `{"jsonrpc":"1.0","method":"getcurrentnet","params":[],"id":1}`,
			unmarshalled: &GetCurrentNetCmd{},
		},
		{
			name: "node",
			newCmd: func() (interface{}, error) {
				return NewCmd("node", "test", "abc")
			},
			staticCmd: func() interface{} {
				return NewNodeCmd("test", "abc", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"node","params":["test","abc"],"id":1}`,
			unmarshalled: &NodeCmd{
				SubCmd: "test",
				Target: "abc",
			},
		},
		{
			name: "node optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("node", "test", "abc", "xyz")
			},
			staticCmd: func() interface{} {
				return NewNodeCmd("test", "abc", String("xyz"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"node","params":["test","abc","xyz"],"id":1}`,
			unmarshalled: &NodeCmd{
				SubCmd:        "test",
				Target:        "abc",
				ConnectSubCmd: String("xyz"),
			},
		},
		{
			name: "getnewaddress",
			newCmd: func() (interface{}, error) {
				return NewCmd("getnewaddress")
			},
			staticCmd: func() interface{} {
				return NewGetNewAddressCmd(nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getnewaddress","params":[],"id":1}`,
			unmarshalled: &GetNewAddressCmd{
				Account: String(""),
			},
		},
		{
			name: "getnewaddress optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("getnewaddress", "abc")
			},
			staticCmd: func() interface{} {
				return NewGetNewAddressCmd(String("abc"))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getnewaddress","params":["abc"],"id":1}`,
			unmarshalled: &GetNewAddressCmd{
				Account: String("abc"),
			},
		},
		{
			name: "listunspent",
			newCmd: func() (interface{}, error) {
				return NewCmd("listunspent")
			},
			staticCmd: func() interface{} {
				return NewListUnspentCmd(nil, nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"listunspent","params":[],"id":1}`,
			unmarshalled: &ListUnspentCmd{
				MinConf:       Int32(1),
				MaxConf:       Int32(9999999),
				IncludeUnsafe: Bool(true),
			},
		},
		{
			name: "listunspent optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("listunspent", 3, 100, []string{"abc", "xyz"}, false)
			},
			staticCmd: func() interface{} {
				return NewListUnspentCmd(Int32(3), Int32(100), &[]string{"abc", "xyz"}, Bool(false))
			},
			marshalled: `{"jsonrpc":"1.0","method":"listunspent","params":[3,100,["abc","xyz"],false],"id":1}`,
			unmarshalled: &ListUnspentCmd{
				MinConf:       Int32(3),
				MaxConf:       Int32(100),
				Addresses:     &[]string{"abc", "xyz"},
				IncludeUnsafe: Bool(false),
			},
		},
		{
			name: "settxfee",
			newCmd: func() (interface{}, error) {
				return NewCmd("settxfee", 0.01)
			},
			staticCmd: func() interface{} {
				return NewSetTxFeeCmd(0.01)
			},
			marshalled: `{"jsonrpc":"1.0","method":"settxfee","params":[0.01],"id":1}`,
			unmarshalled: &SetTxFeeCmd{
				Amount: 0.01,
			},
		},
		{
			name: "sendtoaddress",
			newCmd: func() (interface{}, error) {
				return NewCmd("sendtoaddress", "abc", "0.01")
			},
			staticCmd: func() interface{} {
				return NewSendToAddressCmd("abc", "0.01", nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"sendtoaddress","params":["abc","0.01"],"id":1}`,
			unmarshalled: &SendToAddressCmd{
				Address:               "abc",
				Amount:                "0.01",
				SubtractFeeFromAmount: Bool(false),
			},
		},
		{
			name: "sendtoaddress optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("sendtoaddress", "abc", 0.01, "test", "xyz", true)
			},
			staticCmd: func() interface{} {
				return NewSendToAddressCmd("abc", 0.01, String("test"), String("xyz"), Bool(true))
			},
			marshalled: `{"jsonrpc":"1.0","method":"sendtoaddress","params":["abc",0.01,"test","xyz",true],"id":1}`,
			unmarshalled: &SendToAddressCmd{
				Address:               "abc",
				Amount:                0.01,
				Comment:               String("test"),
				CommentTo:             String("xyz"),
				SubtractFeeFromAmount: Bool(true),
			},
		},
		{
			name: "getbalance",
			newCmd: func() (interface{}, error) {
				return NewCmd("getbalance")
			},
			staticCmd: func() interface{} {
				return NewGetBalanceCmd(nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"getbalance","params":[],"id":1}`,
			unmarshalled: &GetBalanceCmd{
				MinConf: Int(1),
			},
		},
		{
			name: "getbalance optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("getbalance", "abc", 123)
			},
			staticCmd: func() interface{} {
				return NewGetBalanceCmd(String("abc"), Int(123))
			},
			marshalled: `{"jsonrpc":"1.0","method":"getbalance","params":["abc",123],"id":1}`,
			unmarshalled: &GetBalanceCmd{
				Account: String("abc"),
				MinConf: Int(123),
			},
		},
		{
			name: "gettransaction",
			newCmd: func() (interface{}, error) {
				return NewCmd("gettransaction", "abc")
			},
			staticCmd: func() interface{} {
				return NewGetTransactionCmd("abc", nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettransaction","params":["abc"],"id":1}`,
			unmarshalled: &GetTransactionCmd{
				Txid:             "abc",
				IncludeWatchOnly: Bool(false),
			},
		},
		{
			name: "gettransaction optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("gettransaction", "abc", true)
			},
			staticCmd: func() interface{} {
				return NewGetTransactionCmd("abc", Bool(true))
			},
			marshalled: `{"jsonrpc":"1.0","method":"gettransaction","params":["abc",true],"id":1}`,
			unmarshalled: &GetTransactionCmd{
				Txid:             "abc",
				IncludeWatchOnly: Bool(true),
			},
		},
		{
			name: "sendmany",
			newCmd: func() (interface{}, error) {
				return NewCmd("sendmany", "abc", `{"456":0.0123}`)
			},
			staticCmd: func() interface{} {
				amounts := map[string]AmountType{"456": .0123}
				return NewSendManyCmd("abc", amounts, nil, nil, nil)
			},
			marshalled: `{"jsonrpc":"1.0","method":"sendmany","params":["abc",{"456":0.0123}],"id":1}`,
			unmarshalled: &SendManyCmd{
				FromAccount: "abc",
				Amounts:     map[string]AmountType{"456": .0123},
				MinConf:     Int32(1),
			},
		},
		{
			name: "sendmany optional",
			newCmd: func() (interface{}, error) {
				return NewCmd("sendmany", "abc", `{"456":0.0123}`, 8, "xyz", []string{"test"})
			},
			staticCmd: func() interface{} {
				amounts := map[string]AmountType{"456": .0123}
				return NewSendManyCmd("abc", amounts, Int32(8), String("xyz"), &[]string{"test"})
			},
			marshalled: `{"jsonrpc":"1.0","method":"sendmany","params":["abc",{"456":0.0123},8,"xyz",["test"]],"id":1}`,
			unmarshalled: &SendManyCmd{
				FromAccount:     "abc",
				Amounts:         map[string]AmountType{"456": .0123},
				MinConf:         Int32(8),
				Comment:         String("xyz"),
				SubTractFeeFrom: &[]string{"test"},
			},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Marshal the command as created by the new static command
		// creation function.
		marshalled, err := MarshalCmd(testID, test.staticCmd())
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshalled, []byte(test.marshalled)) {
			t.Errorf("Test #%d (%s) unexpected marshalled data - "+
				"got %s, want %s", i, test.name, marshalled,
				test.marshalled)
			t.Errorf("\n%s\n%s", marshalled, test.marshalled)
			continue
		}

		// Ensure the command is created without error via the generic
		// new command creation function.
		cmd, err := test.newCmd()
		if err != nil {
			t.Errorf("Test #%d (%s) unexpected NewCmd error: %v ",
				i, test.name, err)
		}

		// Marshal the command as created by the generic new command
		// creation function.
		marshalled, err = MarshalCmd(testID, cmd)
		if err != nil {
			t.Errorf("MarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !bytes.Equal(marshalled, []byte(test.marshalled)) {
			t.Errorf("Test #%d (%s) unexpected marshalled data - "+
				"got %s, want %s", i, test.name, marshalled,
				test.marshalled)
			continue
		}

		var request Request
		if err := json.Unmarshal(marshalled, &request); err != nil {
			t.Errorf("Test #%d (%s) unexpected error while "+
				"unmarshalling JSON-RPC request: %v", i,
				test.name, err)
			continue
		}

		cmd, err = UnmarshalCmd(&request)
		if err != nil {
			t.Errorf("UnmarshalCmd #%d (%s) unexpected error: %v", i,
				test.name, err)
			continue
		}

		if !reflect.DeepEqual(cmd, test.unmarshalled) {
			t.Errorf("Test #%d (%s) unexpected unmarshalled command "+
				"- got %s, want %s", i, test.name,
				fmt.Sprintf("(%T) %+[1]v", cmd),
				fmt.Sprintf("(%T) %+[1]v\n", test.unmarshalled))
			continue
		}
	}
}

// TestChainSvrCmdErrors ensures any errors that occur in the command during
// custom mashal and unmarshal are as expected.
func TestChainSvrCmdErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		result     interface{}
		marshalled string
		err        error
	}{
		{
			name:       "template request with invalid type",
			result:     &TemplateRequest{},
			marshalled: `{"mode":1}`,
			err:        &json.UnmarshalTypeError{},
		},
		{
			name:       "invalid template request sigoplimit field",
			result:     &TemplateRequest{},
			marshalled: `{"sigoplimit":"invalid"}`,
			err:        Error{ErrorCode: ErrInvalidType},
		},
		{
			name:       "invalid template request sizelimit field",
			result:     &TemplateRequest{},
			marshalled: `{"sizelimit":"invalid"}`,
			err:        Error{ErrorCode: ErrInvalidType},
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		err := json.Unmarshal([]byte(test.marshalled), &test.result)
		if reflect.TypeOf(err) != reflect.TypeOf(test.err) {
			t.Errorf("Test #%d (%s) wrong error - got %T (%[2]v), "+
				"want %T", i, test.name, err, test.err)
			continue
		}

		if terr, ok := test.err.(Error); ok {
			gotErrorCode := err.(Error).ErrorCode
			if gotErrorCode != terr.ErrorCode {
				t.Errorf("Test #%d (%s) mismatched error code "+
					"- got %v (%v), want %v", i, test.name,
					gotErrorCode, terr, terr.ErrorCode)
				continue
			}
		}
	}
}
