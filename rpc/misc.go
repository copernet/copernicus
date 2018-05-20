package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/internal/btcjson"
	"github.com/btcboost/copernicus/model/bitaddr"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/util/base58"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/mempool"
)

var miscHandlers = map[string]commandHandler{
	"getinfo":                handleGetInfo,         // complete
	"validateaddress":        handleValidateAddress, // complete
	"createmultisig":         handleCreatemultisig,
	"verifymessage":          handleVerifyMessage,          // complete
	"signmessagewithprivkey": handleSignMessageWithPrivkey, // complete
	"setmocktime":            handleSetMocktime,            // complete
	"echo":                   handleEcho,                   // complete
	"help":                   handleHelp,                   // complete
	"stop":                   handleStop,                   // complete
}

func handleGetInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	best := chain.GlobalChain.Tip()
	var height int32
	if best == nil {
		height = 0
	}

	ret := &btcjson.InfoChainResult{
		Version:         protocol.Copernicus,
		ProtocolVersion: int32(protocol.BitcoinProtocolVersion),
		Blocks:          height,
		TimeOffset:      util.GetTimeOffset(),
		//Connections: s.cfg.ConnMgr.ConnectedCount(),		// todo open
		Proxy:      conf.AppConf.Proxy,
		Difficulty: getDifficulty(chain.GlobalChain.Tip()),
		TestNet:    conf.AppConf.TestNet3,
		RelayFee:   float64(mempool.DefaultMinRelayTxFee),
	}

	return ret, nil
}

// handleValidateAddress implements the validateaddress command.
func handleValidateAddress(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.ValidateAddressCmd)

	result := btcjson.ValidateAddressChainResult{}
	dest, err := bitaddr.AddressFromString(c.Address)
	if err != nil {
		result.IsValid = false
		return result, nil
	}

	result.IsValid = true
	result.Address = c.Address
	result.ScriptPubKey = hex.EncodeToString(dest.EncodeToPubKeyHash())

	return result, nil
}

func handleCreatemultisig(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleVerifyMessage(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.VerifyMessageCmd)

	addr, err := bitaddr.AddressFromString(c.Address)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCType,
			Message: "Invalid address",
		}
	}

	hash160 := addr.EncodeToPubKeyHash()
	if hash160 == nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.ErrRPCType,
			Message: "Address does not refer to key",
		}
	}

	data := []byte(strMessageMagic + c.Message)
	hash := chainhash.DoubleHashB(data)
	originBytes, err := base64.StdEncoding.DecodeString(c.Signature)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Malformed base64 encoding",
		}
	}

	var pk crypto.PublicKey
	pk, wasCompressed, err := RecoverCompact(curveInstance, originBytes, hash) // todo realise
	if err != nil {
		return false, nil
	}

	pubKeyBytes := pk.ToBytes()
	addr2, err := bitaddr.AddressFromPublicKey(pubKeyBytes)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Invalid Public Key encoding",
		}
	}

	return bytes.Equal(addr2.EncodeToPubKeyHash(), hash160), nil
}

func handleSignMessageWithPrivkey(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SignMessageWithPrivkeyCmd)

	bs, _, err := base58.CheckDecode(c.Privkey)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Invalid private key",
		}
	}
	privKey := crypto.PrivateKeyFromBytes(bs)

	data := []byte(strMessageMagic + c.Message) // todo define <strMessageMagic>
	originBytes := util.DoubleSha256Bytes(data)
	signature, err := privKey.Sign(originBytes)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInvalidAddressOrKey,
			Message: "Sign failed",
		}
	}
	return base64.StdEncoding.EncodeToString(signature.Serialize()), nil
}

func handleSetMocktime(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SetMocktimeCmd)

	if !consensus.ActiveNetParams.MineBlocksOnDemands {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCForbiddenBySafeMode,
			Message: "etmocktime for regression testing (-regtest mode) only",
		}
	}

	// For now, don't change mocktime if we're in the middle of validation, as
	// this could have an effect on mempool time-based eviction, as well as
	// IsCurrentForFeeEstimation() and IsInitialBlockDownload().
	// TODO: figure out the right way to synchronize around mocktime, and
	// ensure all callsites of GetTime() are accessing this safely.
	util.SetMockTime(c.Timestamp)

	return nil, nil
}

func handleEcho(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return cmd, nil
}

// handleHelp implements the help command.
func handleHelp(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.HelpCmd)
	var command string
	if c.Command != nil {
		command = *c.Command
	}
	if command == "" {
		usage, err := s.helpCacher.rpcUsage(false)
		if err != nil {
			context := "Failed to generate RPC usage"
			return nil, internalRPCError(err.Error(), context)
		}
		return usage, nil
	}

	if _, ok := rpcHandlers[command]; !ok {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Unknown command: " + command,
		}
	}

	help, err := s.helpCacher.rpcMethodHelp(command)
	if err != nil {
		context := "Failed to generate help"
		return nil, internalRPCError(err.Error(), context)
	}
	return help, nil
}

// handleStop implements the stop command.
func handleStop(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	select {
	case s.requestProcessShutdown <- struct{}{}:
	default:
	}
	return "stopping.", nil
}

func registerMiscRPCCommands() {
	for name, handler := range miscHandlers {
		appendCommand(name, handler)
	}
}
