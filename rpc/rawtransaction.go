package rpc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/copernet/copernicus/model/wallet"
	"gopkg.in/fatih/set.v0"
	"math"
	"strconv"

	"github.com/copernet/copernicus/crypto"
	"github.com/copernet/copernicus/errcode"
	"github.com/copernet/copernicus/log"
	"github.com/copernet/copernicus/logic/lmempool"
	"github.com/copernet/copernicus/logic/lmerkleblock"
	"github.com/copernet/copernicus/logic/ltx"
	"github.com/copernet/copernicus/logic/lutxo"
	"github.com/copernet/copernicus/logic/lwallet"
	"github.com/copernet/copernicus/model/blockindex"
	"github.com/copernet/copernicus/model/chain"
	"github.com/copernet/copernicus/model/mempool"
	"github.com/copernet/copernicus/model/opcodes"
	"github.com/copernet/copernicus/model/outpoint"
	"github.com/copernet/copernicus/model/script"
	"github.com/copernet/copernicus/model/tx"
	"github.com/copernet/copernicus/model/txin"
	"github.com/copernet/copernicus/model/txout"
	"github.com/copernet/copernicus/model/utxo"
	"github.com/copernet/copernicus/net/server"
	"github.com/copernet/copernicus/net/wire"
	"github.com/copernet/copernicus/persist/disk"
	"github.com/copernet/copernicus/rpc/btcjson"
	"github.com/copernet/copernicus/util"
	"github.com/copernet/copernicus/util/amount"
	"github.com/copernet/copernicus/util/cashaddr"
)

var rawTransactionHandlers = map[string]commandHandler{
	"getrawtransaction":    handleGetRawTransaction,    // complete
	"createrawtransaction": handleCreateRawTransaction, // complete
	"decoderawtransaction": handleDecodeRawTransaction, // complete
	"decodescript":         handleDecodeScript,         // complete
	"sendrawtransaction":   handleSendRawTransaction,   // complete
	"signrawtransaction":   handleSignRawTransaction,   // partial complete
	"gettxoutproof":        handleGetTxoutProof,        // complete
	"verifytxoutproof":     handleVerifyTxoutProof,     // complete
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
		verbose = *c.Verbose
	}

	tx, hashBlock, ok := GetTransaction(txHash, true)
	if !ok {
		return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidAddressOrKey,
			"No such mempool or blockchain transaction. Use gettransaction for wallet transactions.")
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

	rawTxn, rpcErr := getTxRawResult(tx, hashBlock, strHex)
	if rpcErr != nil {
		return nil, rpcErr
	}
	return rawTxn, nil
}

// getTxRawResult converts the passed transaction and associated parameters
// to a raw transaction JSON object.
func getTxRawResult(tx *tx.Tx, hashBlock *util.Hash, strHex string) (*btcjson.TxRawResult, *btcjson.RPCError) {
	hash := tx.GetHash()
	txReply := &btcjson.TxRawResult{
		Hex:      strHex,
		TxID:     hash.String(),
		Hash:     hash.String(),
		Size:     int(tx.SerializeSize()),
		Version:  tx.GetVersion(),
		LockTime: tx.GetLockTime(),
		Vin:      getVinList(tx),
		Vout:     getVoutList(tx),
	}

	if hashBlock != nil && !hashBlock.IsNull() {
		txReply.BlockHash = hashBlock.String()
		bindex := chain.GetInstance().FindBlockIndex(*hashBlock)
		if bindex != nil {
			if chain.GetInstance().Contains(bindex) {
				txReply.Confirmations = chain.GetInstance().TipHeight() - bindex.Height + 1
				txReply.Time = bindex.Header.Time
				txReply.Blocktime = bindex.Header.Time
			} else {
				txReply.Confirmations = 0
			}
		}
	}
	return txReply, nil
}

