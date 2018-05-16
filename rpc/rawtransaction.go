package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcboost/copernicus/blockchain"
	"github.com/btcboost/copernicus/core"
	"github.com/btcboost/copernicus/crypto"
	"github.com/btcboost/copernicus/internal/btcjson"
	"github.com/btcboost/copernicus/mempool"
	"github.com/btcboost/copernicus/net/msg"
	"github.com/btcboost/copernicus/utils"
	"github.com/btcboost/copernicus/utxo"
)

var rawTransactionHandlers = map[string]commandHandler{
	"getrawtransaction":    handleGetRawTransaction, // complete
	"createrawtransaction": handleCreateRawTransaction,
	"decoderawtransaction": handleDecodeRawTransaction,
	"decodescript":         handleDecodeScript,
	"sendrawtransaction":   handleSendRawTransaction,

	"signrawtransaction": handleSignRawTransaction,
	"gettxoutproof":      handleGetTxoutProof,
	"verifytxoutproof":   handleVerifyTxoutProof,
}

func handleGetRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetRawTransactionCmd)

	// Convert the provided transaction hash hex to a Hash.
	txHash, err := utils.GetHashFromStr(c.Txid)
	if err != nil {
		return nil, rpcDecodeHexError(c.Txid)
	}

	verbose := false
	if c.Verbose != nil {
		verbose = *c.Verbose != 0
	}

	tx, hashBlock, ok := GetTransaction(txHash, true)
	if !ok {
		if blockchain.GTxIndex {
			return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidAddressOrKey,
				"No such mempool or blockchain transaction")
		}
		return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidAddressOrKey,
			"No such mempool transaction. Use -txindex to enable blockchain transaction queries. Use gettransaction for wallet transactions.")
	}

	buf := bytes.NewBuffer(nil)
	err = tx.Serialize(buf)
	if err != nil {
		return nil, rpcDecodeHexError(c.Txid)
	}
	strHex := hex.EncodeToString(buf.Bytes())
	if !verbose {
		return strHex, nil
	}
	rawTxn, err := createTxRawResult(tx, hashBlock, msg.ActiveNetParams)
	if err != nil {
		return nil, err
	}
	return *rawTxn, nil
}

// createTxRawResult converts the passed transaction and associated parameters
// to a raw transaction JSON object.
func createTxRawResult(tx *core.Tx, hashBlock *utils.Hash, params *msg.BitcoinParams) (*btcjson.TxRawResult, error) {

	hash := tx.TxHash()
	txReply := &btcjson.TxRawResult{
		TxID:     hash.ToString(),
		Hash:     hash.ToString(),
		Size:     tx.SerializeSize(),
		Version:  tx.Version,
		LockTime: tx.LockTime,
		Vin:      createVinList(tx),
		Vout:     createVoutList(tx, params),
	}

	if !hashBlock.IsNull() {
		txReply.BlockHash = hashBlock.ToString()
		bindex := blockchain.GChainActive.FetchBlockIndexByHash(hashBlock) // todo realise: get *BlockIndex by blockhash
		if bindex != nil {
			if blockchain.GChainActive.Contains(bindex) {
				txReply.Confirmations = blockchain.GChainActive.Height() - bindex.Height + 1
				txReply.Time = bindex.Header.Time
				txReply.Blocktime = bindex.Header.Time
			} else {
				txReply.Confirmations = 0
			}
		}
	}
	return txReply, nil
}

// createVinList returns a slice of JSON objects for the inputs of the passed
// transaction.
func createVinList(tx *core.Tx) []btcjson.Vin {
	vinList := make([]btcjson.Vin, len(tx.Ins))
	for index, in := range tx.Ins {
		if tx.IsCoinBase() {
			vinList[index].Coinbase = hex.EncodeToString(in.Script.GetScriptByte())
		} else {
			vinList[index].Txid = in.PreviousOutPoint.Hash.ToString()
			vinList[index].Vout = in.PreviousOutPoint.Index
			vinList[index].ScriptSig.Asm = ScriptToAsmStr(in.Script, true)
			vinList[index].ScriptSig.Hex = hex.EncodeToString(in.Script.GetScriptByte())
		}
		vinList[index].Sequence = in.Sequence
	}
	return vinList
}

