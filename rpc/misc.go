package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/copernet/copernicus/conf"
	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/model"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/net/server"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/service"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/base58"
)

// API version constants
const (
	jsonrpcSemverString = "1.0.0"
	jsonrpcSemverMajor  = 1
	jsonrpcSemverMinor  = 1
	jsonrpcSemverPatch  = 0
)

var miscHandlers = map[string]commandHandler{
	"getinfo":                handleGetInfo,
	"validateaddress":        handleValidateAddress,
	"createmultisig":         handleCreatemultisig,
	"verifymessage":          handleVerifyMessage,
	"signmessagewithprivkey": handleSignMessageWithPrivkey,
	"setmocktime":            handleSetMocktime,
	"echo":                   handleEcho,
	"help":                   handleHelp,
	"stop":                   handleStop,
	"version":                handleVersion,
}

func handleGetInfo(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	best := chain.GetInstance().Tip()
	var height int32
	if best != nil {
		height = best.Height
	}

	request := &service.GetConnectionCountRequest{}
	response, err := server.ProcessForRPC(request)
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Can not acquire connection count",
		}
	}
	count, ok := response.(*service.GetConnectionCountResponse)
	if !ok {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "Server handle error",
		}
	}

	ret := &btcjson.InfoChainResult{
		Version:         1000000*conf.AppMajor + 10000*conf.AppMinor + 100*conf.AppPatch,
		ProtocolVersion: int32(wire.ProtocolVersion),
		Blocks:          height,
		TimeOffset:      util.GetTimeOffset(),
		Connections:     int32(count.Count),
		Proxy:           conf.Cfg.P2PNet.Proxy,
		Difficulty:      getDifficulty(chain.GetInstance().Tip()),
		TestNet:         model.ActiveNetParams.BitcoinNet == wire.TestNet3,
		RelayFee:        0, // todo define DefaultMinRelayTxFee
	}

	return ret, nil
}

// handleValidateAddress implements the validateaddress command.
func handleValidateAddress(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.ValidateAddressCmd)

	result := btcjson.ValidateAddressChainResult{}
	dest, err := script.AddressFromString(c.Address)
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
	/*	c := cmd.(*btcjson.VerifyMessageCmd)

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

		return bytes.Equal(addr2.EncodeToPubKeyHash(), hash160), nil*/ //todo open
	return nil, nil
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

	buf := bytes.NewBuffer(make([]byte, 0, 4+len(c.Message)))
	err = util.BinarySerializer.PutUint32(buf, binary.LittleEndian, uint32(chain.GetInstance().GetParams().BitcoinNet))
	if err != nil {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCInternalError,
			Message: "serialize BitcoinNet error",
		}
	}
	buf.Write([]byte(c.Message))

	originBytes := util.DoubleSha256Bytes(buf.Bytes())
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

	if !model.ActiveNetParams.MineBlocksOnDemands {
		return nil, btcjson.RPCError{
			Code:    btcjson.RPCForbiddenBySafeMode,
			Message: "etmocktime for regression testing (-regtest mode) only",
		}
	}

	// For now, don't change mocktime if we're in the middle of validation, as
	// this could have an effect on mempool time-based eviction, as well as
	// IsCurrentForFeeEstimation() and IsInitialBlockDownload().
	// figure out the right way to synchronize around mocktime, and
	// ensure all callsites of GetTime() are accessing this safely.
	util.SetMockTime(c.Timestamp)

	return nil, nil
}

func handleEcho(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	const echoArgPrefix = "arg"
	if _, ok := cmd.(*map[string]json.RawMessage); !ok {
		return cmd, nil
	}
	// JSON format
	params := cmd.(*map[string]json.RawMessage)
	retLen := 0
	args := make(map[int]interface{})
	for argName, arg := range *params {
		if !strings.HasPrefix(strings.ToLower(argName), echoArgPrefix) {
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidParameter,
				Message: "Unknown named parameter " + argName,
			}
		}
		argIdx := argName[len(echoArgPrefix):]
		if index, err := strconv.Atoi(argIdx); err == nil {
			args[index] = arg
			if index >= retLen {
				retLen = index + 1
			}
		}
	}
	result := make([]interface{}, retLen)
	for index, arg := range args {
		result[index] = arg
	}
	return result, nil
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
	return "Copernicus server stopping", nil
}

func handleVersion(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
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

func registerMiscRPCCommands() {
	for name, handler := range miscHandlers {
		appendCommand(name, handler)
	}
}