// getVinList returns a slice of JSON objects for the inputs of the passed transaction.
func getVinList(tx *tx.Tx) []btcjson.Vin {
	vinList := make([]btcjson.Vin, len(tx.GetIns()))
	for i, in := range tx.GetIns() {
		if tx.IsCoinBase() {
			vinList[i] = btcjson.Vin{
				Coinbase: hex.EncodeToString(in.GetScriptSig().GetData()),
				Sequence: in.Sequence,
			}
		} else {
			vinList[i] = btcjson.Vin{
				Txid: in.PreviousOutPoint.Hash.String(),
				Vout: in.PreviousOutPoint.Index,
				ScriptSig: &btcjson.ScriptSig{
					Asm: ScriptToAsmStr(in.GetScriptSig(), true),
					Hex: hex.EncodeToString(in.GetScriptSig().GetData()),
				},
				Sequence: in.Sequence,
			}
		}
	}
	return vinList
}

// getVoutList returns a slice of JSON objects for the outputs of the passed transaction.
func getVoutList(tx *tx.Tx) []btcjson.Vout {
	voutList := make([]btcjson.Vout, tx.GetOutsCount())
	for i := 0; i < tx.GetOutsCount(); i++ {
		out := tx.GetTxOut(i)
		scriptPubKeyJSON := ScriptPubKeyToJSON(out.GetScriptPubKey(), true)
		voutList[i] = btcjson.Vout{
			Value:        valueFromAmount(int64(out.GetValue())),
			N:            uint32(i),
			ScriptPubKey: *scriptPubKeyJSON,
		}
	}
	return voutList
}