func ScriptToAsmStr(script *core.Script, attemptSighashDecode bool) string { // todo complete
	var str string
	var opcode byte
	vch := make([]byte, 0)
	b := script.GetScriptByte()
	for i := 0; i < len(b); i++ {
		if len(str) != 0 {
			str += " "
		}

		if !script.GetOp(&i, &opcode, &vch) {
			str += "[error]"
			return str
		}

		if opcode >= 0 && opcode <= core.OP_PUSHDATA4 {
			if len(vch) <= 4 {
				num, _ := core.GetCScriptNum(vch, false, core.DefaultMaxNumSize)
				str += fmt.Sprintf("%d", num.Value)
			} else {
				// the IsUnspendable check makes sure not to try to decode
				// OP_RETURN data that may match the format of a signature
				if attemptSighashDecode && !script.IsUnspendable() {
					var strSigHashDecode string
					// goal: only attempt to decode a defined sighash type from
					// data that looks like a signature within a scriptSig. This
					// won't decode correctly formatted public keys in Pubkey or
					// Multisig scripts due to the restrictions on the pubkey
					// formats (see IsCompressedOrUncompressedPubKey) being
					// incongruous with the checks in CheckSignatureEncoding.
					flags := crypto.ScriptVerifyStrictenc
					if vch[len(vch)-1]&crypto.SigHashForkID != 0 {
						// If the transaction is using SIGHASH_FORKID, we need
						// to set the apropriate flag.
						// TODO: Remove after the Hard Fork.
						flags |= crypto.ScriptEnableSigHashForkID
					}
					if ok, _ := crypto.CheckSignatureEncoding(vch, uint32(flags)); ok {
						//chsigHashType := vch[len(vch)-1]
						//if t, ok := crypto.MapSigHashTypes[chsigHashType]; ok { // todo realise define
						//	strSigHashDecode = "[" + t + "]"
						//	// remove the sighash type byte. it will be replaced
						//	// by the decode.
						//	vch = vch[:len(vch)-1]
						//}
					}

					str += hex.EncodeToString(vch) + strSigHashDecode
				} else {
					str += hex.EncodeToString(vch)
				}
			}
		} else {
			str += core.GetOpName(int(opcode))
		}
	}
	return str
}

// createVoutList returns a slice of JSON objects for the outputs of the passed
// transaction.
func createVoutList(tx *core.Tx, params *msg.BitcoinParams) []btcjson.Vout {
	voutList := make([]btcjson.Vout, len(tx.Outs))
	for index, out := range tx.Outs {
		voutList[index].Value = out.Value
		voutList[index].N = uint32(index)
		voutList[index].ScriptPubKey = ScriptPubKeyToJSON(out.Script, true)
	}

	return voutList
}

func ScriptPubKeyToJSON(script *core.Script, includeHex bool) btcjson.ScriptPubKeyResult { // todo complete

	return btcjson.ScriptPubKeyResult{}
}

func GetTransaction(hash *utils.Hash, allowSlow bool) (*core.Tx, *utils.Hash, bool) {
	tx := mempool.GetTxByHash(hash) // todo realize: in mempool get *core.Tx by hash
	if tx != nil {
		return tx, nil, true
	}

	if blockchain.GTxIndex {
		blockchain.GBlockTree.ReadTxIndex(hash)
		//blockchain.OpenBlockFile(, true)
		// todo complete
	}

	// use coin database to locate block that contains transaction, and scan it
	var indexSlow *core.BlockIndex
	if allowSlow {
		coin := utxo.AccessByTxid(blockchain.GCoinsTip, hash)
		if !coin.IsSpent() {
			indexSlow = blockchain.GChainActive.FetchBlockIndexByHeight(coin.GetHeight()) // todo realise : get *BlockIndex by height
		}
	}

	if indexSlow != nil {
		var block *core.Block
		if blockchain.ReadBlockFromDisk(block, indexSlow, msg.ActiveNetParams) {
			for _, tx := range block.Txs {
				if *hash == tx.TxHash() {
					return tx, &indexSlow.BlockHash, true
				}
			}
		}
	}

	return nil, nil, false
}

func handleCreateRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*
		c := cmd.(*btcjson.CreateRawTransactionCmd)

		// Validate the locktime, if given.
		if c.LockTime != nil &&
			(*c.LockTime < 0 || *c.LockTime > int64(wire.MaxTxInSequenceNum)) {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidParameter,
				Message: "Locktime out of range",
			}
		}

		// Add all transaction inputs to a new transaction after performing
		// some validity checks.
		mtx := wire.NewMsgTx(wire.TxVersion)
		for _, input := range c.Inputs {
			txHash, err := chainhash.NewHashFromStr(input.Txid)
			if err != nil {
				return nil, rpcDecodeHexError(input.Txid)
			}

			prevOut := wire.NewOutPoint(txHash, input.Vout)
			txIn := wire.NewTxIn(prevOut, []byte{}, nil)
			if c.LockTime != nil && *c.LockTime != 0 {
				txIn.Sequence = wire.MaxTxInSequenceNum - 1
			}
			mtx.AddTxIn(txIn)
		}

		// Add all transaction outputs to the transaction after performing
		// some validity checks.
		params := s.cfg.ChainParams
		for encodedAddr, amount := range c.Amounts {
			// Ensure amount is in the valid range for monetary amounts.
			if amount <= 0 || amount > btcutil.MaxSatoshi {
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCType,
					Message: "Invalid amount",
				}
			}

			// Decode the provided address.
			addr, err := btcutil.DecodeAddress(encodedAddr, params)
			if err != nil {
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidAddressOrKey,
					Message: "Invalid address or key: " + err.Error(),
				}
			}

			// Ensure the address is one of the supported types and that
			// the network encoded with the address matches the network the
			// server is currently on.
			switch addr.(type) {
			case *btcutil.AddressPubKeyHash:
			case *btcutil.AddressScriptHash:
			default:
				return nil, &btcjson.RPCError{
					Code:    btcjson.ErrRPCInvalidAddressOrKey,
					Message: "Invalid address or key",
				}
			}
			if !addr.IsForNet(params) {
				return nil, &btcjson.RPCError{
					Code: btcjson.ErrRPCInvalidAddressOrKey,
					Message: "Invalid address: " + encodedAddr +
						" is for the wrong network",
				}
			}

			// Create a new script which pays to the provided address.
			pkScript, err := txscript.PayToAddrScript(addr)
			if err != nil {
				context := "Failed to generate pay-to-address script"
				return nil, internalRPCError(err.Error(), context)
			}

			// Convert the amount to satoshi.
			satoshi, err := btcutil.NewAmount(amount)
			if err != nil {
				context := "Failed to convert amount"
				return nil, internalRPCError(err.Error(), context)
			}

			txOut := wire.NewTxOut(int64(satoshi), pkScript)
			mtx.AddTxOut(txOut)
		}

		// Set the Locktime, if given.
		if c.LockTime != nil {
			mtx.LockTime = uint32(*c.LockTime)
		}

		// Return the serialized and hex-encoded transaction.  Note that this
		// is intentionally not directly returning because the first return
		// value is a string and it would result in returning an empty string to
		// the client instead of nothing (nil) in the case of an error.
		mtxHex, err := messageToHex(mtx)
		if err != nil {
			return nil, err
		}
		return mtxHex, nil
	*/
	return nil, nil
}

func handleDecodeRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.DecodeRawTransactionCmd)

		// Deserialize the transaction.
		hexStr := c.HexTx
		if len(hexStr)%2 != 0 {
			hexStr = "0" + hexStr
		}
		serializedTx, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, rpcDecodeHexError(hexStr)
		}
		var mtx wire.MsgTx
		err = mtx.Deserialize(bytes.NewReader(serializedTx))
		if err != nil {
			return nil, &btcjson.RPCError{
				Code:    btcjson.ErrRPCDeserialization,
				Message: "TX decode failed: " + err.Error(),
			}
		}

		// Create and return the result.
		txReply := btcjson.TxRawDecodeResult{
			Txid:     mtx.TxHash().String(),
			Version:  mtx.Version,
			Locktime: mtx.LockTime,
			Vin:      createVinList(&mtx),
			Vout:     createVoutList(&mtx, s.cfg.ChainParams, nil),
		}
		return txReply, nil*/
	return nil, nil
}

