package rpc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/btcboost/copernicus/internal/btcjson"
	utxo2 "github.com/btcboost/copernicus/logic/utxo"
	"github.com/btcboost/copernicus/model/bitaddr"
	"github.com/btcboost/copernicus/model/block"
	"github.com/btcboost/copernicus/model/blockindex"
	"github.com/btcboost/copernicus/model/chain"
	"github.com/btcboost/copernicus/model/consensus"
	"github.com/btcboost/copernicus/model/mempool"
	"github.com/btcboost/copernicus/model/opcodes"
	"github.com/btcboost/copernicus/model/outpoint"
	"github.com/btcboost/copernicus/model/script"
	"github.com/btcboost/copernicus/model/tx"
	"github.com/btcboost/copernicus/model/txin"
	"github.com/btcboost/copernicus/model/txout"
	"github.com/btcboost/copernicus/model/utxo"
	"github.com/btcboost/copernicus/util"
	"github.com/btcboost/copernicus/util/amount"
	"github.com/btcsuite/btcd/wire"
)

var rawTransactionHandlers = map[string]commandHandler{
	"getrawtransaction":    handleGetRawTransaction,    // complete
	"createrawtransaction": handleCreateRawTransaction, // complete
	"decoderawtransaction": handleDecodeRawTransaction, // complete
	"decodescript":         handleDecodeScript,
	"sendrawtransaction":   handleSendRawTransaction, // complete

	"signrawtransaction": handleSignRawTransaction,
	"gettxoutproof":      handleGetTxoutProof,
	"verifytxoutproof":   handleVerifyTxoutProof,
}

func handleGetRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetRawTransactionCmd)

	// Convert the provided transaction hash hex to a Hash.
	txHash, err := util.GetHashFromStr(c.Txid)
	if err != nil {
		return nil, rpcDecodeHexError(c.Txid)
	}

	verbose := false
	if c.Verbose != nil {
		verbose = *c.Verbose != 0
	}

	tx, hashBlock, ok := GetTransaction(txHash, true)
	if !ok {
		if chain.GTxIndex {
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
	rawTxn, err := createTxRawResult(tx, hashBlock, consensus.ActiveNetParams)
	if err != nil {
		return nil, err
	}
	return *rawTxn, nil
}

// createTxRawResult converts the passed transaction and associated parameters
// to a raw transaction JSON object.
func createTxRawResult(tx *tx.Tx, hashBlock *util.Hash, params *consensus.BitcoinParams) (*btcjson.TxRawResult, error) {

	hash := tx.TxHash()
	txReply := &btcjson.TxRawResult{
		TxID:     hash.String(),
		Hash:     hash.String(),
		Size:     int(tx.SerializeSize()),
		Version:  tx.GetVersion(),
		LockTime: tx.GetLockTime(),
		Vin:      createVinList(tx),
		Vout:     createVoutList(tx, params),
	}

	if !hashBlock.IsNull() {
		txReply.BlockHash = hashBlock.String()
		bindex := chain.GlobalChain.FindBlockIndex(*hashBlock) // todo realise: get *BlockIndex by blockhash
		if bindex != nil {
			if chain.GlobalChain.Contains(bindex) {
				txReply.Confirmations = chain.GlobalChain.Height() - bindex.Height + 1
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
func createVinList(tx *tx.Tx) []btcjson.Vin {
	vinList := make([]btcjson.Vin, len(tx.GetIns()))
	for index, in := range tx.GetIns() {
		if tx.IsCoinBase() {
			vinList[index].Coinbase = hex.EncodeToString(in.GetScriptSig().GetScriptByte())
		} else {
			vinList[index].Txid = in.PreviousOutPoint.Hash.String()
			vinList[index].Vout = in.PreviousOutPoint.Index
			vinList[index].ScriptSig.Asm = ScriptToAsmStr(in.GetScriptSig(), true)
			vinList[index].ScriptSig.Hex = hex.EncodeToString(in.GetScriptSig().GetData())
		}
		vinList[index].Sequence = in.Sequence
	}
	return vinList
}

func ScriptToAsmStr(s *script.Script, attemptSighashDecode bool) string { // todo complete
	var str string
	var opcode byte
	vch := make([]byte, 0)
	b := s.GetData()
	for i := 0; i < len(b); i++ {
		if len(str) != 0 {
			str += " "
		}

		if !s.GetOp(&i, &opcode, &vch) {
			str += "[error]"
			return str
		}

		if opcode >= 0 && opcode <= opcodes.OP_PUSHDATA4 {
			if len(vch) <= 4 {
				num, _ := script.GetCScriptNum(vch, false, script.DefaultMaxNumSize)
				str += fmt.Sprintf("%d", num.Value)
			} else {
				// the IsUnspendable check makes sure not to try to decode
				// OP_RETURN data that may match the format of a signature
				if attemptSighashDecode && !s.IsUnspendable() {
					var strSigHashDecode string
					// goal: only attempt to decode a defined sighash type from
					// data that looks like a signature within a scriptSig. This
					// won't decode correctly formatted public keys in Pubkey or
					// Multisig scripts due to the restrictions on the pubkey
					// formats (see IsCompressedOrUncompressedPubKey) being
					// incongruous with the checks in CheckSignatureEncoding.
					flags := script.ScriptVerifyStrictEnc
					if vch[len(vch)-1]&script.SigHashForkID != 0 {
						// If the transaction is using SIGHASH_FORKID, we need
						// to set the apropriate flag.
						// TODO: Remove after the Hard Fork.
						flags |= script.ScriptEnableSigHashForkId
					}
					if ok, _ := script.CheckSignatureEncoding(vch, uint32(flags)); ok {
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
			str += opcodes.GetOpName(int(opcode))
		}
	}
	return str
}

// createVoutList returns a slice of JSON objects for the outputs of the passed
// transaction.
func createVoutList(tx *tx.Tx, params *consensus.BitcoinParams) []btcjson.Vout {
	voutList := make([]btcjson.Vout, tx.GetOutsCount())
	for i := 0; i < tx.GetOutsCount(); i++ {
		out := tx.GetTxOut(i)
		voutList[i].Value = out.GetValue()
		voutList[i].N = uint32(i)
		voutList[i].ScriptPubKey = ScriptPubKeyToJSON(out.GetScriptPubKey(), true)
	}

	return voutList
}

func ScriptPubKeyToJSON(script *script.Script, includeHex bool) btcjson.ScriptPubKeyResult { // todo complete
	result := btcjson.ScriptPubKeyResult{}

	result.Asm = ScriptToAsmStr(script, includeHex)
	if includeHex {
		result.Hex = hex.EncodeToString(script.GetData())
	}

	t, addresses, required, ok := script.ExtractDestinations(script)
	if !ok {
		result.Type = script.GetTxnOutputType(t)
		return result
	}

	result.ReqSigs = required
	result.Type = script.GetTxnOutputType(t)

	result.Addresses = make([]string, 0, len(addresses))
	for _, address := range addresses {
		result.Addresses = append(result.Addresses, address.String())
	}

	return result
}

func GetTransaction(hash *util.Hash, allowSlow bool) (*tx.Tx, *util.Hash, bool) {
	entry := mempool.Gpool.FindTx(*hash) // todo realize: in mempool get *core.Tx by hash
	if entry != nil {
		return entry.Tx, nil, true
	}

	if chain.GTxIndex {
		chain.GBlockTree.ReadTxIndex(hash)
		//blockchain.OpenBlockFile(, true)
		// todo complete
	}

	// use coin database to locate block that contains transaction, and scan it
	var indexSlow *blockindex.BlockIndex
	if allowSlow {
		coin := utxo2.AccessByTxid(utxo.GetUtxoCacheInstance(), hash)
		if !coin.IsSpent() {
			indexSlow = chain.GlobalChain.GetIndex(int(coin.GetHeight())) // todo realise : get *BlockIndex by height
		}
	}

	if indexSlow != nil {
		var bk *block.Block
		if chain.ReadBlockFromDisk(bk, indexSlow, consensus.ActiveNetParams) {
			for _, tx := range bk.Txs {
				if *hash == tx.TxHash() {
					return tx, &indexSlow.BlockHash, true
				}
			}
		}
	}

	return nil, nil, false
}

func handleCreateRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.CreateRawTransactionCmd)

	var transaction *tx.Tx
	// Validate the locktime, if given.
	if c.LockTime != nil &&
		(*c.LockTime < 0 || *c.LockTime > int64(wire.MaxTxInSequenceNum)) {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCInvalidParameter,
			Message: "Locktime out of range",
		}

		transaction = tx.NewTx(uint32(*c.LockTime), 0)
	}

	for _, input := range c.Inputs {
		hash, err := util.GetHashFromStr(input.Txid)
		if err != nil {
			return nil, rpcDecodeHexError(input.Txid)
		}

		if input.Vout < 0 {
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrInvalidParameter,
				Message: "Invalid parameter, vout must be positive",
			}
		}

		sequence := uint32(math.MaxUint32)
		if transaction.GetLockTime() != 0 {
			sequence = math.MaxUint32 - 1
		}

		// todo lack handle with sequence parameter(optional), is reasonable?
		in := txin.NewTxIn(outpoint.NewOutPoint(*hash, input.Vout), &script.Script{}, sequence)
		transaction.AddTxIn(in)
	}

	for address, cost := range c.Amounts {
		// todo do not support the key named 'data' in btcd
		addr, err := bitaddr.AddressFromString(address)
		if err != nil {
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrRPCInvalidAddressOrKey,
				Message: "Invalid Bitcoin address: " + address,
			}
		}

		outValue := int64(cost * 1e8)
		if !amount.MoneyRange(outValue) {
			return nil, btcjson.RPCError{
				Code:    btcjson.ErrInvalidParameter,
				Message: "Invalid amount",
			}
		}
		out := txout.NewTxOut(outValue, script.NewScriptRaw(addr.EncodeToPubKeyHash()))
		transaction.AddTxOut(out)
	}

	buf := bytes.NewBuffer(nil)
	transaction.Serialize(buf)

	return hex.EncodeToString(buf.Bytes()), nil
}

func handleDecodeRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.DecodeRawTransactionCmd)

	// Deserialize the transaction.
	serializedTx, err := hex.DecodeString(c.HexTx)
	if err != nil {
		return nil, rpcDecodeHexError(c.HexTx)
	}

	var transaction tx.Tx
	err = transaction.Unserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, &btcjson.RPCError{
			Code:    btcjson.ErrRPCDeserialization,
			Message: "TX decode failed: " + err.Error(),
		}
	}

	// Create and return the result.
	txReply := btcjson.TxRawDecodeResult{
		Txid:     transaction.Hash.String(),
		Hash:     transaction.Hash.String(),
		Size:     transaction.SerializeSize(),
		Version:  transaction.GetVersion(),
		Locktime: transaction.GetLockTime(),
		Vin:      createVinList(&transaction),
		Vout:     createVoutList(&transaction, consensus.ActiveNetParams),
	}

	return txReply, nil
}