func ScriptToAsmStr(s *script.Script, attemptSighashDecode bool) string {
	var str string
	for _, scriptOpcodes := range s.ParsedOpCodes {
		if len(str) > 0 {
			str += " "
		}
		opcode := scriptOpcodes.OpValue
		vch := make([]byte, len(scriptOpcodes.Data))
		copy(vch, scriptOpcodes.Data)

		if opcode >= 0 && opcode <= opcodes.OP_PUSHDATA4 {
			if len(vch) <= 4 {
				num, _ := script.GetScriptNum(vch, false, script.DefaultMaxNumSize)
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
					if vch[len(vch)-1]&crypto.SigHashForkID != 0 {
						// If the transaction is using SIGHASH_FORKID, we need
						// to set the appropriate flag.
						// TODO: Remove after the Hard Fork.
						flags |= script.ScriptEnableSigHashForkID
					}
					err := script.CheckSignatureEncoding(vch, uint32(flags))
					if err == nil {
						sigHashType := int(vch[len(vch)-1])
						for desc, hashType := range mapSigHashValues {
							if hashType == sigHashType {
								strSigHashDecode = "[" + desc + "]"
								// remove the sighash type byte. it will be replaced
								// by the decode.
								vch = vch[:len(vch)-1]
								break
							}
						}
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

func ScriptPubKeyToJSON(script *script.Script, includeHex bool) *btcjson.ScriptPubKeyResult {
	result := &btcjson.ScriptPubKeyResult{}

	if script == nil {
		return result
	}

	result.Asm = ScriptToAsmStr(script, false)
	if includeHex {
		result.Hex = hex.EncodeToString(script.GetData())
	}

	t, addresses, required, err := script.ExtractDestinations()
	result.Type = GetTxnOutputType(t)

	if err != nil {
		return result
	}

	result.ReqSigs = int32(required)
	result.Addresses = make([]string, 0, len(addresses))
	for _, address := range addresses {
		result.Addresses = append(result.Addresses, address.String())
	}

	return result
}

func GetTxnOutputType(sType int) string {
	switch sType {
	case script.ScriptNonStandard:
		return "nonstandard"
	case script.ScriptPubkey:
		return "pubkey"
	case script.ScriptPubkeyHash:
		return "pubkeyhash"
	case script.ScriptHash:
		return "scripthash"
	case script.ScriptMultiSig:
		return "multisig"
	case script.ScriptNullData:
		return "nulldata"
	default:
		return "unknown"
	}
}

func GetTransaction(hash *util.Hash, allowSlow bool) (*tx.Tx, *util.Hash, bool) {
	entry := mempool.GetInstance().FindTx(*hash)
	if entry != nil {
		return entry.Tx, nil, true
	}

	/* TODO: NOT support txindex yet
	if chain.GTxIndex {
		chain.GBlockTree.ReadTxIndex(hash)
	}*/

	if !allowSlow {
		return nil, nil, false
	}

	// use coin database to locate block that contains transaction, and scan it
	var indexSlow *blockindex.BlockIndex
	coin := lutxo.AccessByTxid(utxo.GetUtxoCacheInstance(), hash)
	if coin == nil || coin.IsSpent() {
		return nil, nil, false
	}

	indexSlow = chain.GetInstance().GetIndex(coin.GetHeight())
	if indexSlow != nil {
		if bk, ok := disk.ReadBlockFromDisk(indexSlow, chain.GetInstance().GetParams()); ok {
			for _, item := range bk.Txs {
				if *hash == item.GetHash() {
					return item, indexSlow.GetBlockHash(), true
				}
			}
		}
	}
	return nil, nil, false
}

func handleCreateRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.CreateRawTransactionCmd)

	lockTime := uint32(0)
	if c.LockTime != nil && (*c.LockTime < 0 || *c.LockTime > int64(script.SequenceFinal)) {
		return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidParameter, "LockTime out of range")
	}
	transaction := tx.NewTx(lockTime, tx.DefaultVersion)

	for _, input := range c.Inputs {
		txIn, rpcErr := createRawTxInput(&input, lockTime)
		if rpcErr != nil {
			return nil, rpcErr
		}
		transaction.AddTxIn(txIn)
	}

	for address, cost := range c.Outputs {
		txOut, rpcErr := createRawTxOutput(address, cost)
		if rpcErr != nil {
			return nil, rpcErr
		}
		transaction.AddTxOut(txOut)
	}

	buf := bytes.NewBuffer(nil)
	err := transaction.Serialize(buf)
	if err != nil {
		log.Error("rawTransaction:serialize tx failed: %v", err)
		return "", btcjson.ErrRPCInternal
	}

	return hex.EncodeToString(buf.Bytes()), nil
}

func createRawTxInput(input *btcjson.TransactionInput, lockTime uint32) (*txin.TxIn, *btcjson.RPCError) {
	hash, err := util.GetHashFromStr(input.Txid)
	if err != nil {
		return nil, rpcDecodeHexError(input.Txid)
	}

	if input.Vout < 0 {
		return nil, btcjson.NewRPCError(btcjson.ErrInvalidParameter,
			"Invalid parameter, vout must be positive")
	}

	sequence := uint32(math.MaxUint32)
	if input.Sequence != nil {
		if *input.Sequence < 0 || *input.Sequence > math.MaxUint32 {
			return nil, btcjson.NewRPCError(btcjson.ErrInvalidParameter,
				"Invalid parameter, sequence number is out of range")
		}
		sequence = uint32(*input.Sequence)
	} else if lockTime != 0 {
		sequence = math.MaxUint32 - 1
	}

	txIn := txin.NewTxIn(outpoint.NewOutPoint(*hash, input.Vout), script.NewEmptyScript(), sequence)
	return txIn, nil
}

func createRawTxOutput(address string, cost btcjson.AmountType) (*txout.TxOut, *btcjson.RPCError) {
	var nullData []byte
	var rpcErr *btcjson.RPCError
	txAmount := amount.Amount(0)

	if address == "data" {
		data, ok := cost.(string)
		if !ok {
			return nil, btcjson.NewRPCError(btcjson.ErrRPCType, "Data is not a string")
		}
		var err error
		if nullData, err = hex.DecodeString(data); err != nil {
			return nil, rpcDecodeHexError(data)
		}
	} else {
		if txAmount, rpcErr = amountFromValue(cost); rpcErr != nil {
			return nil, rpcErr
		}
	}

	scriptPubKey, rpcErr := getStandardScriptPubKey(address, nullData)
	if rpcErr != nil {
		return nil, rpcErr
	}

	txOut := txout.NewTxOut(txAmount, scriptPubKey)
	return txOut, nil
}

func amountFromValue(amountParam btcjson.AmountType) (amount.Amount, *btcjson.RPCError) {
	amountVal, ok := amountParam.(float64)
	if !ok {
		amountValStr, ok := amountParam.(string)
		if !ok {
			return 0, btcjson.NewRPCError(btcjson.ErrRPCType, "Amount is not a number or string")
		}
		var err error
		if amountVal, err = strconv.ParseFloat(amountValStr, 64); err != nil {
			return 0, btcjson.NewRPCError(btcjson.ErrRPCType, "Invalid amount")
		}
	}
	amt, err := amount.NewAmount(amountVal)
	if err != nil || !amount.MoneyRange(amt) {
		return 0, btcjson.NewRPCError(btcjson.ErrRPCType, "Amount out of range")
	}
	return amt, nil
}

func decodeAddress(address string) (cashaddr.AddressType, []byte, *btcjson.RPCError) {
	var keyHash []byte
	var addrType cashaddr.AddressType

	if legacyAddr, err := script.AddressFromString(address); err == nil {
		switch legacyAddr.GetVersion() {
		case script.AddressVerPubKey():
			addrType = cashaddr.P2PKH
		case script.AddressVerScript():
			addrType = cashaddr.P2SH
		}
		keyHash = legacyAddr.EncodeToPubKeyHash()

	} else {
		// try cash address
		var err error
		if keyHash, _, addrType, err = cashaddr.CheckDecodeCashAddress(address); err != nil {
			return addrType, nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidAddressOrKey,
				"Invalid Bitcoin address: "+address)
		}
	}

	return addrType, keyHash, nil
}

func generateScript(data ...interface{}) (*script.Script, error) {
	sc := script.NewEmptyScript()
	for _, item := range data {
		switch item.(type) {
		case int:
			if err := sc.PushOpCode(item.(int)); err != nil {
				return nil, err
			}
		case []byte:
			if err := sc.PushSingleData(item.([]byte)); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("push unknown type")
		}
	}
	return sc, nil
}

func getStandardScriptPubKey(address string, nullData []byte) (*script.Script, *btcjson.RPCError) {
	if nullData != nil {
		// NullData
		scriptPubKey, err := generateScript(opcodes.OP_RETURN, nullData)
		if err != nil {
			log.Error("generateScript error:%s", err.Error())
			return nil, btcjson.ErrRPCInternal
		}
		return scriptPubKey, nil
	}

	addrType, keyHash, rpcErr := decodeAddress(address)
	if rpcErr != nil {
		return nil, rpcErr
	}

	if addrType == cashaddr.P2PKH {
		scriptPubKey, err := generateScript(opcodes.OP_DUP, opcodes.OP_HASH160, keyHash,
			opcodes.OP_EQUALVERIFY, opcodes.OP_CHECKSIG)
		if err != nil {
			log.Error("generateScript error:%s", err.Error())
			return nil, btcjson.ErrRPCInternal
		}
		return scriptPubKey, nil

	} else if addrType == cashaddr.P2SH {
		scriptPubKey, err := generateScript(opcodes.OP_HASH160, keyHash, opcodes.OP_EQUAL)
		if err != nil {
			log.Error("generateScript error:%s", err.Error())
			return nil, btcjson.ErrRPCInternal
		}
		return scriptPubKey, nil
	}

	return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidAddressOrKey, "Invalid Bitcoin address: "+address)
}

func handleDecodeRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.DecodeRawTransactionCmd)

	transaction := tx.NewEmptyTx()

	// Unserialize the transaction.
	serializedTx, err := hex.DecodeString(c.HexTx)
	if err == nil {
		err = transaction.Unserialize(bytes.NewReader(serializedTx))
	}
	if err != nil || int(transaction.SerializeSize()) != len(serializedTx) {
		return nil, btcjson.NewRPCError(btcjson.ErrRPCDeserialization, "TX decode failed")
	}

	txHash := transaction.GetHash()

	// Create and return the result.
	txReply := &btcjson.TxRawDecodeResult{
		Txid:     txHash.String(),
		Hash:     txHash.String(),
		Size:     transaction.SerializeSize(),
		Version:  transaction.GetVersion(),
		Locktime: transaction.GetLockTime(),
		Vin:      getVinList(transaction),
		Vout:     getVoutList(transaction),
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
		addr, err := script.AddressFromScriptHash(scriptByte)
		if err == nil {
			ret.P2SH = addr.String()
		}
	}

	return ret, nil
}

func handleSendRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SendRawTransactionCmd)

	b, _ := hex.DecodeString(c.HexTx)
	buf := bytes.NewBuffer(b)

	txn := tx.Tx{}
	err := txn.Unserialize(buf)
	if err != nil {
		return nil, rpcDecodeHexError(c.HexTx)
	}

	hash := txn.GetHash()

	// NOT support high fee limit yet
	//maxRawTxFee := mining.MaxTxFee
	//if c.AllowHighFees != nil && *c.AllowHighFees {
	//	maxRawTxFee = 0
	//}

	view := utxo.GetUtxoCacheInstance()
	var inChain bool
	for i := 0; !inChain && i < txn.GetOutsCount(); i++ {
		existingCoin := view.GetCoin(outpoint.NewOutPoint(hash, uint32(i)))
		inChain = existingCoin != nil && !existingCoin.IsSpent()
	}

	entry := mempool.GetInstance().FindTx(hash)

	if entry == nil && !inChain {
		err = lmempool.AcceptTxToMemPool(&txn)
		if err != nil {
			return nil, rpcErrorOfAcceptTx(err)
		}
	} else if inChain {
		return nil, btcjson.NewRPCError(btcjson.RPCTransactionAlreadyInChain,
			"transaction already in block chain")
	}

	txInvMsg := wire.NewInvVect(wire.InvTypeTx, &hash)
	_, err = server.ProcessForRPC(txInvMsg)
	if err != nil {
		log.Info("handleSendRawTransaction process InvTypeTx msg error:%s", err.Error())
		return nil, btcjson.ErrRPCInternal
	}

	return hash.String(), nil
}

