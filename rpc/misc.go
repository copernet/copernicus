package rpc

import (
	"encoding/base64"

	"github.com/btcboost/copernicus/conf"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/internal/btcjson"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/util/base58"
	"github.com/btcsuite/btcd/mempool"
)

var miscHandlers = map[string]commandHandler{
	"getinfo":                handleGetInfo, // complete
	"validateaddress":        handleValidateAddress,
	"createmultisig":         handleCreatemultisig,
	"verifymessage":          handleVerifyMessage,
	"signmessagewithprivkey": handleSignMessageWithPrivkey, // todo 1
	"setmocktime":            handleSetMocktime,            // todo 2
	"echo":                   handleEcho,                   // todo 3
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
	//c := cmd.(*btcjson.ValidateAddressCmd)

	result := btcjson.ValidateAddressChainResult{}
	/*	addr, err := utils.DecodeAddress(c.Address, conf.AppConf.ChainParams)
		if err != nil {
			// Return the default value (false) for IsValid.
			return result, nil
		}

		result.Address = addr.EncodeAddress()   */ // TODO realise
	result.IsValid = true

	return result, nil
}

func handleCreatemultisig(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleVerifyMessage(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*
		c := cmd.(*btcjson.VerifyMessageCmd)

		// Decode the provided address.
		params := msg.ActiveNetParams
		addr, err := btcutil.DecodeAddress(c.Address, params)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Invalid address or key: " + err.Error(),
			}
		}

		// Only P2PKH addresses are valid for signing.
		if _, ok := addr.(*btcutil.AddressPubKeyHash); !ok {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCType,
				Message: "Address is not a pay-to-pubkey-hash address",
			}
		}

		// Decode base64 signature.
		sig, err := base64.StdEncoding.DecodeString(c.Signature)
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCParse.Code,
				Message: "Malformed base64 encoding: " + err.Error(),
			}
		}

		// Validate the signature - this just shows that it was valid at all.
		// we will compare it with the key next.
		var buf bytes.Buffer
		wire.WriteVarString(&buf, 0, "Bitcoin Signed Message:\n")
		wire.WriteVarString(&buf, 0, c.Message)
		expectedMessageHash := chainhash.DoubleHashB(buf.Bytes())
		pk, wasCompressed, err := btcec.RecoverCompact(btcec.S256(), sig,
			expectedMessageHash)
		if err != nil {
			// Mirror Bitcoin Core behavior, which treats error in
			// RecoverCompact as invalid signature.
			return false, nil
		}

		// Reconstruct the pubkey hash.
		var serializedPK []byte
		if wasCompressed {
			serializedPK = pk.SerializeCompressed()
		} else {
			serializedPK = pk.SerializeUncompressed()
		}
		address, err := btcutil.NewAddressPubKey(serializedPK, params)
		if err != nil {
			// Again mirror Bitcoin Core behavior, which treats error in public key
			// reconstruction as invalid signature.
			return false, nil
		}

		// Return boolean if addresses match.
		return address.EncodeAddress() == c.Address, nil
	*/
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
	return nil, nil
}

func handleEcho(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
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