func handleDecodeScript(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.DecodeScriptCmd)

	// Convert the hex script to bytes.
	scriptByte, err := hex.DecodeString(c.HexScript)
	if err != nil {
		return nil, rpcDecodeHexError(c.HexScript)
	}
	st := script.NewScriptRaw(scriptByte)

	ret := ScriptPubKeyToJSON(st, false)

	if ret.Type != "scripthash" {
		// P2SH cannot be wrapped in a P2SH. If this script is already a P2SH,
		// don't return the address for a P2SH of the P2SH.
		ret.P2SH = EncodeDestination(scriptByte) // todo realise
	}

	return ret, nil
}

func handleSendRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SendRawTransactionCmd)

	buf := bytes.NewBufferString(c.HexTx)
	transaction := tx.Tx{}
	err := transaction.Unserialize(buf)
	if err != nil {
		return nil, rpcDecodeHexError(c.HexTx)
	}

	hash := transaction.TxHash()

	maxTxFee := 10000 // todo define this global variable
	maxRawTxFee := maxTxFee
	if c.AllowHighFees != nil && *c.AllowHighFees {
		maxRawTxFee = 0
	}

	view := utxo.GetUtxoCacheInstance()
	var haveChain bool
	for i := 0; !haveChain && i < transaction.GetOutsCount(); i++ {
		existingCoin, _ := view.GetCoin(outpoint.NewOutPoint(hash, uint32(i)))
		haveChain = !existingCoin.IsSpent()
	}

	entry := mempool.Gpool.FindTx(hash)
	if entry != nil {
		s.Handler.ProcessForRpc(transaction)
	}

	// todo here

	return hash.String(), nil
}

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