func rpcErrorOfAcceptTx(err error) *btcjson.RPCError {
	missingInputs := errcode.IsErrorCode(err, errcode.TxErrNoPreviousOut)
	if missingInputs {
		return btcjson.NewRPCError(btcjson.RPCTransactionError, "Missing inputs")
	}

	_, _, isReject := errcode.IsRejectCode(err)
	if isReject {
		return btcjson.NewRPCError(btcjson.RPCTransactionRejected, err.Error())

	}

	return btcjson.NewRPCError(btcjson.ErrUnDefined, err.Error())
}

var mapSigHashValues = map[string]int{
	"ALL":                        crypto.SigHashAll,
	"ALL|ANYONECANPAY":           crypto.SigHashAll | crypto.SigHashAnyoneCanpay,
	"ALL|FORKID":                 crypto.SigHashAll | crypto.SigHashForkID,
	"ALL|FORKID|ANYONECANPAY":    crypto.SigHashAll | crypto.SigHashForkID | crypto.SigHashAnyoneCanpay,
	"NONE":                       crypto.SigHashNone,
	"NONE|ANYONECANPAY":          crypto.SigHashNone | crypto.SigHashAnyoneCanpay,
	"NONE|FORKID":                crypto.SigHashNone | crypto.SigHashForkID,
	"NONE|FORKID|ANYONECANPAY":   crypto.SigHashNone | crypto.SigHashForkID | crypto.SigHashAnyoneCanpay,
	"SINGLE":                     crypto.SigHashSingle,
	"SINGLE|ANYONECANPAY":        crypto.SigHashSingle | crypto.SigHashAnyoneCanpay,
	"SINGLE|FORKID":              crypto.SigHashSingle | crypto.SigHashForkID,
	"SINGLE|FORKID|ANYONECANPAY": crypto.SigHashSingle | crypto.SigHashForkID | crypto.SigHashAnyoneCanpay,
}

func isCoinSpent(coin *utxo.Coin, out *outpoint.OutPoint) bool {
	if coin == nil {
		return false
	}
	if coin.IsMempoolCoin() {
		return mempool.GetInstance().HasSpentOut(out)
	}
	return coin.IsSpent()
}