func handleDecodeScript(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/*	c := cmd.(*btcjson.DecodeScriptCmd)

		// Convert the hex script to bytes.
		hexStr := c.HexScript
		if len(hexStr)%2 != 0 {
			hexStr = "0" + hexStr
		}
		script, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, rpcDecodeHexError(hexStr)
		}

		// The disassembled string will contain [error] inline if the script
		// doesn't fully parse, so ignore the error here.
		disbuf, _ := txscript.DisasmString(script)

		// Get information about the script.
		// Ignore the error here since an error means the script couldn't parse
		// and there is no additinal information about it anyways.
		scriptClass, addrs, reqSigs, _ := txscript.ExtractPkScriptAddrs(script,
			s.cfg.ChainParams)
		addresses := make([]string, len(addrs))
		for i, addr := range addrs {
			addresses[i] = addr.EncodeAddress()
		}

		// Convert the script itself to a pay-to-script-hash address.
		p2sh, err := btcutil.NewAddressScriptHash(script, s.cfg.ChainParams)
		if err != nil {
			context := "Failed to convert script to pay-to-script-hash"
			return nil, internalRPCError(err.Error(), context)
		}

		// Generate and return the reply.
		reply := btcjson.DecodeScriptResult{
			Asm:       disbuf,
			ReqSigs:   int32(reqSigs),
			Type:      scriptClass.String(),
			Addresses: addresses,
		}
		if scriptClass != txscript.ScriptHashTy {
			reply.P2sh = p2sh.EncodeAddress()
		}
		return reply, nil*/
	return nil, nil
}

func handleSendRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	/* c := cmd.(*btcjson.SendRawTransactionCmd)
	// Deserialize and send off to tx relay
	hexStr := c.HexTx
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	serializedTx, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, rpcDecodeHexError(hexStr)
	}
	var msgTx msg.TxMessage
	err = msgTx.BitcoinParse(bytes.NewReader(serializedTx),0)
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "TX decode failed: " + err.Error(),
		}
	}

	// Use 0 for the tag to represent local node.
	acceptedTxs, err := s.cfg.TxMemPool.ProcessTransaction(msgTx.Tx, false, false, 0)
	blockchain.
	if err != nil {
		// When the error is a rule error, it means the transaction was
		// simply rejected as opposed to something actually going wrong,
		// so log it as such.  Otherwise, something really did go wrong,
		// so log it as an actual error.  In both cases, a JSON-RPC
		// error is returned to the client with the deserialization
		// error code (to match bitcoind behavior).
		if _, ok := err.(mempool.RuleError); ok {
			logs.Debug("Rejected transaction %v: %v", tx.Hash(),
				err)
		} else {
			logs.Error("Failed to process transaction %v: %v",
				tx.Hash(), err)
		}
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "TX rejected: " + err.Error(),
		}
	}

	// When the transaction was accepted it should be the first item in the
	// returned array of accepted transactions.  The only way this will not
	// be true is if the API for ProcessTransaction changes and this code is
	// not properly updated, but ensure the condition holds as a safeguard.
	//
	// Also, since an error is being returned to the caller, ensure the
	// transaction is removed from the memory pool.
	if len(acceptedTxs) == 0 || !acceptedTxs[0].Tx.Hash().IsEqual(tx.Hash()) {
		s.cfg.TxMemPool.RemoveTransaction(tx, true)

		errStr := fmt.Sprintf("transaction %v is not in accepted list",
			tx.Hash())
		return nil, internalRPCError(errStr, "")
	}

	// Generate and relay inventory vectors for all newly accepted
	// transactions into the memory pool due to the original being
	// accepted.
	s.cfg.ConnMgr.RelayTransactions(acceptedTxs)

	// Notify both websocket and getblocktemplate long poll clients of all
	// newly accepted transactions.
	s.NotifyNewTransactions(acceptedTxs)

	// Keep track of all the sendrawtransaction request txns so that they
	// can be rebroadcast if they don't make their way into a block.
	txD := acceptedTxs[0]
	iv := wire.NewInvVect(wire.InvTypeTx, txD.Tx.Hash())
	s.cfg.ConnMgr.AddRebroadcastInventory(iv, txD)

	return tx.Hash().String(), nil*/
	return nil, nil
} //Todo

func handleSignRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleGetTxoutProof(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func handleVerifyTxoutProof(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	return nil, nil
}

func registeRawTransactionRPCCommands() {
	for name, handler := range rawTransactionHandlers {
		appendCommand(name, handler)
	}
}
