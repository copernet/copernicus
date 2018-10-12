// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// NOTE: This file is intended to house the RPC commands that are supported by
// a chain server with btcd extensions.

package btcjson

// NodeSubCmd defines the type used in the addnode JSON-RPC command for the
// sub command field.
type NodeSubCmd string

const (
	// NConnect indicates the specified host that should be connected to.
	NConnect NodeSubCmd = "connect"

	// NRemove indicates the specified peer that should be removed as a
	// persistent peer.
	NRemove NodeSubCmd = "remove"

	// NDisconnect indicates the specified peer should be disonnected.
	NDisconnect NodeSubCmd = "disconnect"
)

// NodeCmd defines the dropnode JSON-RPC command.
type NodeCmd struct {
	SubCmd        NodeSubCmd `jsonrpcusage:"\"connect|remove|disconnect\""`
	Target        string
	ConnectSubCmd *string `jsonrpcusage:"\"perm|temp\""`
}

// NewNodeCmd returns a new instance which can be used to issue a `node`
// JSON-RPC command.
//
// The parameters which are pointers indicate they are optional.  Passing nil
// for optional parameters will use the default value.
func NewNodeCmd(subCmd NodeSubCmd, target string, connectSubCmd *string) *NodeCmd {
	return &NodeCmd{
		SubCmd:        subCmd,
		Target:        target,
		ConnectSubCmd: connectSubCmd,
	}
}

// GenerateCmd defines the generate JSON-RPC command.
type GenerateCmd struct {
	NumBlocks uint32  `json:"nblocks"`
	MaxTries  *uint64 `json:"maxtries" jsonrpcdefault:"1000000"`
}

// NewGenerateCmd returns a new instance which can be used to issue a generate
// JSON-RPC command.
func NewGenerateCmd(numBlocks uint32) *GenerateCmd {
	return &GenerateCmd{
		NumBlocks: numBlocks,
	}
}

// EstimateFeeCmd defines the estimatefee JSON-RPC command.
type EstimateFeeCmd struct {
	NumBlocks int64
}

// NewEstimateFeeCmd returns a new instance which can be used to issue a
// estimatefee JSON-RPC command.
func NewEstimateFeeCmd(numBlocks int64) *EstimateFeeCmd {
	return &EstimateFeeCmd{
		NumBlocks: numBlocks,
	}
}

// GenerateToAddressCmd defines the generatetoaddress JSON-RPC command.
type GenerateToAddressCmd struct {
	NumBlocks uint32  `json:"nblocks"`
	Address   string  `json:"address"`
	MaxTries  *uint64 `json:"maxtries" jsonrpcdefault:"1000000"`
}

// GetBestBlockCmd defines the getbestblock JSON-RPC command.
type GetBestBlockCmd struct{}

// NewGetBestBlockCmd returns a new instance which can be used to issue a
// getbestblock JSON-RPC command.
func NewGetBestBlockCmd() *GetBestBlockCmd {
	return &GetBestBlockCmd{}
}

// GetCurrentNetCmd defines the getcurrentnet JSON-RPC command.
type GetCurrentNetCmd struct{}

// NewGetCurrentNetCmd returns a new instance which can be used to issue a
// getcurrentnet JSON-RPC command.
func NewGetCurrentNetCmd() *GetCurrentNetCmd {
	return &GetCurrentNetCmd{}
}

// GetHeadersCmd defines the getheaders JSON-RPC command.
//
// NOTE: This is a btcsuite extension ported from
// github.com/decred/dcrd/dcrjson.
type GetHeadersCmd struct {
	BlockLocators []string `json:"blocklocators"`
	HashStop      string   `json:"hashstop"`
}

// NewGetHeadersCmd returns a new instance which can be used to issue a
// getheaders JSON-RPC command.
//
// NOTE: This is a btcsuite extension ported from
// github.com/decred/dcrd/dcrjson.
func NewGetHeadersCmd(blockLocators []string, hashStop string) *GetHeadersCmd {
	return &GetHeadersCmd{
		BlockLocators: blockLocators,
		HashStop:      hashStop,
	}
}

func init() {
	// No special flags for commands in this file.
	flags := UsageFlag(0)

	MustRegisterCmd("node", (*NodeCmd)(nil), flags)
	MustRegisterCmd("generate", (*GenerateCmd)(nil), flags)
	MustRegisterCmd("getbestblock", (*GetBestBlockCmd)(nil), flags)
	MustRegisterCmd("getcurrentnet", (*GetCurrentNetCmd)(nil), flags)
	MustRegisterCmd("getheaders", (*GetHeadersCmd)(nil), flags)
}