func handleSignRawTransaction(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.SignRawTransactionCmd)

	txData, err := hex.DecodeString(c.HexTx)
	if err != nil {
		return nil, rpcDecodeHexError(c.HexTx)
	}

	txVariants := make([]*tx.Tx, 0)
	totalSerializeSize := 0
	for totalSerializeSize < len(txData) {
		transaction := tx.NewEmptyTx()
		err = transaction.Unserialize(bytes.NewReader(txData[totalSerializeSize:]))
		if err != nil {
			return nil, btcjson.NewRPCError(btcjson.ErrRPCDeserialization, "TX decode failed: "+err.Error())
		}
		txVariants = append(txVariants, transaction)
		totalSerializeSize += int(transaction.SerializeSize())
	}

	mergedTx := txVariants[0]
	coinsMap, redeemScripts, rpcErr := getCoins(mergedTx.GetIns(), c.PrevTxs)
	if rpcErr != nil {
		return nil, rpcErr
	}

	keyStore, rpcErr := getKeys(c.PrivKeys, coinsMap, redeemScripts)
	if rpcErr != nil {
		return nil, rpcErr
	}

	hashType := crypto.SigHashAll | crypto.SigHashForkID
	if c.SigHashType != nil {
		var ok bool
		if hashType, ok = mapSigHashValues[*c.SigHashType]; !ok {
			return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidParameter, "Invalid sighash param")
		}
		if hashType&crypto.SigHashForkID == 0 {
			return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidParameter, "Signature must use SIGHASH_FORKID")
		}
	}

	signErrors := ltx.SignRawTransaction(txVariants, redeemScripts, keyStore, coinsMap, uint32(hashType))
	errors := make([]*btcjson.SignRawTransactionError, 0, len(signErrors))
	for _, signErr := range signErrors {
		errors = append(errors, TxInErrorToJSON(signErr.TxIn, signErr.ErrMsg))
	}

	complete := len(errors) == 0
	buf := bytes.NewBuffer(nil)
	err = mergedTx.Serialize(buf)
	if err != nil {
		log.Error("rawTransaction:serialize transaction failed: %v", err)
		return nil, btcjson.ErrRPCInternal
	}
	return &btcjson.SignRawTransactionResult{
		Hex:      hex.EncodeToString(buf.Bytes()),
		Complete: complete,
		Errors:   errors,
	}, err
}

func getCoins(txIns []*txin.TxIn, prevTxs *[]btcjson.RawTxInput) (*utxo.CoinsMap,
	map[outpoint.OutPoint]*script.Script, *btcjson.RPCError) {
	coinsMap := utxo.NewEmptyCoinsMap()
	for _, in := range txIns {
		// fetch from mempool
		coin := mempool.GetInstance().GetCoin(in.PreviousOutPoint)
		if coin != nil {
			coinsMap.AddCoin(in.PreviousOutPoint, coin, false)
		} else {
			// fetch from utxo
			coinsMap.FetchCoin(in.PreviousOutPoint)
		}
	}

	//The second optional argument (may be null) is an array of previous transaction outputs that
	//this transaction depends on but may not yet be in the block chain.
	if prevTxs == nil {
		return coinsMap, nil, nil
	}
	redeemScripts := make(map[outpoint.OutPoint]*script.Script)
	for _, prevTx := range *prevTxs {
		hash, err := util.GetHashFromStr(prevTx.Txid)
		if err != nil {
			return nil, nil, rpcDecodeHexError(prevTx.Txid)
		}
		if prevTx.Vout < 0 {
			return nil, nil, btcjson.NewRPCError(btcjson.RPCDeserializationError, "vout must be positive")
		}
		out := outpoint.NewOutPoint(*hash, prevTx.Vout)

		scriptPubKeyBuf, err := hex.DecodeString(prevTx.ScriptPubKey)
		if err != nil {
			return nil, nil, rpcDecodeHexError(prevTx.ScriptPubKey)
		}
		scriptPubKey := script.NewScriptRaw(scriptPubKeyBuf)

		coin := coinsMap.GetCoin(out)
		if coin != nil && !isCoinSpent(coin, out) && !coin.GetScriptPubKey().IsEqual(scriptPubKey) {
			coinPrevScript := ScriptToAsmStr(coin.GetScriptPubKey(), false)
			inputPrevScript := ScriptToAsmStr(scriptPubKey, false)
			return nil, nil, btcjson.NewRPCError(btcjson.RPCDeserializationError,
				"Previous output scriptPubKey mismatch:\n"+coinPrevScript+"\nvs:\n"+inputPrevScript)
		}
		outAmount, rpcErr := amountFromValue(prevTx.Amount)
		if rpcErr != nil {
			return nil, nil, rpcErr
		}
		txOut := txout.NewTxOut(outAmount, scriptPubKey)
		coin = utxo.NewFreshCoin(txOut, 1, false)
		coinsMap.AddCoin(out, coin, true)

		if prevTx.RedeemScript != nil {
			redeemScriptData, err := hex.DecodeString(*prevTx.RedeemScript)
			if err != nil {
				return nil, nil, rpcDecodeHexError(*prevTx.RedeemScript)
			}
			redeemScripts[*out] = script.NewScriptRaw(redeemScriptData)
		} else {
			if scriptPubKey.Size() == 23 {
				keyHash := scriptPubKey.GetData()[2:22]
				if redeem := wallet.GetInstance().GetScript(keyHash); redeem != nil {
					redeemScripts[*out] = redeem
				}
			}
		}
	}
	return coinsMap, redeemScripts, nil
}

func getPubKeyHash(scriptPubKey *script.Script) [][]byte {
	pubKeyHash := make([][]byte, 0)
	pubKeyType, pubKeys, _ := scriptPubKey.IsStandardScriptPubKey()

	if pubKeyType == script.ScriptPubkey {
		pubKeyHash = append(pubKeyHash, util.Hash160(pubKeys[0]))

	} else if pubKeyType == script.ScriptPubkeyHash {
		pubKeyHash = append(pubKeyHash, pubKeys[0])

	} else if pubKeyType == script.ScriptMultiSig {
		for _, pubKey := range pubKeys[1:] {
			if len(pubKey) >= 32 {
				pubKeyHash = append(pubKeyHash, util.Hash160(pubKey))
			}
		}
	}
	return pubKeyHash
}

func getKeys(privateKeys *[]string, coinsMap *utxo.CoinsMap,
	redeemScripts map[outpoint.OutPoint]*script.Script) (*crypto.KeyStore, *btcjson.RPCError) {

	keyStore := crypto.NewKeyStore()
	if privateKeys != nil {
		for _, key := range *privateKeys {
			privateKey, err := crypto.DecodePrivateKey(key)
			if err != nil {
				return nil, btcjson.NewRPCError(btcjson.RPCInvalidAddressOrKey, "Invalid private key")
			}
			keyStore.AddKey(privateKey)
		}
	} else if lwallet.IsWalletEnable() {
		pubKeyHashList := make([][]byte, 0)
		for _, coin := range coinsMap.GetMap() {
			pubKeyHash := getPubKeyHash(coin.GetScriptPubKey())
			pubKeyHashList = append(pubKeyHashList, pubKeyHash...)
		}
		for _, redeemScript := range redeemScripts {
			pubKeyHash := getPubKeyHash(redeemScript)
			pubKeyHashList = append(pubKeyHashList, pubKeyHash...)
		}
		keyPairs := lwallet.GetKeyPairs(pubKeyHashList)
		keyStore.AddKeyPairs(keyPairs)
	}
	return keyStore, nil
}

func TxInErrorToJSON(in *txin.TxIn, errorMessage string) *btcjson.SignRawTransactionError {
	return &btcjson.SignRawTransactionError{
		TxID:      in.PreviousOutPoint.Hash.String(),
		Vout:      in.PreviousOutPoint.Index,
		ScriptSig: hex.EncodeToString(in.GetScriptSig().GetData()),
		Sequence:  in.Sequence,
		Error:     errorMessage,
	}
}

func handleGetTxoutProof(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.GetTxOutProofCmd)

	setTxIds := set.New()
	var oneTxID util.Hash
	txIds := c.TxIDs

	for _, txID := range txIds {
		hash, err := util.GetHashFromStr(txID)
		if len(txID) != 64 || err != nil {
			return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidParameter,
				"Invalid txid "+txID)
		}
		if setTxIds.Has(*hash) {
			return nil, btcjson.NewRPCError(btcjson.ErrRPCInvalidParameter,
				"Invalid parameter, duplicated txid: "+txID)
		}
		setTxIds.Add(*hash)
		oneTxID = *hash
	}

	var bindex *blockindex.BlockIndex
	var hashBlock *util.Hash
	if c.BlockHash != nil {
		var err error
		hashBlock, err = util.GetHashFromStr(*c.BlockHash)
		if err != nil {
			return nil, rpcDecodeHexError(*c.BlockHash)
		}

		bindex = chain.GetInstance().FindBlockIndex(*hashBlock)
		if bindex == nil {
			return nil, btcjson.NewRPCError(btcjson.RPCInvalidAddressOrKey, "Block not found")
		}
	} else {
		view := utxo.GetUtxoCacheInstance()
		coin := lutxo.AccessByTxid(view, &oneTxID)
		if coin != nil && !coin.IsSpent() && coin.GetHeight() > 0 &&
			coin.GetHeight() <= chain.GetInstance().Height() {
			bindex = chain.GetInstance().GetIndex(coin.GetHeight())
		}

		if bindex == nil {
			_, hashBlock, ok := GetTransaction(&oneTxID, false)
			if !ok || hashBlock == nil {
				return nil, btcjson.NewRPCError(btcjson.RPCInvalidAddressOrKey, "Transaction not yet in block")
			}
			bindex = chain.GetInstance().FindBlockIndex(*hashBlock)
			if bindex == nil {
				return nil, btcjson.NewRPCError(btcjson.RPCInternalError, "Transaction index corrupt")
			}
		}
	}

	bk, ok := disk.ReadBlockFromDisk(bindex, chain.GetInstance().GetParams())
	if !ok {
		return nil, btcjson.NewRPCError(btcjson.RPCInternalError, "Can not read block from disk")
	}

	found := 0
	for _, transaction := range bk.Txs {
		if setTxIds.Has(transaction.GetHash()) {
			found++
		}
	}

	if found != setTxIds.Size() {
		return nil, btcjson.NewRPCError(btcjson.RPCInvalidAddressOrKey,
			"(Not all) transactions not found in specified block")
	}

	mb := lmerkleblock.NewMerkleBlock(bk, setTxIds)
	buf := bytes.NewBuffer(nil)
	mb.Serialize(buf)
	return hex.EncodeToString(buf.Bytes()), nil
}

func handleVerifyTxoutProof(s *Server, cmd interface{}, closeChan <-chan struct{}) (interface{}, error) {
	c := cmd.(*btcjson.VerifyTxOutProofCmd)

	b, err := hex.DecodeString(c.Proof)
	if err != nil {
		return nil, rpcDecodeHexError(c.Proof)
	}

	mb := &lmerkleblock.MerkleBlock{}
	err = mb.Unserialize(bytes.NewReader(b))
	if err != nil {
		return nil, btcjson.NewRPCError(btcjson.RPCDeserializationError, "MerkleBlock Unserialize error")
	}

	matches := make([]util.Hash, 0)
	items := make([]int, 0)
	if !mb.Txn.ExtractMatches(&matches, &items).IsEqual(&mb.Header.MerkleRoot) {
		return nil, nil
	}

	bindex := chain.GetInstance().FindBlockIndex(mb.Header.GetHash())
	if bindex == nil || !chain.GetInstance().Contains(bindex) {
		return nil, btcjson.NewRPCError(btcjson.RPCInvalidAddressOrKey, "Block not found in chain")
	}

	ret := make([]string, 0, len(matches))
	for _, hash := range matches {
		ret = append(ret, hash.String())
	}
	return ret, nil
}

func registerRawTransactionRPCCommands() {
	for name, handler := range rawTransactionHandlers {
		appendCommand(name, handler)
	}
}
